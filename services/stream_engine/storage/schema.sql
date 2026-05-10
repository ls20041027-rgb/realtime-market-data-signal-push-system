-- =====================================================================
-- stream_engine PostgreSQL 建表脚本
--
-- 本脚本必须与 storage/models.py 字段级严格一致（R6），修改前需人工审定
-- （AI_CODING_RULES.md 第 10 条）。
--
-- 约定：
--   * 所有金额 / 价格 / 比率 / 股本数量：NUMERIC(20, 4)，禁止 FLOAT / DOUBLE
--   * 成交量（股数）：BIGINT
--   * symbol 主键统一：VARCHAR(16)
--
-- 使用方式：
--   psql -U postgres -d tornado_seeker -f services/stream_engine/storage/schema.sql
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1. stock_daily_kline —— 日 K 线
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_daily_kline (
    symbol      VARCHAR(16)    NOT NULL,
    trade_date  DATE           NOT NULL,
    open        NUMERIC(20, 4) NOT NULL,
    high        NUMERIC(20, 4) NOT NULL,
    low         NUMERIC(20, 4) NOT NULL,
    close       NUMERIC(20, 4) NOT NULL,
    volume      BIGINT         NOT NULL,
    turnover    NUMERIC(20, 4) NOT NULL,
    PRIMARY KEY (symbol, trade_date)
);

-- ---------------------------------------------------------------------
-- 2. stock_5min_kline —— 5 分钟 K 线
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_5min_kline (
    symbol      VARCHAR(16)    NOT NULL,
    trade_time  TIMESTAMP      NOT NULL,
    open        NUMERIC(20, 4) NOT NULL,
    high        NUMERIC(20, 4) NOT NULL,
    low         NUMERIC(20, 4) NOT NULL,
    close       NUMERIC(20, 4) NOT NULL,
    volume      BIGINT         NOT NULL,
    turnover    NUMERIC(20, 4) NOT NULL,
    PRIMARY KEY (symbol, trade_time)
);

-- ---------------------------------------------------------------------
-- 2b. stock_minute_kline —— 分时（1 分钟粒度）明细
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_minute_kline (
    symbol      VARCHAR(16)    NOT NULL,
    trade_time  TIMESTAMP      NOT NULL,
    price       NUMERIC(20, 4) NOT NULL,
    volume      BIGINT         NOT NULL,
    turnover    NUMERIC(20, 4) NOT NULL,
    PRIMARY KEY (symbol, trade_time)
);

-- ---------------------------------------------------------------------
-- 3. stock_finance —— 财务数据
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_finance (
    symbol        VARCHAR(16)    NOT NULL,
    report_date   DATE           NOT NULL,
    total_shares  NUMERIC(20, 4) NOT NULL,
    float_shares  NUMERIC(20, 4) NOT NULL,
    eps           NUMERIC(20, 4),
    bps           NUMERIC(20, 4),
    net_profit    NUMERIC(20, 4),
    PRIMARY KEY (symbol, report_date)
);

-- ---------------------------------------------------------------------
-- 4. stock_info —— 证券基础信息
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_info (
    symbol    VARCHAR(16)  NOT NULL,
    name      VARCHAR(64)  NOT NULL,
    exchange  VARCHAR(8)   NOT NULL,
    lot_size  INT          NOT NULL DEFAULT 100,
    PRIMARY KEY (symbol)
);

-- ---------------------------------------------------------------------
-- 5. stock_power —— 除权除息
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_power (
    symbol           VARCHAR(16)    NOT NULL,
    ex_date          DATE           NOT NULL,
    bonus            NUMERIC(20, 4) NOT NULL DEFAULT 0,
    allotment        NUMERIC(20, 4) NOT NULL DEFAULT 0,
    allotment_price  NUMERIC(20, 4),
    dividend         NUMERIC(20, 4) NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, ex_date)
);

-- ---------------------------------------------------------------------
-- 6. daily_capital_flow —— 日度资金流向
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS daily_capital_flow (
    symbol      VARCHAR(16)    NOT NULL,
    trade_date  DATE           NOT NULL,
    big_buy     NUMERIC(20, 4) NOT NULL DEFAULT 0,
    big_sell    NUMERIC(20, 4) NOT NULL DEFAULT 0,
    net_inflow  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, trade_date)
);

-- ---------------------------------------------------------------------
-- 7. signal_history —— 策略信号历史
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS signal_history (
    id           BIGSERIAL    NOT NULL,
    signal_id    VARCHAR(64)  NOT NULL,
    signal_type  VARCHAR(32)  NOT NULL,
    symbol       VARCHAR(16)  NOT NULL,
    severity     VARCHAR(16)  NOT NULL,
    summary      VARCHAR(255) NOT NULL,
    indicators   JSONB,
    signal_time  TIMESTAMP    NOT NULL,
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT uq_signal_history_signal_id UNIQUE (signal_id)
);

CREATE INDEX IF NOT EXISTS ix_signal_history_symbol ON signal_history (symbol);
CREATE INDEX IF NOT EXISTS ix_signal_history_signal_time ON signal_history (signal_time);

-- ---------------------------------------------------------------------
-- 8. stock_signal —— 实时信号（纯 Pathway 产出，替代 signal_history 的实时部分）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_signal (
    signal_type    VARCHAR(32)    NOT NULL,
    symbol         VARCHAR(16)    NOT NULL,
    severity       VARCHAR(8)     NOT NULL,
    action         VARCHAR(8)     NOT NULL,
    strategy_name  VARCHAR(32)    NOT NULL,
    trigger_price  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    reason         TEXT,
    signal_time    VARCHAR(32)    NOT NULL,
    PRIMARY KEY (symbol, signal_type, signal_time)
);

CREATE INDEX IF NOT EXISTS ix_stock_signal_symbol ON stock_signal (symbol);
CREATE INDEX IF NOT EXISTS ix_stock_signal_time ON stock_signal (signal_time);
CREATE INDEX IF NOT EXISTS ix_stock_signal_type ON stock_signal (signal_type);

-- ---------------------------------------------------------------------
-- 9. stock_quote_snapshot —— 行情快照（替代 Redis quote:{symbol} hash）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_quote_snapshot (
    symbol      VARCHAR(16)    NOT NULL,
    last_price  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    prev_close  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    open_price  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    high_price  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    low_price   NUMERIC(20, 4) NOT NULL DEFAULT 0,
    volume      BIGINT         NOT NULL DEFAULT 0,
    turnover    NUMERIC(20, 4) NOT NULL DEFAULT 0,
    event_time  VARCHAR(32)    NOT NULL DEFAULT '',
    PRIMARY KEY (symbol)
);

-- ---------------------------------------------------------------------
-- 10. stock_5min_kline_rt —— 实时 5 分钟 K 线（fenbi 聚合产出）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_5min_kline_rt (
    symbol      VARCHAR(16)    NOT NULL,
    trade_time  VARCHAR(32)    NOT NULL,
    open        NUMERIC(20, 4) NOT NULL DEFAULT 0,
    high        NUMERIC(20, 4) NOT NULL DEFAULT 0,
    low         NUMERIC(20, 4) NOT NULL DEFAULT 0,
    close       NUMERIC(20, 4) NOT NULL DEFAULT 0,
    volume      BIGINT         NOT NULL DEFAULT 0,
    turnover    NUMERIC(20, 4) NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, trade_time)
);

-- ---------------------------------------------------------------------
-- 11. stock_indicator_state —— 有状态指标冷启动（每 symbol 一行，upsert 覆盖）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock_indicator_state (
    symbol      VARCHAR(16)    NOT NULL,
    ema_fast    BIGINT         NOT NULL DEFAULT 0,
    ema_slow    BIGINT         NOT NULL DEFAULT 0,
    dea         BIGINT         NOT NULL DEFAULT 0,
    bar_time    BIGINT         NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol)
);
