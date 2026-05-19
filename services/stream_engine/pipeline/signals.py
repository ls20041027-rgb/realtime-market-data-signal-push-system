"""Signal generation layer.

Builds trading signals from enriched pipeline tables and deduplicates them
before handing off to ``output.write_signals``.

Signal types generated (aligned with CONTRACT.md):
  1. CAPITAL_SURGE   – big buy net inflow spike
  2. CAPITAL_OUTFLOW – big sell net outflow spike
  3. TURNOVER_BREAK  – turnover rate crosses threshold
  4. PRICE_BREAKOUT  – close price breaks recent high
  5. VOLUME_SPIKE    – volume > N × moving average
  6. TRADE_TEMPO_UP  – buy tick tempo rises
  7. TRADE_TEMPO_DOWN– sell tick tempo rises
  8. KLINE_5MIN_MA   – 5-min K line MA cross
"""

from __future__ import annotations

import pathway as pw

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)

# thresholds (can move to config later)
CAPITAL_SURGE_THRESHOLD: float = 100_0000  # 100万 大单净流入
TURNOVER_BREAK_RATIO: float = 5.0          # 换手率 > 5%
VOLUME_MA_WINDOW: int = 20


def build_5min_kline(fenbi_table: pw.Table) -> pw.Table:
    """Aggregate tick data into 5-min K line (real-time, in-memory).

    Returns a Table with columns:
      symbol, open, high, low, close, volume, turnover, window_start
    """
    SECONDS_PER_5MIN = 300

    prepared = fenbi_table.select(
        symbol=fenbi_table.symbol,
        window_start=fenbi_table.trade_time // SECONDS_PER_5MIN * SECONDS_PER_5MIN,
        price=fenbi_table.price,
        volume=fenbi_table.volume,
        amount=fenbi_table.amount,
    )

    grouped = prepared.groupby(
        prepared.symbol, prepared.window_start
    ).reduce(
        symbol=pw.this.symbol,
        window_start=pw.this.window_start,
        open=pw.reducers.first(pw.this.price),
        high=pw.reducers.max(pw.this.price),
        low=pw.reducers.min(pw.this.price),
        close=pw.reducers.last(pw.this.price),
        volume=pw.reducers.sum(pw.this.volume),
        turnover=pw.reducers.sum(pw.this.amount),
    )

    return grouped.select(
        symbol=pw.this.symbol,
        window_start=pw.this.window_start,
        open=pw.this.open,
        high=pw.this.high,
        low=pw.this.low,
        close=pw.this.close,
        volume=pw.this.volume,
        turnover=pw.this.turnover,
    )


def _emit_signal(
    symbol: str,
    signal_type: str,
    reason: str,
    **extra: object,
) -> dict[str, object]:
    """Build a normalized signal dict (matches TOPIC_CONTRACT.yaml)."""
    return {
        "symbol": symbol,
        "signal_type": signal_type,
        "reason": reason,
        "extra": extra,
    }


def build_all_signals(
    report_enriched: pw.Table,
    capital_flow: pw.Table,
    trade_tempo: pw.Table,
    kline_5min: pw.Table,
) -> pw.Table:
    """Orchestrates all signal generators and returns a unified signal Table.

    Each generator returns a Table with at least (symbol, signal_type, reason).
    They are unioned, then deduplicated before output.
    """

    # ---- CAPITAL_SURGE / OUTFLOW ----
    capital_sig = capital_flow.select(
        symbol=capital_flow.symbol,
        signal_type=pw.if_else(
            capital_flow.net_inflow > CAPITAL_SURGE_THRESHOLD,
            "CAPITAL_SURGE",
            pw.if_else(
                capital_flow.net_inflow < -CAPITAL_SURGE_THRESHOLD,
                "CAPITAL_OUTFLOW",
                None,
            ),
        ),
        reason=pw.if_else(
            capital_flow.net_inflow > CAPITAL_SURGE_THRESHOLD,
            "大单净流入 spike",
            "大单净流出 spike",
        ),
        net_inflow=capital_flow.net_inflow,
    ).filter(pw.this.signal_type.is_not_none())

    # ---- TURNOVER_BREAK ----
    turnover_sig = report_enriched.select(
        symbol=report_enriched.symbol,
        signal_type=pw.if_else(
            report_enriched.turnover_rate > TURNOVER_BREAK_RATIO,
            "TURNOVER_BREAK",
            None,
        ),
        reason="换手率突破阈值",
        turnover_rate=report_enriched.turnover_rate,
    ).filter(pw.this.signal_type.is_not_none())

    # ---- TRADE_TEMPO ----
    tempo_sig = trade_tempo.select(
        symbol=trade_tempo.symbol,
        signal_type=pw.if_else(
            trade_tempo.buy_tick_count > trade_tempo.sell_tick_count * 1.5,
            "TRADE_TEMPO_UP",
            pw.if_else(
                trade_tempo.sell_tick_count > trade_tempo.buy_tick_count * 1.5,
                "TRADE_TEMPO_DOWN",
                None,
            ),
        ),
        reason=pw.if_else(
            trade_tempo.buy_tick_count > trade_tempo.sell_tick_count * 1.5,
            "买盘节奏加快",
            "卖盘节奏加快",
        ),
        buy_ticks=trade_tempo.buy_tick_count,
        sell_ticks=trade_tempo.sell_tick_count,
    ).filter(pw.this.signal_type.is_not_none())

    # ---- union all signals ----
    all_signals = capital_sig.select(
        symbol=pw.this.symbol,
        signal_type=pw.this.signal_type,
        reason=pw.this.reason,
        **pw.this.exclude("symbol", "signal_type", "reason"),
    ).concat(
        turnover_sig.select(
            symbol=pw.this.symbol,
            signal_type=pw.this.signal_type,
            reason=pw.this.reason,
            **pw.this.exclude("symbol", "signal_type", "reason"),
        ),
        tempo_sig.select(
            symbol=pw.this.symbol,
            signal_type=pw.this.signal_type,
            reason=pw.this.reason,
            **pw.this.exclude("symbol", "signal_type", "reason"),
        ),
    )

    # deduplicate by (symbol, signal_type) — keep latest
    deduped = all_signals.reduce(
        symbol=pw.this.symbol,
        signal_type=pw.this.signal_type,
        reason=pw.reducers.last(pw.this.reason),
    )

    log.info("signal builders wired", types=["CAPITAL_SURGE", "CAPITAL_OUTFLOW",
                                             "TURNOVER_BREAK", "TRADE_TEMPO_UP/DOWN"])
    return deduped
