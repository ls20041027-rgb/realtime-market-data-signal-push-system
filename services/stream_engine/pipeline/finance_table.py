from __future__ import annotations

import pathway as pw

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)


def build_finance_table(finance_table: pw.Table) -> pw.Table:
    return finance_table.groupby(finance_table.symbol).reduce(
        symbol=pw.this.symbol,
        float_shares=pw.reducers.latest(finance_table.float_shares),
        total_shares=pw.reducers.latest(finance_table.total_shares),
    )


def compute_turnover_rate(volume: int, float_shares: int) -> int:
    if float_shares <= 0:
        return 0
    return volume * 10000 // float_shares
