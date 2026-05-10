from __future__ import annotations

from dataclasses import dataclass

import pathway as pw

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)

PRICE_SCALE: int = 10000


class ReportSchema(pw.Schema):
    symbol: str
    exchange: str
    event_time: int
    last_price: int
    prev_close: int
    open_price: int
    high_price: int
    low_price: int
    volume: int
    turnover: int
    bid1_price: int
    bid2_price: int
    bid3_price: int
    bid4_price: int
    bid5_price: int
    bid1_volume: int
    bid2_volume: int
    bid3_volume: int
    bid4_volume: int
    bid5_volume: int
    ask1_price: int
    ask2_price: int
    ask3_price: int
    ask4_price: int
    ask5_price: int
    ask1_volume: int
    ask2_volume: int
    ask3_volume: int
    ask4_volume: int
    ask5_volume: int
    source: str
    timestamp: int


class FenbiSchema(pw.Schema):
    symbol: str
    trade_time: int
    price: int
    volume: int
    amount: int
    direction: str
    bid1: int
    ask1: int
    source: str
    timestamp: int


class FinanceSchema(pw.Schema):
    symbol: str
    report_date: int
    total_shares: int
    float_shares: int
    eps: int
    bps: int
    net_profit: int
    source: str
    timestamp: int


class MkttblSchema(pw.Schema):
    symbol: str
    name: str
    exchange: str
    lot_size: int
    source: str
    timestamp: int


class FiledataHistorySchema(pw.Schema):
    symbol: str
    trade_date: int
    open: int
    high: int
    low: int
    close: int
    volume: int
    turnover: int
    source: str
    timestamp: int


class FiledataMinuteSchema(pw.Schema):
    symbol: str
    trade_time: int
    price: int
    volume: int
    turnover: int
    source: str
    timestamp: int


class Filedata5MinuteSchema(pw.Schema):
    symbol: str
    trade_time: int
    open: int
    high: int
    low: int
    close: int
    volume: int
    turnover: int
    active_buy_vol: int
    source: str
    timestamp: int


class FiledataPowerSchema(pw.Schema):
    symbol: str
    ex_date: int
    bonus: int
    allotment: int
    allotment_price: int
    dividend: int
    source: str
    timestamp: int


@dataclass
class SourceTables:
    report: pw.Table
    fenbi: pw.Table
    mkttbl: pw.Table
    finance: pw.Table
    filedata_history: pw.Table
    filedata_minute: pw.Table
    filedata_5minute: pw.Table
    filedata_power: pw.Table


def _kafka_settings() -> dict:
    return {
        "bootstrap.servers": settings.kafka.bootstrap_servers,
        "group.id": settings.kafka.group_id,
        "auto.offset.reset": settings.kafka.auto_offset_reset,
        "enable.auto.commit": str(settings.kafka.enable_auto_commit).lower(),
    }


def build_source_tables() -> SourceTables:
    rdkafka = _kafka_settings()
    autocommit_ms = settings.pathway.autocommit_duration_ms

    report = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_report,
        schema=ReportSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    fenbi = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_fenbi,
        schema=FenbiSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    mkttbl = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_mkttbl,
        schema=MkttblSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    finance = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_finance,
        schema=FinanceSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    filedata_history = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_filedata_history,
        schema=FiledataHistorySchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    filedata_minute = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_filedata_minute,
        schema=FiledataMinuteSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    filedata_5minute = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_filedata_5minute,
        schema=Filedata5MinuteSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )
    filedata_power = pw.io.kafka.read(
        rdkafka_settings=rdkafka,
        topic=settings.kafka.topic_filedata_power,
        schema=FiledataPowerSchema,
        format="json",
        autocommit_duration_ms=autocommit_ms,
    )

    log.info("kafka sources created, PRICE_SCALE=%d", PRICE_SCALE)

    return SourceTables(
        report=report,
        fenbi=fenbi,
        mkttbl=mkttbl,
        finance=finance,
        filedata_history=filedata_history,
        filedata_minute=filedata_minute,
        filedata_5minute=filedata_5minute,
        filedata_power=filedata_power,
    )
