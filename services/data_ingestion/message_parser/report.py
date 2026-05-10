"""RCV_REPORT parser: produce fully normalized flat messages for Pathway direct ingestion.

Output fields (all scalar, no nesting):
  symbol, exchange, event_time, last_price, prev_close, open_price, high_price, low_price,
  volume, turnover, bid1_price..bid5_price, bid1_volume..bid5_volume,
  ask1_price..ask5_price, ask1_volume..ask5_volume, source, timestamp
"""

import ctypes

from config import settings
from models.market_protocol import RCV_DATA, RCV_REPORT_STRUCTExV3
from utils.envelope import (
    current_timestamp,
    detect_exchange,
    enqueue_kafka_message,
    format_epoch_timestamp,
    normalize_symbol,
    price_to_int,
    to_int,
)
from utils.struct_utils import struct_to_dict


def parse_report_message(lparam):
    header_ptr = ctypes.cast(lparam, ctypes.POINTER(RCV_DATA))
    if not header_ptr:
        return
    header = header_ptr.contents
    report_ptr = ctypes.cast(
        header.data_union.m_pReportV3,
        ctypes.POINTER(RCV_REPORT_STRUCTExV3),
    )
    for index in range(header.m_nPacketNum):
        buf = report_ptr[index]
        raw = struct_to_dict(buf)

        market_code = raw.get("m_wMarket", 0)
        raw_label = raw.get("m_szLabel", "")
        symbol = normalize_symbol(market_code, raw_label)
        if not symbol:
            continue
        exchange = detect_exchange(symbol)

        last_price = price_to_int(raw.get("m_fNewPrice"))
        prev_close = price_to_int(raw.get("m_fLastClose"))
        open_price = price_to_int(raw.get("m_fOpen"))
        high_price = price_to_int(raw.get("m_fHigh"))
        low_price = price_to_int(raw.get("m_fLow"))

        volume = to_int(raw.get("m_fVolume"))
        turnover = price_to_int(raw.get("m_fAmount"))

        buy_prices = raw.get("m_fBuyPrice", [0, 0, 0])
        buy_volumes = raw.get("m_fBuyVolume", [0, 0, 0])
        sell_prices = raw.get("m_fSellPrice", [0, 0, 0])
        sell_volumes = raw.get("m_fSellVolume", [0, 0, 0])

        event_time = format_epoch_timestamp(raw.get("m_time"))

        msg = {
            "symbol": symbol,
            "exchange": exchange,
            "event_time": event_time,
            "last_price": last_price,
            "prev_close": prev_close,
            "open_price": open_price,
            "high_price": high_price,
            "low_price": low_price,
            "volume": volume,
            "turnover": turnover,
            "bid1_price": price_to_int(buy_prices[0] if len(buy_prices) > 0 else 0),
            "bid2_price": price_to_int(buy_prices[1] if len(buy_prices) > 1 else 0),
            "bid3_price": price_to_int(buy_prices[2] if len(buy_prices) > 2 else 0),
            "bid4_price": price_to_int(raw.get("m_fBuyPrice4", 0)),
            "bid5_price": price_to_int(raw.get("m_fBuyPrice5", 0)),
            "bid1_volume": to_int(buy_volumes[0] if len(buy_volumes) > 0 else 0),
            "bid2_volume": to_int(buy_volumes[1] if len(buy_volumes) > 1 else 0),
            "bid3_volume": to_int(buy_volumes[2] if len(buy_volumes) > 2 else 0),
            "bid4_volume": to_int(raw.get("m_fBuyVolume4", 0)),
            "bid5_volume": to_int(raw.get("m_fBuyVolume5", 0)),
            "ask1_price": price_to_int(sell_prices[0] if len(sell_prices) > 0 else 0),
            "ask2_price": price_to_int(sell_prices[1] if len(sell_prices) > 1 else 0),
            "ask3_price": price_to_int(sell_prices[2] if len(sell_prices) > 2 else 0),
            "ask4_price": price_to_int(raw.get("m_fSellPrice4", 0)),
            "ask5_price": price_to_int(raw.get("m_fSellPrice5", 0)),
            "ask1_volume": to_int(sell_volumes[0] if len(sell_volumes) > 0 else 0),
            "ask2_volume": to_int(sell_volumes[1] if len(sell_volumes) > 1 else 0),
            "ask3_volume": to_int(sell_volumes[2] if len(sell_volumes) > 2 else 0),
            "ask4_volume": to_int(raw.get("m_fSellVolume4", 0)),
            "ask5_volume": to_int(raw.get("m_fSellVolume5", 0)),
            "source": settings.service.source_name,
            "timestamp": current_timestamp(),
        }
        enqueue_kafka_message(settings.kafka.topic_report, msg)
