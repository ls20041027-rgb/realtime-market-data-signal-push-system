"""RCV_MKTTBLDATA market table parser: produce fully normalized flat messages.

Output fields for security records (all scalar):
  symbol, name, exchange, lot_size, source, timestamp
Market-level metadata records are skipped (no downstream consumer needs them).
"""

import ctypes

from config import settings
from models.market_protocol import HLMarketType, RCV_TABLE_STRUCT
from utils.envelope import (
    current_timestamp,
    detect_exchange,
    enqueue_kafka_message,
    normalize_symbol,
)
from utils.struct_utils import struct_to_dict

_MKTTBL_HEADER_SIZE = 54
_SECURITY_STRUCT_SIZE = ctypes.sizeof(RCV_TABLE_STRUCT)


def parse_mkttbl_message(lparam):
    market_header = ctypes.cast(lparam, ctypes.POINTER(HLMarketType)).contents
    market_payload = struct_to_dict(market_header)
    market_code = market_payload.get("m_wMarket", 0)

    for index in range(market_header.m_nCount):
        offset = _MKTTBL_HEADER_SIZE + _SECURITY_STRUCT_SIZE * index
        security_struct = ctypes.cast(
            lparam + offset, ctypes.POINTER(RCV_TABLE_STRUCT)
        ).contents
        security_record = struct_to_dict(security_struct)

        raw_label = security_record.get("m_szLabel", "")
        symbol = normalize_symbol(market_code, raw_label)
        if not symbol:
            continue

        exchange = detect_exchange(symbol)
        name = str(security_record.get("m_szName", "")).strip()
        if not name:
            continue

        lot_size = int(security_record.get("m_cProperty", 0))
        if lot_size <= 0:
            continue

        msg = {
            "symbol": symbol,
            "name": name,
            "exchange": exchange,
            "lot_size": lot_size,
            "source": settings.service.source_name,
            "timestamp": current_timestamp(),
        }
        enqueue_kafka_message(settings.kafka.topic_mkttbl, msg)
