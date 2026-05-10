"""RCV_FINANCEDATA parser: produce fully normalized flat messages.

Output fields (all scalar):
  symbol, report_date, total_shares, float_shares, eps, bps, net_profit, source, timestamp
"""

import ctypes
import re

from config import settings
from models.market_protocol import Fin_LJF_STRUCTEx, RCV_DATA
from utils.envelope import (
    current_timestamp,
    emit_system_event,
    enqueue_kafka_message,
    format_epoch_date,
    normalize_symbol,
    price_to_int,
    to_int,
)
from utils.struct_utils import struct_to_dict

_DATE_PREFIX_RE = re.compile(r"^(\d{4})-(\d{2})-(\d{2})")


def _parse_finance_date(raw_payload: dict) -> str:
    """Extract report_date as YYYY-MM-DD string."""
    import datetime
    raw = raw_payload.get("finance_date") or raw_payload.get("formatted_date")
    if isinstance(raw, str) and raw:
        m = _DATE_PREFIX_RE.match(raw)
        if m:
            return f"{m.group(1)}-{m.group(2)}-{m.group(3)}"
    bgrq = raw_payload.get("BGRQ")
    if isinstance(bgrq, (int, float)) and bgrq > 0:
        return format_epoch_date(bgrq)
    return ""


def parse_finance_message(lparam):
    header_ptr = ctypes.cast(lparam, ctypes.POINTER(RCV_DATA))
    if not header_ptr:
        return
    header = header_ptr.contents
    base_address = ctypes.cast(header.data_union.m_pData, ctypes.c_void_p).value
    if not base_address:
        emit_system_event(
            "WARNING",
            "finance_missing_data_pointer",
            {"m_nPacketNum": int(header.m_nPacketNum)},
        )
        return

    struct_size = ctypes.sizeof(Fin_LJF_STRUCTEx)
    for index in range(header.m_nPacketNum):
        item_address = base_address + struct_size * index
        finance_record = ctypes.cast(
            item_address, ctypes.POINTER(Fin_LJF_STRUCTEx),
        ).contents
        raw = struct_to_dict(finance_record)

        market_code = raw.get("m_wMarket", 0)
        raw_label = raw.get("m_szLabel", "")
        symbol = normalize_symbol(market_code, raw_label)
        if not symbol:
            continue

        report_date = _parse_finance_date(raw)
        if not report_date:
            continue

        total_shares = to_int(raw.get("ZGB"))
        float_shares = to_int(raw.get("MQLT"))
        eps = price_to_int(raw.get("MGSY"))
        bps = price_to_int(raw.get("MGJZC"))
        net_profit = price_to_int(raw.get("JLR"))

        msg = {
            "symbol": symbol,
            "report_date": report_date,
            "total_shares": total_shares,
            "float_shares": float_shares,
            "eps": eps,
            "bps": bps,
            "net_profit": net_profit,
            "source": settings.service.source_name,
            "timestamp": current_timestamp(),
        }
        enqueue_kafka_message(settings.kafka.topic_finance, msg)
