"""时间工具（精简版）：仅保留业务真在用的 ``hhmmss_to_time``。

原模块里 ``to_iso8601 / dll_epoch_to_ms / is_in_trading_session / is_trading_day``
在 stream_engine 非测试代码中零调用，已随本次简化删除。
"""

from __future__ import annotations

import datetime as dt
from typing import Final

HHMMSS_MAX: Final[int] = 235959


def hhmmss_to_time(v: int) -> dt.time:
    """把分笔 ``m_lTime`` 形式的整数（HHMMSS）转为 :class:`datetime.time`。

    例如 ``93015`` → ``time(9, 30, 15)``；非 int / 越界抛异常。
    """
    if not isinstance(v, int) or isinstance(v, bool):
        raise TypeError(f"hhmmss must be int, got {type(v).__name__}")
    if v < 0 or v > HHMMSS_MAX:
        raise ValueError(f"hhmmss out of range [0, {HHMMSS_MAX}]: {v}")
    hour, rem = divmod(v, 10000)
    minute, second = divmod(rem, 100)
    return dt.time(hour=hour, minute=minute, second=second)
