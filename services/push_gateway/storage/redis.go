package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"push_gateway/config"
)

var ErrNotFound = errors.New("resource not found")

type RedisStore struct {
	cli *redis.Client
	cfg config.RedisConfig
}

func NewRedis(cfg config.RedisConfig) (*RedisStore, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		DB:           cfg.DB,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()
	if err := cli.Ping(ctx).Err(); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisStore{cli: cli, cfg: cfg}, nil
}

func (r *RedisStore) Ping(ctx context.Context) error {
	return r.cli.Ping(ctx).Err()
}

func (r *RedisStore) Close() error { return r.cli.Close() }

// Client 返回底层 redis 客户端，仅供需要原生 client 的模块（如 latency reporter）使用。
func (r *RedisStore) Client() *redis.Client { return r.cli }

// GetQuote 读取 hash 形式的行情快照（HGETALL）。
func (r *RedisStore) GetQuote(ctx context.Context, symbol string) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.Quote(symbol))
}

// GetQuotes 用 pipeline 批量 HGETALL，缺失的 symbol 不出现在结果中。
func (r *RedisStore) GetQuotes(ctx context.Context, symbols []string) (map[string]map[string]string, error) {
	if len(symbols) == 0 {
		return map[string]map[string]string{}, nil
	}
	pipe := r.cli.Pipeline()
	cmds := make(map[string]*redis.MapStringStringCmd, len(symbols))
	for _, sym := range symbols {
		cmds[sym] = pipe.HGetAll(ctx, r.cfg.Quote(sym))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("redis pipeline exec failed: %w", err)
	}
	out := make(map[string]map[string]string, len(symbols))
	for sym, cmd := range cmds {
		m, err := cmd.Result()
		if err != nil {
			continue
		}
		if len(m) == 0 {
			continue
		}
		out[sym] = m
	}
	return out, nil
}

// SetQuoteHash 写 hash 形式的行情快照缓存（DEL+HSET+EXPIRE，原子下发）。
func (r *RedisStore) SetQuoteHash(ctx context.Context, symbol string, fields map[string]string, ttl time.Duration) error {
	if len(fields) == 0 {
		return nil
	}
	key := r.cfg.Quote(symbol)
	pipe := r.cli.Pipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, fields)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis set quote hash %s failed: %w", symbol, err)
	}
	return nil
}

func (r *RedisStore) GetIndicator(ctx context.Context, symbol string) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.Indicator(symbol))
}

func (r *RedisStore) GetTech(ctx context.Context, symbol string) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.Tech(symbol))
}

func (r *RedisStore) GetCapital(ctx context.Context, symbol string) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.Capital(symbol))
}

func (r *RedisStore) GetFinance(ctx context.Context, symbol string) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.Finance(symbol))
}

func (r *RedisStore) GetStockList(ctx context.Context) (map[string]string, error) {
	return r.hgetAll(ctx, r.cfg.StockListKey)
}

func (r *RedisStore) GetFenbi(ctx context.Context, symbol string, limit int64) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("fenbi limit must be positive, got %d", limit)
	}
	key := r.cfg.Fenbi(symbol)
	items, err := r.cli.LRange(ctx, key, -limit, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lrange %s failed: %w", key, err)
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

func (r *RedisStore) GetHistDaily(ctx context.Context, symbol string, limit int64) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("hist daily limit must be positive, got %d", limit)
	}
	key := r.cfg.HistDaily(symbol)
	items, err := r.cli.LRange(ctx, key, -limit, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lrange %s failed: %w", key, err)
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

func (r *RedisStore) hgetAll(ctx context.Context, key string) (map[string]string, error) {
	m, err := r.cli.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall %s failed: %w", key, err)
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	return m, nil
}

func (r *RedisStore) CountByPrefix(ctx context.Context, prefix string, maxScan int) (count int64, truncated bool, err error) {
	if prefix == "" {
		return 0, false, fmt.Errorf("empty prefix")
	}
	match := prefix + "*"
	var cursor uint64
	for {
		keys, next, scanErr := r.cli.Scan(ctx, cursor, match, 500).Result()
		if scanErr != nil {
			return count, false, fmt.Errorf("redis scan %s failed: %w", match, scanErr)
		}
		count += int64(len(keys))
		if maxScan > 0 && count >= int64(maxScan) {
			return count, true, nil
		}
		if next == 0 {
			return count, false, nil
		}
		cursor = next
	}
}

func (r *RedisStore) HLen(ctx context.Context, key string) (int64, error) {
	n, err := r.cli.HLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis hlen %s failed: %w", key, err)
	}
	return n, nil
}

func (r *RedisStore) HGetAllMap(ctx context.Context, key string) (map[string]string, error) {
	m, err := r.cli.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall %s failed: %w", key, err)
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}
