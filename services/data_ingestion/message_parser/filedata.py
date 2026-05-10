"""RCV_FILEDATA multi-subtype parser: split into 4 separate Kafka topics.

Each subtype gets its own topic with fully normalized flat messages:
  - market.filedata.history  → symbol, trade_date, open, high, low, close, volume, turnover
  - market.filedata.minute   → symbol, trade_time, price, volume, turnover
  - market.filedata.5minute  → symbol, trade_time, open, high, low, close, volume, turnover, active_buy_vol
  - market.filedata.power    → symbol, ex_date, bonus, allotment, allotment_price, dividend

Only record_type=="data" records are emitted. Headers/file_headers are skipped.
"""

import ctypes

from config import settings
from models.market_protocol import (
    EKE_HEAD_TAG,
    FILE_5MINUTE_EX,
    FILE_BASE_EX,
    FILE_HISTORY_EX,
    FILE_HTML_EX,
    FILE_MINUTE_EX,
    FILE_NEWS_EX,
    FILE_POWER_EX,
    FILE_TYPE_RES,
    RCV_DATA,
)
from utils.envelope import (
    current_timestamp,
    emit_system_event,
    enqueue_kafka_message,
    format_epoch_date,
    format_epoch_timestamp,
    normalize_symbol,
    price_to_int,
    to_int,
)
from utils.struct_utils import struct_to_dict


def _build_history_msg(symbol: str, record: dict) -> dict:
    return {
        "symbol": symbol,
        "trade_date": format_epoch_date(record.get("m_time")),
        "open": price_to_int(record.get("m_fOpen")),
        "high": price_to_int(record.get("m_fHigh")),
        "low": price_to_int(record.get("m_fLow")),
        "close": price_to_int(record.get("m_fClose")),
        "volume": to_int(record.get("m_fVolume")),
        "turnover": price_to_int(record.get("m_fAmount")),
        "source": settings.service.source_name,
        "timestamp": current_timestamp(),
    }


def _build_minute_msg(symbol: str, record: dict) -> dict:
    return {
        "symbol": symbol,
        "trade_time": format_epoch_timestamp(record.get("m_time")),
        "price": price_to_int(record.get("m_fPrice")),
        "volume": to_int(record.get("m_fVolume")),
        "turnover": price_to_int(record.get("m_fAmount")),
        "source": settings.service.source_name,
        "timestamp": current_timestamp(),
    }


def _build_5minute_msg(symbol: str, record: dict) -> dict:
    return {
        "symbol": symbol,
        "trade_time": format_epoch_timestamp(record.get("m_time")),
        "open": price_to_int(record.get("m_fOpen")),
        "high": price_to_int(record.get("m_fHigh")),
        "low": price_to_int(record.get("m_fLow")),
        "close": price_to_int(record.get("m_fClose")),
        "volume": to_int(record.get("m_fVolume")),
        "turnover": price_to_int(record.get("m_fAmount")),
        "active_buy_vol": to_int(record.get("m_fActiveBuyVol")),
        "source": settings.service.source_name,
        "timestamp": current_timestamp(),
    }


def _build_power_msg(symbol: str, record: dict) -> dict:
    return {
        "symbol": symbol,
        "ex_date": format_epoch_date(record.get("m_time")),
        "bonus": price_to_int(record.get("m_fGive")),
        "allotment": price_to_int(record.get("m_fPei")),
        "allotment_price": price_to_int(record.get("m_fPeiPrice")),
        "dividend": price_to_int(record.get("m_fProfit")),
        "source": settings.service.source_name,
        "timestamp": current_timestamp(),
    }


STRUCTURED_FILEDATA_DISPATCH = {
    FILE_HISTORY_EX: ("m_pDay", "topic_filedata_history", _build_history_msg),
    FILE_MINUTE_EX: ("m_pMinute", "topic_filedata_minute", _build_minute_msg),
    FILE_POWER_EX: ("m_pPower", "topic_filedata_power", _build_power_msg),
    FILE_5MINUTE_EX: ("m_p5Min", "topic_filedata_5minute", _build_5minute_msg),
}


def _parse_structured_file_records(record_pointer, packet_num, topic, msg_builder):
    """Parse structured file records and emit normalized messages to the given topic."""
    current_symbol = ""

    for index in range(packet_num):
        record = record_pointer[index]
        if record.m_head.m_dwHeadTag == EKE_HEAD_TAG:
            head_dict = struct_to_dict(record.m_head)
            market_code = head_dict.get("m_wMarket", 0)
            raw_label = head_dict.get("m_szLabel", "")
            current_symbol = normalize_symbol(market_code, raw_label)
            continue

        if not current_symbol:
            continue

        record_dict = struct_to_dict(record)
        record_dict.pop("m_head", None)
        msg = msg_builder(current_symbol, record_dict)
        enqueue_kafka_message(topic, msg)


def parse_filedata_message(lparam):
    header_ptr = ctypes.cast(lparam, ctypes.POINTER(RCV_DATA))
    if not header_ptr:
        return
    header = header_ptr.contents
    if not header.data_union.m_pData or header.m_wDataType == FILE_TYPE_RES:
        emit_system_event(
            "WARNING", "invalid_filedata",
            {"m_wDataType": int(header.m_wDataType), "m_nPacketNum": int(header.m_nPacketNum)},
        )
        return

    dispatch = STRUCTURED_FILEDATA_DISPATCH.get(header.m_wDataType)
    if dispatch is not None:
        pointer_attr, topic_attr, msg_builder = dispatch
        topic = getattr(settings.kafka, topic_attr)
        _parse_structured_file_records(
            record_pointer=getattr(header.data_union, pointer_attr),
            packet_num=header.m_nPacketNum,
            topic=topic,
            msg_builder=msg_builder,
        )
    elif header.m_wDataType in (FILE_BASE_EX, FILE_NEWS_EX, FILE_HTML_EX):
        pass
    else:
        emit_system_event(
            "WARNING", "unknown_filedata_type",
            {"m_wDataType": int(header.m_wDataType)},
        )
