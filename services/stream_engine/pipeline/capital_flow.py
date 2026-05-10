from __future__ import annotations

import pathway as pw

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)

BIG_ORDER_THRESHOLD: int = int(settings.thresholds.big_order_threshold) * 10000
TEMPO_WINDOW_SECONDS: int = settings.thresholds.tempo_window_seconds
SECONDS_PER_DAY: int = 86400


def build_capital_flow(fenbi_table: pw.Table) -> pw.Table:
    prepared = fenbi_table.select(
        symbol=fenbi_table.symbol,
        trade_date=fenbi_table.trade_time // SECONDS_PER_DAY * SECONDS_PER_DAY,
        amount=fenbi_table.amount,
        direction=fenbi_table.direction,
        is_buy=pw.if_else(fenbi_table.direction == "BUY", 1, 0),
        is_sell=pw.if_else(fenbi_table.direction == "SELL", 1, 0),
        big_buy_amount=pw.if_else(
            (fenbi_table.direction == "BUY") & (fenbi_table.amount >= BIG_ORDER_THRESHOLD),
            fenbi_table.amount,
            0,
        ),
        big_sell_amount=pw.if_else(
            (fenbi_table.direction == "SELL") & (fenbi_table.amount >= BIG_ORDER_THRESHOLD),
            fenbi_table.amount,
            0,
        ),
    )

    grouped = prepared.groupby(prepared.symbol, prepared.trade_date).reduce(
        symbol=pw.this.symbol,
        trade_date=pw.this.trade_date,
        big_buy=pw.reducers.sum(pw.this.big_buy_amount),
        big_sell=pw.reducers.sum(pw.this.big_sell_amount),
        buy_tick_count=pw.reducers.sum(pw.this.is_buy),
        sell_tick_count=pw.reducers.sum(pw.this.is_sell),
    )

    return grouped.select(
        symbol=pw.this.symbol,
        trade_date=pw.this.trade_date,
        big_buy=pw.this.big_buy,
        big_sell=pw.this.big_sell,
        net_inflow=pw.this.big_buy - pw.this.big_sell,
        buy_tick_count=pw.this.buy_tick_count,
        sell_tick_count=pw.this.sell_tick_count,
    )


def build_trade_tempo(fenbi_table: pw.Table) -> pw.Table:
    prepared = fenbi_table.select(
        symbol=fenbi_table.symbol,
        epoch_seconds=fenbi_table.trade_time,
        amount=fenbi_table.amount,
        is_buy=pw.if_else(fenbi_table.direction == "BUY", 1, 0),
        is_sell=pw.if_else(fenbi_table.direction == "SELL", 1, 0),
    )

    windowed = prepared.windowby(
        prepared.epoch_seconds,
        window=pw.temporal.sliding(hop=TEMPO_WINDOW_SECONDS, duration=TEMPO_WINDOW_SECONDS),
        instance=prepared.symbol,
    ).reduce(
        symbol=pw.this._pw_instance,
        tick_count=pw.reducers.count(),
        buy_tick_count=pw.reducers.sum(pw.this.is_buy),
        sell_tick_count=pw.reducers.sum(pw.this.is_sell),
        total_amount=pw.reducers.sum(pw.this.amount),
    )

    return windowed.select(
        symbol=pw.this.symbol,
        tick_count=pw.this.tick_count,
        buy_tick_count=pw.this.buy_tick_count,
        sell_tick_count=pw.this.sell_tick_count,
        avg_amount=pw.if_else(pw.this.tick_count > 0, pw.this.total_amount // pw.this.tick_count, 0),
        window_seconds=TEMPO_WINDOW_SECONDS,
    )
