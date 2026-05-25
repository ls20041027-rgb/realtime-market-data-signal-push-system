package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type WatchlistRow struct {
	ID        int64  `db:"id" json:"id"`
	UserID    int64  `db:"user_id" json:"user_id"`
	Symbol    string `db:"symbol" json:"symbol"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

// EnsureWatchlistTable creates the user_watchlist table if it does not exist.
func (m *PostgresStore) EnsureWatchlistTable(ctx context.Context) error {
	ddl := `CREATE TABLE IF NOT EXISTS user_watchlist (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		symbol VARCHAR(16) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(user_id, symbol)
	)`
	_, err := m.db.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("ensure user_watchlist table: %w", err)
	}
	return nil
}

// AddToWatchlist inserts a symbol for the given user.
// Returns ErrDuplicate if the (user_id, symbol) pair already exists.
func (m *PostgresStore) AddToWatchlist(ctx context.Context, userID int64, symbol string) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO user_watchlist (user_id, symbol) VALUES ($1, $2)
		 ON CONFLICT(user_id, symbol) DO NOTHING`,
		userID, symbol)
	return err
}

// RemoveFromWatchlist deletes a symbol for the given user.
func (m *PostgresStore) RemoveFromWatchlist(ctx context.Context, userID int64, symbol string) error {
	_, err := m.db.ExecContext(ctx,
		`DELETE FROM user_watchlist WHERE user_id = $1 AND symbol = $2`,
		userID, symbol)
	return err
}

// GetWatchlist returns all symbols for the given user, ordered by created_at ascending.
func (m *PostgresStore) GetWatchlist(ctx context.Context, userID int64) ([]string, error) {
	var symbols []string
	err := m.db.SelectContext(ctx,
		&symbols,
		`SELECT symbol FROM user_watchlist WHERE user_id = $1 ORDER BY created_at ASC`,
		userID)
	if errors.Is(err, sql.ErrNoRows) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get watchlist failed: %w", err)
	}
	return symbols, nil
}

// IsInWatchlist checks whether a symbol exists in the user's watchlist.
func (m *PostgresStore) IsInWatchlist(ctx context.Context, userID int64, symbol string) (bool, error) {
	var count int
	err := m.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM user_watchlist WHERE user_id = $1 AND symbol = $2`,
		userID, symbol)
	if err != nil {
		return false, fmt.Errorf("check watchlist failed: %w", err)
	}
	return count > 0, nil
}
