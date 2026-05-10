from __future__ import annotations

import pathway as pw

from pipeline.capital_flow import (
    build_capital_flow,
    build_trade_tempo,
)
from pipeline.finance_table import build_finance_table
from pipeline.output import (
    write_5min_kline_rt,
    write_capital_flow,
    write_filedata_5minute,
    write_filedata_history,
    write_filedata_minute,
    write_filedata_power,
    write_finance,
    write_market_normalized,
    write_quote_snapshot,
    write_signals,
    write_stock_info,
)
from pipeline.signals import build_all_signals, build_5min_kline
from pipeline.source import build_source_tables
from utils.logger import get_logger

log = get_logger(__name__)


def build_pipeline() -> None:
    sources = build_source_tables()

    finance_table = build_finance_table(sources.finance)

    report_joined = sources.report.join_left(
        finance_table,
        sources.report.symbol == finance_table.symbol,
    ).select(
        *sources.report,
        float_shares=pw.coalesce(finance_table.float_shares, 0),
    )

    report_enriched = report_joined.select(
        *pw.this,
        turnover_rate=pw.if_else(
            pw.this.float_shares > 0,
            pw.this.volume * 10000 // pw.this.float_shares,
            0,
        ),
    )

    capital_flow_table = build_capital_flow(sources.fenbi)
    trade_tempo_table = build_trade_tempo(sources.fenbi)

    kline_5min_rt = build_5min_kline(sources.fenbi)

    deduped_signals = build_all_signals(
        report_enriched=report_enriched,
        capital_flow=capital_flow_table,
        trade_tempo=trade_tempo_table,
        kline_5min=kline_5min_rt,
    )

    write_signals(deduped_signals)
    write_quote_snapshot(report_enriched)
    write_market_normalized(sources.report)
    write_capital_flow(capital_flow_table)
    write_finance(finance_table)
    write_stock_info(sources.mkttbl)
    write_filedata_history(sources.filedata_history)
    write_filedata_minute(sources.filedata_minute)
    write_filedata_5minute(sources.filedata_5minute)
    write_filedata_power(sources.filedata_power)
    write_5min_kline_rt(kline_5min_rt)

    log.info("pipeline built, signal_types=8, output_tables=10")
