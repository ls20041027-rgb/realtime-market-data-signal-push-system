"""Pipeline output layer.

Writes Pathway in-memory tables to external storage:
  - Kafka (real-time push to push_gateway)
  - PostgreSQL (persistent storage)
  - Redis (hot cache for API queries)

Each ``write_*`` function accepts a Pathway Table and sinks it
to the appropriate downstream storage according to TOPIC_CONTRACT.yaml.
"""

from __future__ import annotations

import json
from datetime import date
from decimal import Decimal
from typing import Any

import pathway as pw

from config import settings
from producer.signal_producer import get_producer
from storage.mysql_writer import insert_daily_capital_flow
from storage.redis_cache import get_redis_client
from utils.logger import get_logger

log = get_logger(__name__)

# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

def _to_jsonable(row: dict[str, Any]) -> dict[str, Any]:
    """Convert Decimal / date to JSON-safe types."""
    out: dict[str, Any] = {}
    for k, v in row.items():
        if isinstance(v, Decimal):
            out[k] = float(v)
        elif isinstance(v, date):
            out[k] = v.isoformat()
        else:
            out[k] = v
    return out


def _publish_table(
    table: pw.Table,
    topic: str,
    message_type: str,
) -> None:
    """Generic sink: emit every row to Kafka with ``producer.publish_*``.

    Pathway ``pw.io.kafka.write`` is not used here because we need
    fine-grained envelope control (message_type / source / timestamp).
    Instead we poll the table via ``pw.io.subscribe`` in a separate thread.
    """
    producer = get_producer()

    def _callback(key: Any, row: dict[str, Any]) -> None:
        symbol = row.get("symbol", key) if isinstance(key, str) else str(key)
        payload = _to_jsonable(row)
        if topic == settings.kafka.topic_trading_signals:
            producer.publish_signal(symbol, payload)
        elif topic == settings.kafka.topic_market_data_normalized:
            producer.publish_normalized(symbol, payload)
        else:
            log.warning("unknown output topic, skip", topic=topic)

    pw.io.subscribe(table, _callback)
    log.info("kafka sink subscribed", topic=topic, message_type=message_type)


# ---------------------------------------------------------------------------
# signal output
# ---------------------------------------------------------------------------

def write_signals(table: pw.Table) -> None:
    """Write trading signals to Kafka ``topic_trading_signals``."""
    _publish_table(table, settings.kafka.topic_trading_signals, "TRADING_SIGNAL")
    log.info("signal output wired", topic=settings.kafka.topic_trading_signals)


# ---------------------------------------------------------------------------
# quote snapshot → Kafka + Redis
# ---------------------------------------------------------------------------

def write_quote_snapshot(table: pw.Table) -> None:
    """Write real-time quote snapshot to Kafka + Redis cache."""
    _publish_table(table, settings.kafka.topic_market_data_normalized, "MARKET_SNAPSHOT_NORMALIZED")

    # Redis hot cache: latest snapshot per symbol
    redis = get_redis_client()

    def _cache(row: dict[str, Any]) -> None:
        symbol = row.get("symbol")
        if symbol is None:
            return
        key = f"quote:{symbol}"
        redis.set(key, json.dumps(_to_jsonable(row), ensure_ascii=False))
        redis.expire(key, 300)

    pw.io.subscribe(table, lambda k, r: _cache(r))
    log.info("quote snapshot output wired", topic=settings.kafka.topic_market_data_normalized)


# ---------------------------------------------------------------------------
# market normalized → Kafka
# ---------------------------------------------------------------------------

def write_market_normalized(table: pw.Table) -> None:
    """Write normalized market data to Kafka."""
    _publish_table(table, settings.kafka.topic_market_data_normalized, "MARKET_SNAPSHOT_NORMALIZED")
    log.info("market normalized output wired")


# ---------------------------------------------------------------------------
# capital flow → PostgreSQL + Redis
# ---------------------------------------------------------------------------

def write_capital_flow(table: pw.Table) -> None:
    """Upsert capital flow to PostgreSQL ``daily_capital_flow`` + Redis."""

    def _persist(row: dict[str, Any]) -> None:
        try:
            from datetime import date as _date
            symbol = row["symbol"]
            trade_date = row.get("trade_date") or _date.today()
            if isinstance(trade_date, str):
                trade_date = _date.fromisoformat(trade_date)
            big_buy = Decimal(str(row.get("big_buy", 0)))
            big_sell = Decimal(str(row.get("big_sell", 0)))
            net_inflow = big_buy - big_sell
            insert_daily_capital_flow(symbol, trade_date, big_buy, big_sell, net_inflow)
        except Exception:
            log.exception("capital flow persist failed", symbol=row.get("symbol"))

    pw.io.subscribe(table, lambda k, r: _persist(r))

    # Redis cache
    redis = get_redis_client()

    def _cache(row: dict[str, Any]) -> None:
        symbol = row.get("symbol")
        if symbol is None:
            return
        key = f"capital_flow:{symbol}"
        redis.set(key, json.dumps(_to_jsonable(row), ensure_ascii=False))
        redis.expire(key, 300)

    pw.io.subscribe(table, lambda k, r: _cache(r))
    log.info("capital flow output wired")


# ---------------------------------------------------------------------------
# finance → PostgreSQL
# ---------------------------------------------------------------------------

def write_finance(table: pw.Table) -> None:
    """Upsert financial data to PostgreSQL ``stock_finance``."""
    from storage.models import StockFinance
    from storage.mysql_writer import get_session

    def _persist(row: dict[str, Any]) -> None:
        try:
            from datetime import date as _date
            symbol = row["symbol"]
            report_date = row.get("report_date")
            if isinstance(report_date, str):
                report_date = _date.fromisoformat(report_date)

            with get_session() as session:
                obj = session.get(StockFinance, (symbol, report_date))
                if obj is None:
                    obj = StockFinance(
                        symbol=symbol,
                        report_date=report_date,
                        total_shares=Decimal(str(row.get("total_shares", 0))),
                        float_shares=Decimal(str(row.get("float_shares", 0))),
                        eps=Decimal(str(row["eps"])) if row.get("eps") is not None else None,
                        bps=Decimal(str(row["bps"])) if row.get("bps") is not None else None,
                        net_profit=Decimal(str(row["net_profit"])) if row.get("net_profit") is not None else None,
                    )
                    session.add(obj)
                else:
                    obj.total_shares = Decimal(str(row.get("total_shares", obj.total_shares)))
                    obj.float_shares = Decimal(str(row.get("float_shares", obj.float_shares)))
                    if row.get("eps") is not None:
                        obj.eps = Decimal(str(row["eps"]))
                    if row.get("bps") is not None:
                        obj.bps = Decimal(str(row["bps"]))
                    if row.get("net_profit") is not None:
                        obj.net_profit = Decimal(str(row["net_profit"]))
        except Exception:
            log.exception("finance persist failed", symbol=row.get("symbol"))

    pw.io.subscribe(table, lambda k, r: _persist(r))
    log.info("finance output wired")


# ---------------------------------------------------------------------------
# stock info → PostgreSQL
# ---------------------------------------------------------------------------

def write_stock_info(table: pw.Table) -> None:
    """Upsert stock basic info to PostgreSQL ``stock_info``."""
    from storage.models import StockInfo
    from storage.mysql_writer import get_session

    def _persist(row: dict[str, Any]) -> None:
        try:
            symbol = row["symbol"]
            with get_session() as session:
                obj = session.get(StockInfo, symbol)
                if obj is None:
                    obj = StockInfo(
                        symbol=symbol,
                        name=row.get("name", symbol),
                        exchange=row.get("exchange", ""),
                        lot_size=int(row.get("lot_size", 100)),
                    )
                    session.add(obj)
                else:
                    obj.name = row.get("name", obj.name)
                    obj.exchange = row.get("exchange", obj.exchange)
                    obj.lot_size = int(row.get("lot_size", obj.lot_size))
        except Exception:
            log.exception("stock_info persist failed", symbol=row.get("symbol"))

    pw.io.subscribe(table, lambda k, r: _persist(r))
    log.info("stock_info output wired")


# ---------------------------------------------------------------------------
# filedata → PostgreSQL (history / minute / 5minute / power)
# ---------------------------------------------------------------------------

def _write_filedata_generic(table: pw.Table, topic_name: str) -> None:
    """Generic filedata sink — log only (detail table TBD)."""
    def _log(row: dict[str, Any]) -> None:
        log.debug("filedata row", topic=topic_name, symbol=row.get("symbol"))

    pw.io.subscribe(table, lambda k, r: _log(r))
    log.info("filedata output wired", topic=topic_name)


def write_filedata_history(table: pw.Table) -> None:
    _write_filedata_generic(table, "filedata_history")


def write_filedata_minute(table: pw.Table) -> None:
    _write_filedata_generic(table, "filedata_minute")


def write_filedata_5minute(table: pw.Table) -> None:
    _write_filedata_generic(table, "filedata_5minute")


def write_filedata_power(table: pw.Table) -> None:
    _write_filedata_generic(table, "filedata_power")


# ---------------------------------------------------------------------------
# 5-min K line realtime → Kafka + Redis
# ---------------------------------------------------------------------------

def write_5min_kline_rt(table: pw.Table) -> None:
    """Write real-time 5-min K line to Kafka + Redis."""
    _publish_table(table, settings.kafka.topic_market_data_normalized, "MARKET_SNAPSHOT_NORMALIZED")

    redis = get_redis_client()

    def _cache(row: dict[str, Any]) -> None:
        symbol = row.get("symbol")
        if symbol is None:
            return
        key = f"kline_5min:{symbol}"
        redis.set(key, json.dumps(_to_jsonable(row), ensure_ascii=False))
        redis.expire(key, 300)

    pw.io.subscribe(table, lambda k, r: _cache(r))
    log.info("5min kline rt output wired")
