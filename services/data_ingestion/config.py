"""
data_ingestion config loader (single source of truth).

Style aligned with stream_engine/config.py: yaml-based, no env-var magic.
All business code uses ``from ..config import settings`` (or relative import).
"""

from pathlib import Path

import yaml

CONFIG_PATH = Path(__file__).resolve().parent / "config.yaml"


class KafkaSettings:
    def __init__(self, bootstrap_servers, topic_report, topic_fenbi,
                 topic_mkttbl, topic_finance,
                 topic_filedata_history, topic_filedata_minute,
                 topic_filedata_5minute, topic_filedata_power,
                 topic_system_events, client_id,
                 batch_size=1048576, linger_ms=50,
                 buffer_memory=67108864, compression_type="lz4",
                 queue_buffering_max_messages=500000):
        self.bootstrap_servers = bootstrap_servers
        self.topic_report = topic_report
        self.topic_fenbi = topic_fenbi
        self.topic_mkttbl = topic_mkttbl
        self.topic_finance = topic_finance
        self.topic_filedata_history = topic_filedata_history
        self.topic_filedata_minute = topic_filedata_minute
        self.topic_filedata_5minute = topic_filedata_5minute
        self.topic_filedata_power = topic_filedata_power
        self.topic_system_events = topic_system_events
        self.client_id = client_id
        self.batch_size = batch_size
        self.linger_ms = linger_ms
        self.buffer_memory = buffer_memory
        self.compression_type = compression_type
        self.queue_buffering_max_messages = queue_buffering_max_messages


class ServiceSettings:
    def __init__(self, source_name, queue_maxsize, timezone,
                 batch_drain_size=500):
        self.source_name = source_name
        self.queue_maxsize = queue_maxsize
        self.batch_drain_size = batch_drain_size
        self.timezone = timezone


class RuntimeSettings:
    def __init__(self, log_level, fenbi_header_size):
        self.log_level = log_level
        self.fenbi_header_size = fenbi_header_size


def load_yaml():
    """Read config.yaml. Missing file / parse failure → fail-fast."""
    if not CONFIG_PATH.is_file():
        raise FileNotFoundError(f"config.yaml not found at {CONFIG_PATH}")
    with CONFIG_PATH.open("r", encoding="utf-8") as fh:
        data = yaml.safe_load(fh)
    if not isinstance(data, dict):
        raise ValueError(f"config.yaml root must be a mapping, got {type(data).__name__}")
    return data


class Settings:
    """Aggregate root. Instantiated once at module load time."""

    def __init__(self):
        raw = load_yaml()
        self.kafka = KafkaSettings(**raw["kafka"])
        self.service = ServiceSettings(**raw["service"])
        self.runtime = RuntimeSettings(**raw["runtime"])


settings = Settings()
