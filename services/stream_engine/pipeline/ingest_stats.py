"""Ingest stats sidecar: lightweight Kafka consumer thread that counts messages per topic.

Replaces the old Pathway groupby+count approach. Runs as a separate thread with its
own consumer group, subscribes to all input topics, and counts messages per topic.
Writes counts to Redis Hash for the frontend to display.

This is completely decoupled from the Pathway pipeline — no GIL contention with
the main processing path.
"""

from __future__ import annotations

import threading
import time
from typing import Any

from confluent_kafka import Consumer, KafkaError, KafkaException

from config import settings
from storage.redis_cache import get_redis
from utils.logger import get_logger

log = get_logger(__name__)

REDIS_KEY = "stream:ingest:counters"

TOPIC_TO_PREFIX: dict[str, str] = {
    settings.kafka.topic_report: "mt:RCV_REPORT",
    settings.kafka.topic_fenbi: "mt:RCV_FENBIDATA",
    settings.kafka.topic_mkttbl: "mt:RCV_MKTTBLDATA",
    settings.kafka.topic_finance: "mt:RCV_FINANCEDATA",
    settings.kafka.topic_filedata_history: "mt:FILE_HISTORY_EX",
    settings.kafka.topic_filedata_minute: "mt:FILE_MINUTE_EX",
    settings.kafka.topic_filedata_5minute: "mt:FILE_5MINUTE_EX",
    settings.kafka.topic_filedata_power: "mt:FILE_POWER_EX",
}

_STARTED_AT_MS: int = int(time.time() * 1000)
_FLUSH_INTERVAL_SECONDS: float = 1.0


class IngestStatsWorker:
    """Background thread that counts Kafka messages per topic and flushes to Redis."""

    def __init__(self) -> None:
        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None
        self._counts: dict[str, int] = {}
        self._lock = threading.Lock()

    def start(self) -> None:
        self._thread = threading.Thread(
            target=self._run, name="ingest-stats", daemon=True
        )
        self._thread.start()
        log.info("ingest stats sidecar started")

    def stop(self) -> None:
        self._stop_event.set()
        if self._thread is not None:
            self._thread.join(timeout=3.0)

    def _run(self) -> None:
        try:
            client = get_redis()
            client.hset(REDIS_KEY, "__started_at_ms", str(_STARTED_AT_MS))
        except Exception:
            log.exception("ingest_stats init started_at failed")

        topics = list(TOPIC_TO_PREFIX.keys())
        conf = {
            "bootstrap.servers": settings.kafka.bootstrap_servers,
            "group.id": f"{settings.kafka.group_id}-stats",
            "client.id": f"{settings.kafka.client_id}-stats",
            "auto.offset.reset": settings.kafka.auto_offset_reset,
            "enable.auto.commit": "true",
        }
        consumer = Consumer(conf)
        consumer.subscribe(topics)
        log.info("ingest stats consumer subscribed", topics=topics)

        last_flush = time.monotonic()
        dirty = False

        try:
            while not self._stop_event.is_set():
                msg = consumer.poll(0.5)
                if msg is None:
                    if dirty and (time.monotonic() - last_flush) >= _FLUSH_INTERVAL_SECONDS:
                        self._flush_to_redis()
                        last_flush = time.monotonic()
                        dirty = False
                    continue

                err = msg.error()
                if err is not None:
                    if err.code() == KafkaError._PARTITION_EOF:
                        continue
                    log.warning("ingest stats consumer error", error=str(err))
                    continue

                topic = msg.topic()
                field = TOPIC_TO_PREFIX.get(topic, f"mt:{topic}")

                with self._lock:
                    self._counts[field] = self._counts.get(field, 0) + 1

                dirty = True
                if (time.monotonic() - last_flush) >= _FLUSH_INTERVAL_SECONDS:
                    self._flush_to_redis()
                    last_flush = time.monotonic()
                    dirty = False
        finally:
            try:
                consumer.close()
            except Exception:
                pass
            log.info("ingest stats consumer closed")

    def _flush_to_redis(self) -> None:
        """Batch write all counters to Redis Hash."""
        with self._lock:
            snapshot = dict(self._counts)
        if not snapshot:
            return
        try:
            client = get_redis()
            pipe = client.pipeline()
            for field, count in snapshot.items():
                pipe.hset(REDIS_KEY, field, str(count))
            pipe.hset(REDIS_KEY, "__updated_at_ms", str(int(time.time() * 1000)))
            pipe.execute()
        except Exception:
            log.exception("ingest_stats flush to redis failed")


_worker: IngestStatsWorker | None = None


def start_ingest_stats() -> None:
    """Start the ingest stats sidecar thread. Call once from main."""
    global _worker
    if _worker is not None:
        return
    _worker = IngestStatsWorker()
    _worker.start()


def stop_ingest_stats() -> None:
    """Stop the ingest stats sidecar thread."""
    global _worker
    if _worker is not None:
        _worker.stop()
        _worker = None
