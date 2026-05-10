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
	TradeDate string          `db:"trade_date" json:"trade_date"`
	Open      decimal.Decimal `db:"open" json:"open"`
	High      decimal.Decimal `db:"high" json:"high"`
	Low       decimal.Decimal `db:"low" json:"low"`
	Close     decimal.Decimal `db:"close" json:"close"`
	Volume    int64           `db:"volume" json:"volume"`
	Turnover  decimal.Decimal `db:"turnover" json:"turnover"`
}

type Stock5MinKlineRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	TradeTime string          `db:"trade_time" json:"trade_time"`
	Open      decimal.Decimal `db:"open" json:"open"`
	High      decimal.Decimal `db:"high" json:"high"`
	Low       decimal.Decimal `db:"low" json:"low"`
	Close     decimal.Decimal `db:"close" json:"close"`
	Volume    int64           `db:"volume" json:"volume"`
	Turnover  decimal.Decimal `db:"turnover" json:"turnover"`
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

type SignalHistoryRow struct {
	ID         int64     `db:"id" json:"id"`
	SignalID   string    `db:"signal_id" json:"signal_id"`
	SignalType string    `db:"signal_type" json:"signal_type"`
	Symbol     string    `db:"symbol" json:"symbol"`
	Severity   string    `db:"severity" json:"severity"`
	Summary    string    `db:"summary" json:"summary"`
	Indicators string    `db:"indicators" json:"indicators"`
	SignalTime time.Time `db:"signal_time" json:"signal_time"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type DailyCapitalFlowRow struct {
	Symbol    string          `db:"symbol" json:"symbol"`
	TradeDate time.Time       `db:"trade_date" json:"trade_date"`
	BigBuy    decimal.Decimal `db:"big_buy" json:"big_buy"`
	BigSell   decimal.Decimal `db:"big_sell" json:"big_sell"`
	NetInflow decimal.Decimal `db:"net_inflow" json:"net_inflow"`
}

type SignalFilter struct {
	Symbol     string
	SignalType string
	Severity   string
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
	query += fmt.Sprintf(` ORDER BY trade_date ASC LIMIT $%d`, argIdx)
	args = append(args, clampLimit(limit, 1000))

	var rows []StockDailyKlineRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_daily_kline failed: %w", err)
	}
	return rows, nil
}

func (m *PostgresStore) Query5MinKline(ctx context.Context, symbol string, start, end time.Time, limit int) ([]Stock5MinKlineRow, error) {
	query := `SELECT symbol, trade_time, open, high, low, close, volume, turnover
              FROM stock_5min_kline WHERE symbol = $1`
	args := []interface{}{symbol}
	argIdx := 2
	if !start.IsZero() {
		query += fmt.Sprintf(` AND trade_time >= $%d`, argIdx)
		args = append(args, start)
		argIdx++
	}
	if !end.IsZero() {
		query += fmt.Sprintf(` AND trade_time <= $%d`, argIdx)
		args = append(args, end)
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY trade_time ASC LIMIT $%d`, argIdx)
	args = append(args, clampLimit(limit, 1000))

	var rows []Stock5MinKlineRow
	if err := m.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("query stock_5min_kline failed: %w", err)
	}
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

func (m *PostgresStore) QuerySignals(ctx context.Context, f SignalFilter) ([]SignalHistoryRow, int64, error) {
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
	if f.From != nil {
		where += fmt.Sprintf(" AND signal_time >= $%d", argIdx)
		args = append(args, *f.From)
		argIdx++
	}
	if f.To != nil {
		where += fmt.Sprintf(" AND signal_time <= $%d", argIdx)
		args = append(args, *f.To)
		argIdx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM signal_history" + where
	if err := m.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count signal_history failed: %w", err)
	}

	page := f.Page
	if page <= 0 {
		page = 1
	}
	pageSize := clampLimit(f.PageSize, 200)
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(`SELECT id, signal_id, signal_type, symbol, severity, summary, indicators, signal_time, created_at
                  FROM signal_history%s ORDER BY signal_time DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)

	var rows []SignalHistoryRow
	if err := m.db.SelectContext(ctx, &rows, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("query signal_history failed: %w", err)
	}
	return rows, total, nil
}

func (m *PostgresStore) QuerySignalByID(ctx context.Context, id int64) (*SignalHistoryRow, error) {
	var row SignalHistoryRow
	err := m.db.GetContext(ctx, &row,
		`SELECT id, signal_id, signal_type, symbol, severity, summary, indicators, signal_time, created_at
         FROM signal_history WHERE id = $1 LIMIT 1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query signal_history by id failed: %w", err)
	}
	return &row, nil
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
	"stock_info":         true,
	"stock_daily_kline":  true,
	"stock_5min_kline":   true,
	"stock_minute_kline": true,
	"stock_power":        true,
	"stock_finance":      true,
	"daily_capital_flow": true,
	"signal_history":     true,
}
