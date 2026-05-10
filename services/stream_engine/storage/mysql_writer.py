"""
PostgreSQL 写入门面（对应 CONTRACT.md §四 PostgreSQL 表清单）。

* 业务代码一律通过 :func:`get_session` 获取会话；禁止直接 ``create_engine()``（R6）。
* upsert 策略：采用 ORM 层 "SELECT → UPDATE / INSERT" 实现，跨 PostgreSQL / SQLite
  一致，便于 4.3 测试跑 SQLite in-memory。生产环境（PostgreSQL）下单次 upsert 两条 SQL
  的开销完全可接受（低频：finance / stock_info / stock_power / signal_history）。
* 批量 K 线使用 ``session.execute(Model.__table__.insert(), rows)``，按
  ``settings.postgres.batch_insert_size`` 分批 flush（R3：批大小来自 config）。
* 金额 / 股本 / 比率参数一律接受 :class:`Decimal`（R2）；字段缺失 → ``None`` 透传到
  ORM，由字段 ``nullable`` 决定是否报错。
"""

from __future__ import annotations

import threading
from contextlib import contextmanager
from datetime import date
from decimal import Decimal
from typing import Any, Iterator

from sqlalchemy import Engine, create_engine
from sqlalchemy.orm import Session, sessionmaker

from config import settings
from storage.models import DailyCapitalFlow
from utils.logger import get_logger

log = get_logger(__name__)


engine_instance: Engine = None
SessionLocal: sessionmaker[Session] = None
lock = threading.Lock()


def build_engine() -> Engine:
    """Build SQLAlchemy Engine from :mod:`config.settings.postgres`."""
    cfg = settings.postgres
    return create_engine(
        cfg.sqlalchemy_url,
        pool_size=cfg.pool_size,
        max_overflow=cfg.max_overflow,
        pool_recycle=cfg.pool_recycle_seconds,
        pool_pre_ping=cfg.pool_pre_ping,
        echo=cfg.echo_sql,
        future=True,
    )


def get_engine() -> Engine:
    """返回进程内共享的 :class:`Engine` 单例。"""
    global engine_instance, SessionLocal

    if engine_instance is not None:
        return engine_instance

    with lock:
        if engine_instance is not None:
            return engine_instance

        engine = build_engine()
        engine_instance = engine
        SessionLocal = sessionmaker(
            bind=engine,
            autoflush=False,
            autocommit=False,
            expire_on_commit=False,
            class_=Session,
        )
        log.info(
            "postgres engine initialized",
            host=settings.postgres.host,
            port=settings.postgres.port,
            database=settings.postgres.database,
            pool_size=settings.postgres.pool_size,
        )
        return engine_instance


def set_engine(engine: Engine) -> None:
    """测试专用：手动注入一个 Engine（例如 SQLite in-memory）。

    同时初始化 SessionLocal；再次调用 :func:`get_engine` 将直接返回该 engine。
    """
    global engine_instance, SessionLocal
    with lock:
        engine_instance = engine
        SessionLocal = sessionmaker(
            bind=engine,
            autoflush=False,
            autocommit=False,
            expire_on_commit=False,
            class_=Session,
        )


def reset_engine() -> None:
    """Release current Engine & SessionLocal. For test / hot-reload only."""
    global engine_instance, SessionLocal
    with lock:
        if engine_instance is not None:
            try:
                engine_instance.dispose()
            except Exception:
                log.exception("postgres engine dispose failed on reset")
        engine_instance = None
        SessionLocal = None
        log.info("postgres engine reset")


@contextmanager
def get_session() -> Iterator[Session]:
    """会话上下文管理器。正常 commit / 异常 rollback + 抛出（R4：不吞异常）。"""
    get_engine()
    assert SessionLocal is not None, "SessionLocal must be initialized after get_engine()"

    session = SessionLocal()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        log.exception("postgres session rolled back")
        raise
    finally:
        session.close()


def insert_daily_capital_flow(
    symbol: str,
    trade_date: date,
    big_buy: Decimal,
    big_sell: Decimal,
    net_inflow: Decimal,
) -> None:
    """写入 / 更新 ``daily_capital_flow``（盘后 15:05 归档）。"""
    with get_session() as session:
        row = session.get(DailyCapitalFlow, (symbol, trade_date))
        if row is None:
            session.add(
                DailyCapitalFlow(
                    symbol=symbol,
                    trade_date=trade_date,
                    big_buy=big_buy,
                    big_sell=big_sell,
                    net_inflow=net_inflow,
                )
            )
        else:
            row.big_buy = big_buy
            row.big_sell = big_sell
            row.net_inflow = net_inflow
