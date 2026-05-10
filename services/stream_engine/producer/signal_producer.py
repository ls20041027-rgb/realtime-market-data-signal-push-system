"""Kafka 生产者封装（简化版：去掉内存 retry_queue）。

本次简化动机：
  1. 原 ``_retry_queue`` + ``_enqueue_retry`` + ``_drain_retry_queue`` +
     ``retry_queue_size`` 这一套内存失败重放没有持久化，进程挂了仍丢消息，
     毕设层级的"最多重放一次 BufferError"用 ``flush+再 produce`` 即可；
  2. 懒初始化双检锁简化成 ``__init__`` 里直接 new Producer；
  3. delivery callback 改静态函数，失败就打 log。

对外保留的三个 publish 方法与 envelope 约定（yaml + R5）不变。
"""

from __future__ import annotations

import json
import threading
import uuid
from datetime import datetime, timezone
from decimal import Decimal
from typing import Any, Mapping

from confluent_kafka import KafkaError, Message, Producer

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)

SOURCE: str = "stream_engine"

MT_MARKET_NORMALIZED: str = "MARKET_SNAPSHOT_NORMALIZED"
MT_TRADING_SIGNAL: str = "TRADING_SIGNAL"

LEVEL_TO_MESSAGE_TYPE: dict[str, str] = {
    "info": "SERVICE_INFO",
    "warning": "SERVICE_WARNING",
    "error": "SERVICE_ERROR",
    "critical": "SERVICE_CRITICAL",
}

SYSTEM_SYMBOL_PLACEHOLDER: str = "SYSTEM"


def now_iso8601() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def json_default(obj: Any) -> str:
    """Decimal / datetime 等 → str，保精度（R2）。"""
    if isinstance(obj, Decimal):
        return str(obj)
    return str(obj)


def delivery_cb(err: KafkaError, msg: Message) -> None:
    """静态 delivery callback：失败打 error，成功走 debug（吞 debug 消耗低）。"""
    if err is not None:
        log.error(
            "kafka delivery failed",
            topic=msg.topic() if msg else None,
            error=str(err),
        )
        return
    log.debug(
        "kafka delivery ok",
        topic=msg.topic(),
        partition=msg.partition(),
        offset=msg.offset(),
    )


class StreamProducer:
    """线程安全的 Kafka 生产者门面。"""

    def __init__(self) -> None:
        cfg = {
            "bootstrap.servers": settings.kafka.bootstrap_servers,
            "client.id": settings.kafka.client_id,
            "acks": settings.kafka.producer_acks,
            "linger.ms": settings.kafka.producer_linger_ms,
            "enable.idempotence": False,
        }
        self._producer: Producer = Producer(cfg)
        log.info(
            "kafka producer initialized",
            bootstrap_servers=settings.kafka.bootstrap_servers,
            client_id=settings.kafka.client_id,
        )

    def flush(self, timeout: float = None) -> int:
        wait = (
            timeout
            if timeout is not None
            else settings.kafka.producer_flush_timeout_seconds
        )
        remaining = self._producer.flush(wait)
        if remaining:
            log.warning(
                "kafka producer flush timeout",
                remaining=remaining,
                timeout_seconds=wait,
            )
        return int(remaining)

    def close(self) -> None:
        try:
            self.flush()
        finally:
            log.info("kafka producer closed")

    def publish_normalized(self, symbol: str, payload: Mapping[str, Any]) -> None:
        self._publish(
            settings.kafka.topic_market_data_normalized,
            MT_MARKET_NORMALIZED,
            symbol,
            payload,
        )

    def publish_signal(self, symbol: str, payload: Mapping[str, Any]) -> None:
        self._publish(
            settings.kafka.topic_trading_signals,
            MT_TRADING_SIGNAL,
            symbol,
            payload,
        )

    def publish_system_event(
        self,
        level: str,
        event_type: str,
        message: str,
        *,
        symbol: str = SYSTEM_SYMBOL_PLACEHOLDER,
        details: Mapping[str, Any] = None,
        retry_count: int = None,
        related_topic: str = None,
        event_id: str = None,
    ) -> None:
        lvl = level.lower()
        mt = LEVEL_TO_MESSAGE_TYPE.get(lvl)
        if mt is None:
            raise ValueError(f"unknown system event level='{level}', valid={list(LEVEL_TO_MESSAGE_TYPE)}")
        payload: dict[str, Any] = {
            "event_id": event_id or f"evt-stream_engine-{uuid.uuid4().hex[:12]}",
            "service": SOURCE,
            "level": lvl,
            "event_type": event_type,
            "message": message,
        }
        if details is not None:
            payload["details"] = dict(details)
        if retry_count is not None:
            payload["retry_count"] = int(retry_count)
        if related_topic is not None:
            payload["related_topic"] = related_topic
        self._publish(settings.kafka.topic_system_events, mt, symbol, payload)

    def _publish(
        self,
        topic: str,
        message_type: str,
        symbol: str,
        payload: Mapping[str, Any],
    ) -> None:
        envelope = {
            "message_type": message_type,
            "source": SOURCE,
            "timestamp": now_iso8601(),
            "symbol": str(symbol),
            "payload": dict(payload) if payload is not None else {},
        }
        try:
            value_bytes = json.dumps(
                envelope,
                ensure_ascii=False,
            default=json_default,
                separators=(",", ":"),
            ).encode("utf-8")
        except (TypeError, ValueError):
            log.exception(
                "kafka produce serialize failed",
                topic=topic,
                message_type=message_type,
                symbol=symbol,
            )
            raise

        self._producer.poll(0)
        try:
            self._producer.produce(
                topic=topic,
                key=str(symbol).encode("utf-8"),
                value=value_bytes,
                on_delivery=delivery_cb,
            )
        except BufferError:
            log.warning("kafka producer local queue full, flushing and retrying", topic=topic)
            self._producer.flush(settings.kafka.producer_flush_timeout_seconds)
            try:
                self._producer.produce(
                    topic=topic,
                    key=str(symbol).encode("utf-8"),
                    value=value_bytes,
                    on_delivery=delivery_cb,
                )
            except Exception:
                log.exception(
                    "kafka produce failed after flush retry",
                    topic=topic,
                    message_type=message_type,
                    symbol=symbol,
                )
        except Exception:
            log.exception(
                "kafka produce raised",
                topic=topic,
                message_type=message_type,
                symbol=symbol,
            )

    def _ensure_producer(self) -> Producer:
        return self._producer


producer_singleton: StreamProducer = None
singleton_lock = threading.Lock()


def get_producer() -> StreamProducer:
    global producer_singleton
    if producer_singleton is not None:
        return producer_singleton
    with singleton_lock:
        if producer_singleton is None:
            producer_singleton = StreamProducer()
        return producer_singleton


def reset_producer() -> None:
    global producer_singleton
    with singleton_lock:
        if producer_singleton is not None:
            try:
                producer_singleton.close()
            except Exception:
                log.exception("kafka producer close failed on reset")
        producer_singleton = None
