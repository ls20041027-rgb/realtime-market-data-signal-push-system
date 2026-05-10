"""SQLAlchemy 2.0 声明式 ORM 模型（对应 CONTRACT.md §四 PostgreSQL 表清单）。

本模块是 ``stream_engine`` PostgreSQL 侧的**单一字段定义来源**，与
[schema.sql](./schema.sql) 必须保持字段级严格一致（R6）。

约定（与 CONTRACT.md §5.2 对齐）：

* 金额 / 价格 / 比率 / 股本数量一律 ``NUMERIC(20, 4)``，禁止 ``Float`` / ``Double``。
* 成交量（股数）一律 ``BigInteger``。
* ``symbol`` 主键统一 ``VARCHAR(16)``（沪深两市 6 位 + 后缀留裕量）。

表结构变更必须由人工审定（AI_CODING_RULES.md 第 10 条）。
"""
from __future__ import annotations

from datetime import date
from decimal import Decimal

from sqlalchemy import (
    BigInteger,
    Date,
    Integer,
    Numeric,
    String,
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    """所有 stream_engine ORM 模型的基类。"""


MONEY = Numeric(20, 4)


class StockDailyKline(Base):
    """日 K 线表，主键 ``(symbol, trade_date)``。"""

    __tablename__ = "stock_daily_kline"

    symbol: Mapped[str] = mapped_column(String(16), primary_key=True)
    trade_date: Mapped[date] = mapped_column(Date, primary_key=True)

    open: Mapped[Decimal] = mapped_column(MONEY, nullable=False)
    high: Mapped[Decimal] = mapped_column(MONEY, nullable=False)
    low: Mapped[Decimal] = mapped_column(MONEY, nullable=False)
    close: Mapped[Decimal] = mapped_column(MONEY, nullable=False)
    volume: Mapped[int] = mapped_column(BigInteger, nullable=False)
    turnover: Mapped[Decimal] = mapped_column(MONEY, nullable=False)





class StockFinance(Base):
    """财务快照表，主键 ``(symbol, report_date)``。"""

    __tablename__ = "stock_finance"

    symbol: Mapped[str] = mapped_column(String(16), primary_key=True)
    report_date: Mapped[date] = mapped_column(Date, primary_key=True)

    total_shares: Mapped[Decimal] = mapped_column(MONEY, nullable=False)
    float_shares: Mapped[Decimal] = mapped_column(MONEY, nullable=False)

    eps: Mapped[Decimal] = mapped_column(MONEY, nullable=True)
    bps: Mapped[Decimal] = mapped_column(MONEY, nullable=True)
    net_profit: Mapped[Decimal] = mapped_column(MONEY, nullable=True)


class StockInfo(Base):
    """证券基础信息表，主键 ``symbol``。"""

    __tablename__ = "stock_info"

    symbol: Mapped[str] = mapped_column(String(16), primary_key=True)
    name: Mapped[str] = mapped_column(String(64), nullable=False)
    exchange: Mapped[str] = mapped_column(String(8), nullable=False)
    lot_size: Mapped[int] = mapped_column(Integer, nullable=False, default=100)





class DailyCapitalFlow(Base):
    """日度资金流向汇总表，主键 ``(symbol, trade_date)``。"""

    __tablename__ = "daily_capital_flow"

    symbol: Mapped[str] = mapped_column(String(16), primary_key=True)
    trade_date: Mapped[date] = mapped_column(Date, primary_key=True)

    big_buy: Mapped[Decimal] = mapped_column(MONEY, nullable=False, default=Decimal("0"))
    big_sell: Mapped[Decimal] = mapped_column(MONEY, nullable=False, default=Decimal("0"))
    net_inflow: Mapped[Decimal] = mapped_column(MONEY, nullable=False, default=Decimal("0"))



