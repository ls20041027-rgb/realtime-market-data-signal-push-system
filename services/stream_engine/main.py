"""stream_engine service entry point.

Flow: start ingest stats sidecar → build_pipeline → pw.run → shutdown.
"""

from __future__ import annotations

import os
import signal
import sys
import threading
from datetime import datetime
from pathlib import Path
from typing import Any

SERVICE_ROOT = Path(__file__).resolve().parent
if str(SERVICE_ROOT) not in sys.path:
    sys.path.insert(0, str(SERVICE_ROOT))

try:
    import pathway as pw
except ImportError:
    import logging

    logging.exception("pathway import failed, aborting startup")
    sys.exit(-2)

from config import settings
from pipeline.build import build_pipeline
from pipeline.ingest_stats import start_ingest_stats, stop_ingest_stats
from producer.signal_producer import get_producer
from storage.mysql_writer import reset_engine
from storage.redis_cache import reset_redis
from utils.logger import get_logger


log = get_logger("stream_engine.main")


_shutdown_event = threading.Event()


def install_signal_handlers() -> None:
    def handler(signum: int, _frame: Any) -> None:
        log.info("main received signal", signum=signum)
        _shutdown_event.set()

    try:
        signal.signal(signal.SIGINT, handler)
        signal.signal(signal.SIGTERM, handler)
        log.info("signal handlers installed", signals=["SIGINT", "SIGTERM"])
    except ValueError:
        log.debug("skip signal handlers: not in main thread")


def shutdown() -> None:
    """Release resources in order: stats → producer → mysql → redis."""
    log.info("graceful shutdown begin")
    try:
        stop_ingest_stats()
    except Exception:
        log.exception("ingest stats shutdown failed")
    try:
        producer = get_producer()
        remaining = producer.flush()
        producer.close()
        log.info("producer flushed and closed", remaining=remaining)
    except Exception:
        log.exception("producer shutdown failed")
    try:
        reset_engine()
    except Exception:
        log.exception("mysql engine shutdown failed")
    try:
        reset_redis()
    except Exception:
        log.exception("redis shutdown failed")
    log.info("graceful shutdown done")


def main() -> int:
    log.info(
        "stream_engine starting",
        pid=os.getpid(),
        service_root=str(SERVICE_ROOT),
    )

    start_ingest_stats()

    install_signal_handlers()
    try:
        build_pipeline()
    except Exception:
        log.exception("pipeline build failed, aborting startup")
        return -3

    log.info(
        "pw.run entering main loop",
        topics=[
            settings.kafka.topic_report,
            settings.kafka.topic_fenbi,
            settings.kafka.topic_mkttbl,
            settings.kafka.topic_finance,
            settings.kafka.topic_filedata_history,
            settings.kafka.topic_filedata_minute,
            settings.kafka.topic_filedata_5minute,
            settings.kafka.topic_filedata_power,
        ],
        group_id=settings.kafka.group_id,
    )
    try:
        pw.run(monitoring_level=pw.MonitoringLevel.NONE)
    except KeyboardInterrupt:
        log.info("keyboard interrupt at main level")
    except Exception:
        log.exception("pw.run raised, shutting down")
    finally:
        shutdown()
    log.info("stream_engine exited cleanly")
    return 0


if __name__ == "__main__":
    sys.exit(main())
