package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"push_gateway/config"
)

type PostgresStore struct {
	db  *sqlx.DB
	cfg config.PostgresConfig
}

func NewPostgres(cfg config.PostgresConfig) (*PostgresStore, error) {
	db, err := sqlx.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("postgres open failed: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var probe int
	if err := db.GetContext(ctx, &probe, "SELECT 1"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres probe failed: %w", err)
	}
	return &PostgresStore{db: db, cfg: cfg}, nil
}

func (m *PostgresStore) Ping(ctx context.Context) error {
	var one int
	return m.db.GetContext(ctx, &one, "SELECT 1")
}

func (m *PostgresStore) Close() error { return m.db.Close() }

type StockInfoRow struct {
	Symbol   string `db:"symbol" json:"symbol"`
	Name     string `db:"name" json:"name"`
	Exchange string `db:"exchange" json:"exchange"`
	LotSize  int    `db:"lot_size" json:"lot_size"`
}

type StockDailyKlineRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	TradeDate int64           `db:"trade_date" json:"trade_date"`
	Open      decimal.Decimal `db:"open" json:"open"`
	High      decimal.Decimal `db:"high" json:"high"`
	Low       decimal.Decimal `db:"low" json:"low"`
	Close     decimal.Decimal `db:"close" json:"close"`
	Volume    int64           `db:"volume" json:"volume"`
	Turnover  decimal.Decimal `db:"turnover" json:"turnover"`
}

type Stock5MinKlineRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	TradeTime int64           `db:"trade_time" json:"trade_time"`
	Open      decimal.Decimal `db:"open" json:"open"`
	High      decimal.Decimal `db:"high" json:"high"`
	Low       decimal.Decimal `db:"low" json:"low"`
	Close     decimal.Decimal `db:"close" json:"close"`
	Volume    int64           `db:"volume" json:"volume"`
	Turnover  decimal.Decimal `db:"turnover" json:"turnover"`
}

type Stock1MinKlineRtRow struct {
	Symbol    string `db:"symbol" json:"symbol"`
	TradeTime int64  `db:"trade_time" json:"trade_time"`
	Close     int64  `db:"close" json:"close"`
	Volume    int64  `db:"volume" json:"volume"`
	Turnover  int64  `db:"turnover" json:"turnover"`
}

type StockFinanceRow struct {
	Symbol      string          `db:"symbol" json:"symbol"`
	ReportDate  string          `db:"report_date" json:"report_date"`
	TotalShares decimal.Decimal `db:"total_shares" json:"total_shares"`
	FloatShares decimal.Decimal `db:"float_shares" json:"float_shares"`
	EPS         decimal.Decimal `db:"eps" json:"eps"`
	BPS         decimal.Decimal `db:"bps" json:"bps"`
	NetProfit   decimal.Decimal `db:"net_profit" json:"net_profit"`
}

type StockSignalRow struct {
	SignalType   string          `db:"signal_type" json:"signal_type"`
	Symbol       string          `db:"symbol" json:"symbol"`
	Severity     string          `db:"severity" json:"severity"`
	Action       string          `db:"action" json:"action"`
	StrategyName string          `db:"strategy_name" json:"strategy_name"`
	TriggerPrice decimal.Decimal `db:"trigger_price" json:"trigger_price"`
	Reason       string          `db:"reason" json:"reason"`
	SignalTime   string          `db:"signal_time" json:"signal_time"`
}

type DailyCapitalFlowRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	TradeDate time.Time       `db:"trade_date" json:"trade_date"`
	BigBuy    decimal.Decimal `db:"big_buy" json:"big_buy"`
	BigSell   decimal.Decimal `db:"big_sell" json:"big_sell"`
	NetInflow decimal.Decimal `db:"net_inflow" json:"net_inflow"`
}

type StockQuoteSnapshotRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	LastPrice decimal.Decimal `db:"last_price" json:"last_price"`
	PrevClose decimal.Decimal `db:"prev_close" json:"prev_close"`
	OpenPrice decimal.Decimal `db:"open_price" json:"open_price"`
	HighPrice decimal.Decimal `db:"high_price" json:"high_price"`
	LowPrice  decimal.Decimal `db:"low_price" json:"low_price"`
	Volume    int64           `db:"volume" json:"volume"`
	Turnover  decimal.Decimal `db:"turnover" json:"turnover"`
	EventTime string          `db:"event_time" json:"event_time"`
}

type SignalFilter struct {
	Symbol     string
	SignalType string
	Severity   string
	Action     string
	From       *time.Time
	To         *time.Time
	Page       int
	PageSize   int
}

func (m *PostgresStore) QueryStockInfo(ctx context.Context, symbol string) (*StockInfoRow, error) {
	var row StockInfoRow
	err := m.db.GetContext(ctx, &row,
		`SELECT symbol, name, exchange, lot_size FROM stock_info WHERE symbol = $1 LIMIT 1`, symbol)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query stock_info failed: %w", err)
	}
	return &row, nil
}

func (m *PostgresStore) QueryStockList(ctx context.Context) ([]StockInfoRow, error) {
	var rows []StockInfoRow
	err := m.db.SelectContext(ctx, &rows,
		`SELECT symbol, name, exchange, lot_size FROM stock_info ORDER BY symbol`)
	if err != nil {
		return nil, fmt.Errorf("query stock_list failed: %w", err)
	}
	return rows, nil
}

func (m *PostgresStore) QueryLatestDailyKlines(ctx context.Context, symbols []string) (map[string]StockDailyKlineRow, error) {
	if len(symbols) == 0 {
		return map[string]StockDailyKlineRow{}, nil
	}
	query, args, err := sqlx.In(
		`SELECT k.symbol, k.trade_date, k.open, k.high, k.low, k.close, k.volume, k.turnover
		 FROM stock_daily_kline k
		 INNER JOIN (
		   SELECT symbol, MAX(trade_date) AS max_date
		   FROM stock_daily_kline
		   WHERE symbol IN (?)
		   GROUP BY symbol
		 ) latest ON k.symbol = latest.symbol AND k.trade_date = latest.max_date`,
		symbols)
	if err != nil {
		return nil, fmt.Errorf("build latest daily kline query failed: %w", err)
	}
	query = m.db.Rebind(query)

	var rows []StockDailyKlineRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query latest daily kline failed: %w", err)
	}
	out := make(map[string]StockDailyKlineRow, len(rows))
	for _, r := range rows {
		out[r.Symbol] = r
	}
	return out, nil
}

func (m *PostgresStore) QueryDailyKline(ctx context.Context, symbol string, start, end time.Time, limit int) ([]StockDailyKlineRow, error) {
	query := `SELECT symbol, trade_date, open, high, low, close, volume, turnover
              FROM stock_daily_kline WHERE symbol = $1`
	args := []interface{}{symbol}
	argIdx := 2
	// trade_date 列为 int8（Unix 秒），需要将 time.Time 转成秒值再绑定。
	if !start.IsZero() {
		query += fmt.Sprintf(` AND trade_date >= $%d`, argIdx)
		args = append(args, start.Unix())
		argIdx++
	}
	if !end.IsZero() {
		query += fmt.Sprintf(` AND trade_date <= $%d`, argIdx)
		args = append(args, end.Unix())
		argIdx++
	}
	// 按时间倒序取最近 N 条，再反转为升序返回，方便前端按 end 锚点向前翻页。
	query += fmt.Sprintf(` ORDER BY trade_date DESC LIMIT $%d`, argIdx)
	args = append(args, clampLimit(limit, 1000))

	var rows []StockDailyKlineRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_daily_kline failed: %w", err)
	}
	reverseRows(rows)
	return rows, nil
}

func (m *PostgresStore) Query5MinKline(ctx context.Context, symbol string, start, end time.Time, limit int) ([]Stock5MinKlineRow, error) {
	query := `SELECT symbol, trade_time, open, high, low, close, volume, turnover
              FROM stock_5min_kline WHERE symbol = $1`
	args := []interface{}{symbol}
	argIdx := 2
	// trade_time 列为 int8（Unix 秒），需要将 time.Time 转成秒值再绑定。
	if !start.IsZero() {
		query += fmt.Sprintf(` AND trade_time >= $%d`, argIdx)
		args = append(args, start.Unix())
		argIdx++
	}
	if !end.IsZero() {
		query += fmt.Sprintf(` AND trade_time <= $%d`, argIdx)
		args = append(args, end.Unix())
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY trade_time DESC LIMIT $%d`, argIdx)
	args = append(args, clampLimit(limit, 1000))

	var rows []Stock5MinKlineRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_5min_kline failed: %w", err)
	}
	reverseRows(rows)
	return rows, nil
}

// Query1MinKlineRt 从 stock_1min_kline_rt 读 1 分钟实时 K 线（fenbi 聚合产出）。
// trade_time 在表里是 unix 秒字符串，10 位长度，字典序等同数值序，可直接字符串比较。
func (m *PostgresStore) Query1MinKlineRt(ctx context.Context, symbol string, start, end time.Time, limit int) ([]Stock1MinKlineRtRow, error) {
	query := `SELECT symbol, trade_time, COALESCE(close,0) as close, COALESCE(volume,0) as volume, COALESCE(turnover,0) as turnover
              FROM stock_1min_kline_rt WHERE symbol = $1`
	args := []interface{}{symbol}
	argIdx := 2
	if !start.IsZero() {
		query += fmt.Sprintf(` AND trade_time >= $%d`, argIdx)
		args = append(args, start.Unix())
		argIdx++
	}
	if !end.IsZero() {
		query += fmt.Sprintf(` AND trade_time <= $%d`, argIdx)
		args = append(args, end.Unix())
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY trade_time DESC LIMIT $%d`, argIdx)
	args = append(args, clampLimit(limit, 1000))

	var rows []Stock1MinKlineRtRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_1min_kline_rt failed: %w", err)
	}
	reverseRows(rows)
	return rows, nil
}

func (m *PostgresStore) QueryFinance(ctx context.Context, symbol string, limit int) ([]StockFinanceRow, error) {
	var rows []StockFinanceRow
	err := m.db.SelectContext(ctx, &rows,
		`SELECT symbol, report_date, total_shares, float_shares, eps, bps, net_profit
         FROM stock_finance WHERE symbol = $1 ORDER BY report_date DESC LIMIT $2`,
		symbol, clampLimit(limit, 40))
	if err != nil {
		return nil, fmt.Errorf("query stock_finance failed: %w", err)
	}
	return rows, nil
}

func (m *PostgresStore) QuerySignals(ctx context.Context, f SignalFilter) ([]StockSignalRow, int64, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	if f.Symbol != "" {
		where += fmt.Sprintf(" AND symbol = $%d", argIdx)
		args = append(args, f.Symbol)
		argIdx++
	}
	if f.SignalType != "" {
		where += fmt.Sprintf(" AND signal_type = $%d", argIdx)
		args = append(args, f.SignalType)
		argIdx++
	}
	if f.Severity != "" {
		where += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, f.Severity)
		argIdx++
	}
	if f.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, f.Action)
		argIdx++
	}
	// stock_signal.signal_time 是 VARCHAR(32)，写入端使用 ISO 字符串，字典序与时间序一致，可直接比较。
	if f.From != nil {
		where += fmt.Sprintf(" AND signal_time >= $%d", argIdx)
		args = append(args, f.From.Format(time.RFC3339))
		argIdx++
	}
	if f.To != nil {
		where += fmt.Sprintf(" AND signal_time <= $%d", argIdx)
		args = append(args, f.To.Format(time.RFC3339))
		argIdx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM stock_signal" + where
	if err := m.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count stock_signal failed: %w", err)
	}

	page := f.Page
	if page <= 0 {
		page = 1
	}
	pageSize := clampLimit(f.PageSize, 200)
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(`SELECT signal_type, symbol, severity, action, strategy_name, trigger_price, reason, signal_time
                  FROM stock_signal%s ORDER BY signal_time DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)

	var rows []StockSignalRow
	if err := m.db.SelectContext(ctx, &rows, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("query stock_signal failed: %w", err)
	}
	return rows, total, nil
}

func (m *PostgresStore) QueryDailyCapitalFlow(ctx context.Context, symbol string, start, end time.Time) ([]DailyCapitalFlowRow, error) {
	query := `SELECT symbol, trade_date, big_buy, big_sell, net_inflow
              FROM daily_capital_flow WHERE symbol = $1`
	args := []interface{}{symbol}
	argIdx := 2
	if !start.IsZero() {
		query += fmt.Sprintf(` AND trade_date >= $%d`, argIdx)
		args = append(args, start)
		argIdx++
	}
	if !end.IsZero() {
		query += fmt.Sprintf(` AND trade_date <= $%d`, argIdx)
		args = append(args, end)
		argIdx++
	}
	query += ` ORDER BY trade_date ASC`

	var rows []DailyCapitalFlowRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query daily_capital_flow failed: %w", err)
	}
	return rows, nil
}

func clampLimit(limit, max int) int {
	if limit <= 0 || limit > max {
		return max
	}
	return limit
}

// reverseRows 原地反转切片，用于将 DESC 查询结果转为 ASC。
func reverseRows[T any](rows []T) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

// QueryQuoteSnapshots 从 stock_quote_snapshot 批量读最新行情快照（替代 Redis quote: 缓存丢失时的回源）。
func (m *PostgresStore) QueryQuoteSnapshots(ctx context.Context, symbols []string) (map[string]StockQuoteSnapshotRow, error) {
	if len(symbols) == 0 {
		return map[string]StockQuoteSnapshotRow{}, nil
	}
	query, args, err := sqlx.In(
		`SELECT symbol, last_price, prev_close, open_price, high_price, low_price,
		        volume, turnover, event_time
		 FROM stock_quote_snapshot
		 WHERE symbol IN (?)`, symbols)
	if err != nil {
		return nil, fmt.Errorf("build quote snapshot query failed: %w", err)
	}
	query = m.db.Rebind(query)

	var rows []StockQuoteSnapshotRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_quote_snapshot failed: %w", err)
	}
	out := make(map[string]StockQuoteSnapshotRow, len(rows))
	for _, r := range rows {
		out[r.Symbol] = r
	}
	return out, nil
}

func (m *PostgresStore) CountTable(ctx context.Context, table string) (int64, error) {
	if !allowedCountTables[table] {
		return 0, fmt.Errorf("table %q not allowed for count", table)
	}
	var n int64
	if err := m.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM "+table); err != nil {
		return 0, fmt.Errorf("count %s failed: %w", table, err)
	}
	return n, nil
}

var allowedCountTables = map[string]bool{
	"stock_info":          true,
	"stock_daily_kline":   true,
	"stock_5min_kline":    true,
	"stock_power":         true,
	"stock_finance":       true,
	"daily_capital_flow":  true,
	"stock_signal":        true,
	"stock_1min_kline_rt": true,
}
