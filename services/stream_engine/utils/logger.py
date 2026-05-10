"""结构化 JSON logger（文件滚动版）。

对外 API：``get_logger(name)`` 返回一个可传任意 kwargs 的 structlog bound logger。
JSON 字段：``timestamp`` / ``level`` / ``logger`` / ``message`` + 业务 kwargs。

日志输出到滚动文件，单文件最大 100 MiB，最多保留 10 个备份，配置统一走 config.yaml。
"""

from __future__ import annotations

import logging
import sys
from logging.handlers import RotatingFileHandler
from pathlib import Path

import structlog

from config import settings

CONFIGURED: bool = False


def resolve_level() -> int:
    raw = settings.runtime.log_level or "INFO"
    level = logging.getLevelName(str(raw).upper())
    if not isinstance(level, int):
        print(f"[FATAL] invalid log_level='{raw}' in config.yaml")
        sys.exit(-1)
    return level


def configure() -> None:
    global CONFIGURED
    if CONFIGURED:
        return
    level = resolve_level()

    log_file = Path(settings.runtime.log_file)
    if not log_file.is_absolute():
        log_file = Path(__file__).resolve().parent.parent / log_file
    log_file.parent.mkdir(parents=True, exist_ok=True)

    max_bytes = settings.runtime.log_max_bytes
    backup_count = settings.runtime.log_backup_count

    root = logging.getLogger()
    for h in list(root.handlers):
        root.removeHandler(h)
    file_handler = RotatingFileHandler(
        str(log_file), maxBytes=max_bytes, backupCount=backup_count, encoding="utf-8"
    )
    root.addHandler(file_handler)
    root.setLevel(level)

    log_fp = open(str(log_file), "a", encoding="utf-8")

    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso", utc=True, key="timestamp"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.EventRenamer("message"),
            structlog.processors.JSONRenderer(sort_keys=False),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(level),
        logger_factory=structlog.PrintLoggerFactory(file=log_fp),
        cache_logger_on_first_use=True,
    )
    CONFIGURED = True


def get_logger(name: str):
    """返回一个结构化 JSON logger；kwargs 作为顶层字段打印。"""
    if not CONFIGURED:
        configure()
    return structlog.get_logger(name).bind(logger=name)
