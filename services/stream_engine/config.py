"""
stream_engine 服务的配置加载层（单一事实源）。

为什么重写：原实现用 pydantic_settings 堆了 7 个 BaseSettings 子类 + 每字段 Field(...)，
424 行里大半是模板。本版用 yaml 单文件 + dataclass，保留对外 API（settings.kafka.xxx /
settings.redis_keys.quote(...) / settings.postgres.sqlalchemy_url / settings.thresholds.kdj_params），
全仓 20+ 个 ``from config import settings`` 引用点零破坏；对齐 R3（配置集中）。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from pathlib import Path
from typing import Any, Tuple

import yaml

DECIMAL_THRESHOLD_FIELDS: set[str] = {
    "big_order_threshold",
    "volume_ratio_threshold",
    "limit_up_warn_ratio",
    "limit_down_warn_ratio",
    "boll_k",
}

CONFIG_PATH: Path = Path(__file__).resolve().parent / "config.yaml"


@dataclass
class KafkaSettings:
    bootstrap_servers: str
    group_id: str
    client_id: str
    topic_report: str
    topic_fenbi: str
    topic_mkttbl: str
    topic_finance: str
    topic_filedata_history: str
    topic_filedata_minute: str
    topic_filedata_5minute: str
    topic_filedata_power: str
    topic_market_data_normalized: str
    topic_trading_signals: str
    topic_system_events: str
    auto_offset_reset: str
    enable_auto_commit: bool
    poll_timeout_seconds: float
    reconnect_wait_seconds: float
    reconnect_max_wait_seconds: float
    producer_linger_ms: int
    producer_acks: str
    producer_flush_timeout_seconds: float


@dataclass
class RedisSettings:
    host: str
    port: int
    db: int
    password: str
    max_connections: int
    socket_timeout_seconds: float
    socket_connect_timeout_seconds: float
    health_check_interval_seconds: int
    default_ttl_seconds: int
    hist_ttl_seconds: int
    signal_dedup_ttl_seconds: int


@dataclass
class RedisKeySettings:
    quote_prefix: str
    indicator_prefix: str
    fenbi_prefix: str
    capital_prefix: str
    tech_prefix: str
    hist_daily_prefix: str
    finance_prefix: str
    stock_list: str
    signal_dedup_prefix: str
    tempo_ts_prefix: str

    def quote(self, symbol: str) -> str:
        return f"{self.quote_prefix}{symbol}"

    def indicator(self, symbol: str) -> str:
        return f"{self.indicator_prefix}{symbol}"

    def fenbi(self, symbol: str) -> str:
        return f"{self.fenbi_prefix}{symbol}"

    def capital(self, symbol: str) -> str:
        return f"{self.capital_prefix}{symbol}"

    def tech(self, symbol: str) -> str:
        return f"{self.tech_prefix}{symbol}"

    def hist_daily(self, symbol: str) -> str:
        return f"{self.hist_daily_prefix}{symbol}"

    def finance(self, symbol: str) -> str:
        return f"{self.finance_prefix}{symbol}"

    def signal_dedup(self, symbol: str, signal_type: str) -> str:
        return f"{self.signal_dedup_prefix}{symbol}:{signal_type}"

    def tempo_ts(self, symbol: str) -> str:
        return f"{self.tempo_ts_prefix}{symbol}"


@dataclass
class PostgresSettings:
    host: str
    port: int
    user: str
    password: str
    database: str
    pool_size: int
    max_overflow: int
    pool_recycle_seconds: int
    pool_pre_ping: bool
    echo_sql: bool
    batch_insert_size: int

    @property
    def sqlalchemy_url(self) -> str:
        pwd = f":{self.password}" if self.password else ""
        return (
            f"postgresql+psycopg2://{self.user}{pwd}@{self.host}:{self.port}/"
            f"{self.database}"
        )


@dataclass
class ThresholdSettings:
    big_order_threshold: Decimal
    volume_ratio_threshold: Decimal
    limit_up_warn_ratio: Decimal
    limit_down_warn_ratio: Decimal
    boll_k: Decimal
    big_order_burst_count: int
    tempo_window_seconds: int
    ma_periods: Tuple[int, ...]
    rsi_period: int
    kdj_n: int
    kdj_m1: int
    kdj_m2: int
    boll_period: int
    macd_fast: int
    macd_slow: int
    macd_signal: int
    hist_daily_lookback: int
    fenbi_max_len: int

    @property
    def kdj_params(self) -> Tuple[int, int, int]:
        return (self.kdj_n, self.kdj_m1, self.kdj_m2)


@dataclass
class RuntimeSettings:
    service_name: str
    log_level: str
    log_file: str
    log_max_bytes: int
    log_backup_count: int
    timezone: str
    morning_session_start: str
    morning_session_end: str
    afternoon_session_start: str
    afternoon_session_end: str
    metrics_interval_seconds: float
    metrics_enabled: bool

@dataclass
class PathwaySettings:
    autocommit_duration_ms: int



def load_yaml() -> dict[str, Any]:
    """读 config.yaml。文件缺失 / 解析失败一律 fail-fast，不兜底默认值。"""
    if not CONFIG_PATH.is_file():
        raise FileNotFoundError(f"config.yaml not found at {CONFIG_PATH}")
    with CONFIG_PATH.open("r", encoding="utf-8") as fh:
        data = yaml.safe_load(fh)
    if not isinstance(data, dict):
        raise ValueError(f"config.yaml root must be a mapping, got {type(data).__name__}")
    return data


def cast(raw: dict[str, Any]) -> dict[str, Any]:
    """yaml 原生类型到 Python 业务类型的最小化转换。

    为什么要转：yaml 没有 Decimal / tuple，下游 Decimal 运算会因类型不匹配抛
    TypeError；此处一次性把金额/比率转 Decimal、把 ma_periods 转 tuple。
    """
    th = raw.get("thresholds") or {}
    for name in DECIMAL_THRESHOLD_FIELDS:
        if name in th:
            th[name] = Decimal(str(th[name]))
    if "ma_periods" in th:
        th["ma_periods"] = tuple(int(p) for p in th["ma_periods"])
    return raw


def validate(s: "Settings") -> None:
    """关键字段校验：只检查最易踩坑、出错代价最大的 3 项（用户放弃其余校验）。"""
    if not s.kafka.bootstrap_servers:
        raise ValueError("kafka.bootstrap_servers must not be empty")
    if not s.postgres.database:
        raise ValueError("postgres.database must not be empty")
    if s.thresholds.macd_slow <= s.thresholds.macd_fast:
        raise ValueError(
            f"macd_slow ({s.thresholds.macd_slow}) must be > "
            f"macd_fast ({s.thresholds.macd_fast})"
        )


class Settings:
    """聚合根。模块加载时实例化一次，业务代码只用 ``from config import settings``。"""

    def __init__(self) -> None:
        raw = cast(load_yaml())
        self.kafka: KafkaSettings = KafkaSettings(**raw["kafka"])
        self.redis: RedisSettings = RedisSettings(**raw["redis"])
        self.redis_keys: RedisKeySettings = RedisKeySettings(**raw["redis_keys"])
        self.postgres: PostgresSettings = PostgresSettings(**raw["postgres"])
        self.thresholds: ThresholdSettings = ThresholdSettings(**raw["thresholds"])
        self.runtime: RuntimeSettings = RuntimeSettings(**raw["runtime"])
        self.pathway: PathwaySettings = PathwaySettings(**raw["pathway"])
        validate(self)


settings: Settings = Settings()
