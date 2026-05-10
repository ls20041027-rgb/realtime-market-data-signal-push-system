"""Redis connection pool and read helpers."""

from __future__ import annotations

import json
import threading
from typing import Any

import redis
from redis.connection import ConnectionPool

from config import settings
from utils.logger import get_logger

log = get_logger(__name__)

pool: ConnectionPool = None
client: redis.Redis = None
lock = threading.Lock()




def get_redis() -> redis.Redis:
    """返回进程内共享的 Redis 客户端；首次调用会做一次 PING 探活，失败直接抛。"""
    global pool, client
    if client is not None:
        return client
    with lock:
        if client is not None:
            return client
        cfg = settings.redis
        new_pool = ConnectionPool(
            host=cfg.host,
            port=cfg.port,
            db=cfg.db,
            password=cfg.password,
            max_connections=cfg.max_connections,
            socket_timeout=cfg.socket_timeout_seconds,
            socket_connect_timeout=cfg.socket_connect_timeout_seconds,
            health_check_interval=cfg.health_check_interval_seconds,
            decode_responses=True,
        )
        new_client = redis.Redis(connection_pool=new_pool)
        try:
            new_client.ping()
        except redis.RedisError:
            log.exception("redis ping failed on startup", host=cfg.host, port=cfg.port)
            try:
                new_pool.disconnect()
            except Exception:
                log.exception("redis pool disconnect failed during rollback")
            raise
        pool = new_pool
        client = new_client
        log.info("redis client initialized", host=cfg.host, port=cfg.port, db=cfg.db)
        return client


def reset_redis() -> None:
    """释放连接池与客户端（测试 / 热重载使用）。"""
    global pool, client
    with lock:
        if pool is not None:
            try:
                pool.disconnect()
            except Exception:
                log.exception("redis pool disconnect failed on reset")
        pool = None
        client = None
        log.info("redis client reset")



if __name__ == "__main__":
    log.info("redis smoke ping result", pong=get_redis().ping())
