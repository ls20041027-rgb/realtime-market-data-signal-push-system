"""市场代码映射与交易所判定（对应 CONTRACT.md §2 exchange 枚举）。

本模块解决两个独立问题：

* :func:`normalize_symbol` —— 上游 DLL 回调里的 ``m_wMarket``（``uint16``）+ 原始
  ``m_szLabel`` 两段字段，拼接为 ``"SH600000"`` / ``"SZ000001"`` 这种前缀式 symbol；
* :func:`detect_exchange` —— 从前缀式 symbol 反查 CONTRACT.md 要求的
  ``exchange`` 枚举（``SSE`` / ``SZSE`` / ``BSE``）。

禁止硬编码市场数值（R3）：市场字节常量（``SH_MARKET_EX`` / ``SZ_MARKET_EX``）
在本模块内直接声明，契约同源于
``services/data_ingestion/models/market_protocol.py``；不在运行期跨服务 import
（避免反向依赖上游数据接入层），改由 ``tests/test_market_protocol_contract.py``
做合同检查。一旦上游改了字节值，测试会红。

关于 BSE（北交所）：
    上游数据接入层目前尚未给出 ``BJ_MARKET_EX`` 字节常量，说明当前接入的证券
    里没有北交所标的。故 :func:`normalize_symbol` 的"数值→前缀"映射中暂无 BJ；
    但 :func:`detect_exchange` 已经支持 ``"BJ"`` 前缀，一旦上游加上 BJ 常量即可
    无缝接入。

兼容说明（2026-04-26）：DLL 不同结构体里 ``m_wMarket`` 字符顺序不自洽
（同一个 SH 既可能是 21320 也可能是 18515），在本模块用
``_build_market_code_map`` 一次性登记两种字节序，兜在实时处理层；
不动接入层契约 / ``topic_contract.yaml``（R5 / R6）。
"""

from __future__ import annotations

from typing import Final, Literal

from utils.logger import get_logger

log = get_logger(__name__)

SH_MARKET_EX: Final[bytes] = b"HS"
SZ_MARKET_EX: Final[bytes] = b"ZS"


PREFIX_LEN: Final[int] = 2

PREFIX_SH: Final[str] = "SH"
PREFIX_SZ: Final[str] = "SZ"
PREFIX_BJ: Final[str] = "BJ"

ExchangeName = Literal["SSE", "SZSE", "BSE"]


def to_uint16(market_bytes: bytes) -> int:
    """把 2 字节 market 常量解释为小端 ``uint16``。

    与 ctypes ``c_uint16`` 的内存布局一致（x86 / x64 默认小端）。
    """
    if len(market_bytes) != 2:
        raise ValueError(
            f"market byte constant must be 2 bytes, got {len(market_bytes)}"
        )
    return int.from_bytes(market_bytes, byteorder="little", signed=False)


def to_uint16_be(market_bytes: bytes) -> int:
    """把 2 字节 market 常量按大端 uint16 解释（字符顺序颠倒）。"""
    if len(market_bytes) != 2:
        raise ValueError(
            f"market byte constant must be 2 bytes, got {len(market_bytes)}"
        )
    return int.from_bytes(market_bytes, byteorder="big", signed=False)


def build_market_code_map() -> dict[int, str]:
    """为每个市场字节常量同时登记两种字节序的 uint16，兼容 DLL 不规范写法。

    碰撞时保留先来者并告警，避免前缀被悄悄覆盖（R4 可观测）。
    """
    mapping: dict[int, str] = {}
    for market_bytes, prefix in (
        (SH_MARKET_EX, PREFIX_SH),
        (SZ_MARKET_EX, PREFIX_SZ),
    ):
        for code in (to_uint16(market_bytes), to_uint16_be(market_bytes)):
            existing = mapping.get(code)
            if existing is not None and existing != prefix:
                log.warning(
                    "market_code collision ignored",
                    code=code,
                    existing_prefix=existing,
                    incoming_prefix=prefix,
                )
                continue
            mapping[code] = prefix
    return mapping


MARKET_CODE_TO_PREFIX: Final[dict[int, str]] = build_market_code_map()

PREFIX_TO_EXCHANGE: Final[dict[str, ExchangeName]] = {
    PREFIX_SH: "SSE",
    PREFIX_SZ: "SZSE",
    PREFIX_BJ: "BSE",
}


def normalize_symbol(market_code: int, raw_label: str) -> str:
    """把上游 DLL 的 ``(m_wMarket, m_szLabel)`` 拼为前缀式 symbol。

    Parameters
    ----------
    market_code:
        上游 ``m_wMarket`` 的 ``uint16`` 原值。
    raw_label:
        上游 ``m_szLabel``（可能是 ``bytes``/``str``，含空白或 NUL 填充）。
        本函数会做 ``strip``，但调用方若已经解码好 ``str`` 更佳。

    Returns
    -------
    str
        ``"SH600000"`` 风格；对已知市场命中即返回，未命中或 label 非法抛异常。

    Raises
    ------
    ValueError
        market_code 不在 :data:`_MARKET_CODE_TO_PREFIX` 中、或 label 去除空白后为空。
    """
    if isinstance(raw_label, (bytes, bytearray)):
        raw_label = raw_label.decode("ascii", errors="ignore")

    label = raw_label.strip().strip("\x00").upper()
    if not label:
        raise ValueError(f"empty raw_label for market_code={market_code}")

    prefix = MARKET_CODE_TO_PREFIX.get(int(market_code))
    if prefix is None:
        raise ValueError(
            f"unknown market_code={market_code}, "
            f"supported={sorted(MARKET_CODE_TO_PREFIX)}"
        )
    return f"{prefix}{label}"


def detect_exchange(symbol: str) -> ExchangeName:
    """从前缀式 symbol 反查 CONTRACT.md 要求的 ``exchange`` 枚举。

    只看前两个字符，不校验后续数字长度 —— A 股 6 位、北交 6 位不做硬约束，
    留给调用方根据业务规则再校验。

    Raises
    ------
    ValueError
        symbol 长度不足或前缀不在 :data:`_PREFIX_TO_EXCHANGE` 中。
    """
    if not isinstance(symbol, str) or len(symbol) < PREFIX_LEN:
        raise ValueError(f"invalid symbol: '{symbol}'")
    prefix = symbol[:PREFIX_LEN].upper()
    exchange = PREFIX_TO_EXCHANGE.get(prefix)
    if exchange is None:
        raise ValueError(
            f"unknown prefix '{prefix}' in symbol '{symbol}', "
            f"supported={sorted(PREFIX_TO_EXCHANGE)}"
        )
    return exchange
