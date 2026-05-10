"""RCV_FENBIDATA tick-by-tick parser: produce fully normalized flat messages.

Output fields (all scalar, no nesting):
  symbol, trade_time, price, volume, amount, direction,
  bid1, ask1, source, timestamp
"""

import ctypes

from config import settings
from models.market_protocol import RCV_FENBI, RCV_FENBI_STRUCTEx
from utils.envelope import (
    current_timestamp,
    enqueue_kafka_message,
    format_epoch_timestamp,
    normalize_symbol,
    price_to_int,
    to_int,
)
from utils.struct_utils import struct_to_dict


def _decide_direction(price: int, bid1: int, ask1: int) -> str:
    """Determine trade direction: BUY / SELL / NEUTRAL (integer comparison)."""
    if ask1 > 0 and price >= ask1:
        return "BUY"
    if bid1 > 0 and price <= bid1:
        return "SELL"
    return "NEUTRAL"


def parse_fenbi_message(lparam):
    fenbi_header = ctypes.cast(lparam, ctypes.POINTER(RCV_FENBI)).contents
    header_dict = struct_to_dict(fenbi_header)
    header_dict.pop("m_Data", None)

    market_code = header_dict.get("m_wMarket", 0)
    raw_label = header_dict.get("m_szLabel", "")
    symbol = normalize_symbol(market_code, raw_label)
    if not symbol:
        return

    for index in range(fenbi_header.m_nCount):
        item_offset = settings.runtime.fenbi_header_size + ctypes.sizeof(RCV_FENBI_STRUCTEx) * index
        fenbi_item = ctypes.cast(
            lparam + item_offset,
            ctypes.POINTER(RCV_FENBI_STRUCTEx),
        ).contents
        item_dict = struct_to_dict(fenbi_item)

        price = price_to_int(item_dict.get("m_fNewPrice"))
        volume = to_int(item_dict.get("m_fVolume"))
        amount = price_to_int(item_dict.get("m_fAmount"))

        buy_prices = item_dict.get("m_fBuyPrice", [0])
        sell_prices = item_dict.get("m_fSellPrice", [0])
        bid1 = price_to_int(buy_prices[0] if buy_prices else 0)
        ask1 = price_to_int(sell_prices[0] if sell_prices else 0)

        direction = _decide_direction(price, bid1, ask1)

        trade_time = format_epoch_timestamp(item_dict.get("m_lTime"))

        msg = {
            "symbol": symbol,
            "trade_time": trade_time,
            "price": price,
            "volume": volume,
            "amount": amount,
            "direction": direction,
            "bid1": bid1,
            "ask1": ask1,
            "source": settings.service.source_name,
            "timestamp": current_timestamp(),
        }
        enqueue_kafka_message(settings.kafka.topic_fenbi, msg)
