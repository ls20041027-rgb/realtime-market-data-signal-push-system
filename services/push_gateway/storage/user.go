package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type UserRow struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// EnsureUserTable creates the users table if it does not exist.
func (m *PostgresStore) EnsureUserTable(ctx context.Context) error {
	ddl := `CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(256) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`
	_, err := m.db.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("ensure users table: %w", err)
	}
	return nil
}

func (m *PostgresStore) CreateUser(ctx context.Context, username, passwordHash string) (*UserRow, error) {
	var user UserRow
	err := m.db.QueryRowxContext(ctx,
		`INSERT INTO users (username, password_hash) VALUES ($1, $2)
		 RETURNING id, username, password_hash, created_at, updated_at`,
		username, passwordHash).StructScan(&user)
	if err != nil {
		return nil, fmt.Errorf("create user failed: %w", err)
	}
	return &user, nil
}

func (m *PostgresStore) QueryUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	var user UserRow
	err := m.db.GetContext(ctx, &user,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = $1`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by username failed: %w", err)
	}
	return &user, nil
}

func (m *PostgresStore) QueryUserByID(ctx context.Context, id int64) (*UserRow, error) {
	var user UserRow
	err := m.db.GetContext(ctx, &user,
		`SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id failed: %w", err)
	}
	return &user, nil
}
