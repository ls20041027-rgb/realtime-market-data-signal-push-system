package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"

	log "push_gateway/internal/log"
	"push_gateway/model"
	"push_gateway/storage"
)

const (
	quotesSymbolsMax  = 200
	fenbiLimitMax     = 500
	fenbiLimitDefault = 100
)

// quoteSF 用于 PG 回源时按 symbol 做请求合并，避免缓存击穿。
var quoteSF singleflight.Group

func handleQuotes(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Query("symbols"))
		if raw == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbols is required"))
			return
		}
		parts := strings.Split(raw, ",")
		symbols := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				symbols = append(symbols, s)
			}
		}
		if len(symbols) == 0 {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbols must contain at least one value"))
			return
		}
		if len(symbols) > quotesSymbolsMax {
			writeError(c, NewApiError(model.CodeInvalidParam,
				"symbols exceeds max "+strconv.Itoa(quotesSymbolsMax)))
			return
		}

		ctx := c.Request.Context()
		entries := make(map[string]map[string]any, len(symbols))

		// 1) 优先从 Redis 批量读取 quote hash（pipeline，极快）
		if d.Redis != nil {
			redisQuotes, err := d.Redis.GetQuotes(ctx, symbols)
			if err != nil {
				log.Warnf("redis GetQuotes failed, fallback to PG: %v", err)
			} else {
				for sym, fields := range redisQuotes {
					entries[sym] = redisQuoteToEntry(sym, fields)
				}
			}
		}

		// 2) Redis 缺失的走 PG singleflight 回源
		var missed []string
		for _, sym := range symbols {
			if _, ok := entries[sym]; !ok {
				missed = append(missed, sym)
			}
		}
		for _, sym := range missed {
			if entry, err := loadQuoteEntry(ctx, d, sym); err == nil {
				entries[sym] = entry
			} else if !errors.Is(err, storage.ErrNotFound) {
				log.Errorf("loadQuoteEntry %s error: %v", sym, err)
			}
		}

		// 3) 仍缺失的走日 K 兜底（保留旧行为）
		var stillMissed []string
		for _, sym := range symbols {
			if _, ok := entries[sym]; !ok {
				stillMissed = append(stillMissed, sym)
			}
		}
		if len(stillMissed) > 0 && d.MySQL != nil {
			fallback, fbErr := d.MySQL.QueryLatestDailyKlines(ctx, stillMissed)
			if fbErr != nil {
				log.Errorf("postgres fallback for quotes failed: %v", fbErr)
			} else {
				for _, sym := range stillMissed {
					if row, ok := fallback[sym]; ok {
						entries[sym] = dailyKlineToQuoteMap(sym, row)
					}
				}
			}
		}

		items := make([]map[string]any, 0, len(symbols))
		for _, sym := range symbols {
			if entry, ok := entries[sym]; ok {
				items = append(items, entry)
			}
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"items": items,
			"total": len(items),
		}))
	}
}

// loadQuoteEntry 通过 singleflight 串行查询 PG。
// 返回的 entry 已是前端可直接使用的字段（价格已 ÷10000 转 float）。
func loadQuoteEntry(ctx context.Context, d Deps, symbol string) (map[string]any, error) {
	v, err, _ := quoteSF.Do(symbol, func() (any, error) {
		if d.MySQL == nil {
			return nil, storage.ErrNotFound
		}
		rows, qErr := d.MySQL.QueryQuoteSnapshots(ctx, []string{symbol})
		if qErr != nil {
			return nil, qErr
		}
		row, ok := rows[symbol]
		if !ok {
			return nil, storage.ErrNotFound
		}
		return quoteSnapshotToEntry(row), nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// redisQuoteToEntry 把 Redis hash 字段转为前端可用的 entry。
func redisQuoteToEntry(symbol string, fields map[string]string) map[string]any {
	parseF := func(key string) float64 {
		if v, ok := fields[key]; ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f / 10000
			}
		}
		return 0
	}
	parseI := func(key string) int64 {
		if v, ok := fields[key]; ok {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				return n
			}
		}
		return 0
	}
	return map[string]any{
		"symbol":     symbol,
		"last_price": parseF("last_price"),
		"prev_close": parseF("prev_close"),
		"open_price": parseF("open_price"),
		"high_price": parseF("high_price"),
		"low_price":  parseF("low_price"),
		"volume":     parseI("volume"),
		"turnover":   parseF("turnover"),
		"event_time": fields["event_time"],
		"_source":    "redis",
	}
}

// quoteSnapshotToEntry 把 PG 行直接转为前端可用的 entry（价格已 ÷10000）。
func quoteSnapshotToEntry(row storage.StockQuoteSnapshotRow) map[string]any {
	last, _ := row.LastPrice.Float64()
	prev, _ := row.PrevClose.Float64()
	open, _ := row.OpenPrice.Float64()
	high, _ := row.HighPrice.Float64()
	low, _ := row.LowPrice.Float64()
	turnover, _ := row.Turnover.Float64()
	return map[string]any{
		"symbol":     row.Symbol,
		"last_price": last / 10000,
		"prev_close": prev / 10000,
		"open_price": open / 10000,
		"high_price": high / 10000,
		"low_price":  low / 10000,
		"volume":     row.Volume,
		"turnover":   turnover / 10000,
		"event_time": row.EventTime,
		"_source":    "pg_quote_snapshot",
	}
}

func handleFenbi(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		limit, err := parseLimitQuery(c, "limit", fenbiLimitDefault, fenbiLimitMax)
		if err != nil {
			writeError(c, err)
			return
		}

		rows, err := d.Redis.GetFenbi(c.Request.Context(), symbol, int64(limit))
		if err != nil {
			log.Errorf("get fenbi failed symbol=%s limit=%d err=%v", symbol, limit, err)
			writeError(c, err)
			return
		}

		fenbiPriceFields := []string{"price", "amount", "bid1", "ask1"}
		items := make([]map[string]any, 0, len(rows))
		for _, raw := range rows {
			var obj map[string]any
			if err := json.Unmarshal([]byte(raw), &obj); err != nil {
				log.Warnf("fenbi row decode failed, skip symbol=%s err=%v", symbol, err)
				continue
			}
			for _, f := range fenbiPriceFields {
				if v, ok := obj[f]; ok {
					if n, ok := v.(float64); ok {
						obj[f] = n / 10000
					}
				}
			}
			items = append(items, obj)
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"items": items,
			"total": len(items),
		}))
	}
}

func parseLimitQuery(c *gin.Context, key string, def, max int) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, NewApiError(model.CodeInvalidParam, key+" must be a positive integer")
	}
	if n > max {
		log.Warnf("query param exceeds max, clamped key=%s raw=%d max=%d", key, n, max)
		return max, nil
	}
	return n, nil
}

func dailyKlineToQuoteMap(symbol string, row storage.StockDailyKlineRow) map[string]any {
	open, _ := row.Open.Float64()
	high, _ := row.High.Float64()
	low, _ := row.Low.Float64()
	cls, _ := row.Close.Float64()
	turnover, _ := row.Turnover.Float64()
	return map[string]any{
		"symbol":     symbol,
		"last_price": cls / 10000,
		"prev_close": open / 10000,
		"open_price": open / 10000,
		"high_price": high / 10000,
		"low_price":  low / 10000,
		"volume":     row.Volume,
		"turnover":   turnover / 10000,
		"trade_date": time.Unix(row.TradeDate, 0).In(shanghaiLoc).Format("2006-01-02"),
		"_source":    "pg_daily_kline",
	}
}
