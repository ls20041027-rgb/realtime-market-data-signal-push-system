"""Message utilities: timestamp formatting, symbol normalization, flat message building, queue enqueue.

Flat message design: no nested 'payload' wrapper. Each message is a fully normalized
flat dict with semantic field names, ready for Pathway format="json" ingestion.
All normalization (symbol, precision, time, array expansion) happens HERE at ingestion
time so that the stream engine never needs Python UDFs.
"""

import datetime
import queue
import uuid
from decimal import Decimal, ROUND_HALF_UP

import pytz

from config import settings
from producer.kafka_producer import data_queue

CHINA_TZ = pytz.timezone(settings.service.timezone)

_SH_CODES = {
    int.from_bytes(b"HS", "little"),
    int.from_bytes(b"HS", "big"),
}
_SZ_CODES = {
    int.from_bytes(b"ZS", "little"),
    int.from_bytes(b"ZS", "big"),
}


def normalize_symbol(market_code, raw_label: str) -> str:
    """Convert (m_wMarket, m_szLabel) → 'SH600000' style symbol.

    Returns empty string if market_code is unknown or label is empty.
    """
    if raw_label is None:
        return ""
    if isinstance(raw_label, (bytes, bytearray)):
        raw_label = raw_label.decode("ascii", errors="ignore")
    label = raw_label.strip().strip("\x00").upper()
    if not label:
        return ""
    code = int(market_code) if market_code is not None else 0
    if code in _SH_CODES:
        return f"SH{label}"
    if code in _SZ_CODES:
        return f"SZ{label}"
    return ""


def detect_exchange(symbol: str) -> str:
    """From prefix-style symbol, return exchange enum (SSE/SZSE/BSE)."""
    if not symbol or len(symbol) < 2:
        return ""
    prefix = symbol[:2].upper()
    if prefix == "SH":
        return "SSE"
    if prefix == "SZ":
        return "SZSE"
    if prefix == "BJ":
        return "BSE"
    return ""


PRICE_SCALE = 10000


def price_to_int(value, places: int = 4) -> int:
    """Quantize then scale to integer: Decimal(value).quantize(places) * PRICE_SCALE → int.

    Eliminates float32 tail noise AND outputs a plain int that Pathway can ingest directly.
    ``places`` controls quantization precision before scaling (default 4 covers 0.001 tick).
    """
    if value is None:
        return 0
    quant = Decimal(10) ** -places
    d = Decimal(str(value)).quantize(quant, rounding=ROUND_HALF_UP)
    return int(d * PRICE_SCALE)


def to_int(value) -> int:
    """Convert float-encoded integer (e.g. volume, shares) to int."""
    if value is None:
        return 0
    return int(Decimal(str(value)))


def current_timestamp() -> int:
    """Return current time as Unix epoch seconds (int)."""
    return int(datetime.datetime.now(CHINA_TZ).timestamp())


def format_epoch_timestamp(raw_timestamp) -> int:
    """Pass through epoch seconds as int. Returns 0 for invalid input."""
    if raw_timestamp in (None, 0, ""):
        return 0
    try:
        return int(raw_timestamp)
    except (TypeError, ValueError):
        return 0


def format_epoch_date(raw_timestamp) -> int:
    """Pass through epoch seconds as int (day-aligned). Returns 0 for invalid input."""
    if raw_timestamp in (None, 0, ""):
        return 0
    try:
        return int(raw_timestamp)
    except (TypeError, ValueError):
        return 0


def extract_symbol(raw_payload):
    if not isinstance(raw_payload, dict):
        return ""
    for field_name in ("m_szLabel", "symbol"):
        value = raw_payload.get(field_name)
        if value not in (None, ""):
            return str(value)
    nested_security = raw_payload.get("security")
    if isinstance(nested_security, dict):
        value = nested_security.get("m_szLabel")
        if value not in (None, ""):
            return str(value)
    return ""


def build_market_symbol(market_code):
    if market_code in (None, ""):
        return ""
    return f"MARKET_{market_code}"


def enqueue_kafka_message(topic, message):
    try:
        data_queue.put_nowait({"topic": topic, "message": message})
    except queue.Full:
        print(f"[WARN] queue full, topic={topic} message dropped")


def emit_system_event(level, event_type, detail):
    normalized_level = str(level).lower()
    event_message = {
        "event_id": f"evt-{settings.service.source_name}-{uuid.uuid4().hex[:12]}",
        "service": settings.service.source_name,
        "level": normalized_level,
        "event_type": event_type,
        "message": f"{event_type} reported by {settings.service.source_name}.",
        "details": detail,
        "symbol": "SYSTEM",
        "timestamp": current_timestamp(),
        "source": settings.service.source_name,
    }
    print(f"[{level}] {event_type}: {detail}")
    enqueue_kafka_message(settings.kafka.topic_system_events, event_message)
