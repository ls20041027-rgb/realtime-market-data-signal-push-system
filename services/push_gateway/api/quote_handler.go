package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
	"push_gateway/storage"
)

const (
	quotesSymbolsMax  = 200
	fenbiLimitMax     = 500
	fenbiLimitDefault = 100
)

func handleQuote(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		data, err := d.Redis.GetQuote(c.Request.Context(), symbol)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			writeError(c, err)
			return
		}
		if errors.Is(err, storage.ErrNotFound) {
			fallback, fbErr := d.MySQL.QueryLatestDailyKlines(c.Request.Context(), []string{symbol})
			if fbErr != nil || len(fallback) == 0 {
				writeError(c, storage.ErrNotFound)
				return
			}
			row := fallback[symbol]
			c.JSON(http.StatusOK, model.Ok(dailyKlineToQuoteMap(symbol, row)))
			return
		}
		c.JSON(http.StatusOK, model.Ok(parseJsonFields(data)))
	}
}

func parseJsonFields(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	jsonFields := map[string]bool{"bid_levels": true, "ask_levels": true}
	for k, v := range m {
		if jsonFields[k] && len(v) > 0 && v[0] == '[' {
			var arr []any
			if err := json.Unmarshal([]byte(v), &arr); err == nil {
				out[k] = arr
				continue
			}
		}
		out[k] = v
	}
	return out
}

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

		got, err := d.Redis.GetQuotes(c.Request.Context(), symbols)
		if err != nil {
			writeError(c, err)
			return
		}

		var missed []string
		for _, sym := range symbols {
			if _, ok := got[sym]; !ok {
				missed = append(missed, sym)
			}
		}

		var fallback map[string]storage.StockDailyKlineRow
		if len(missed) > 0 && d.MySQL != nil {
			var fbErr error
			fallback, fbErr = d.MySQL.QueryLatestDailyKlines(c.Request.Context(), missed)
			if fbErr != nil {
				slog.Warn("postgres fallback for quotes failed",
					"component", "api", "err", fbErr)
			}
		}

		items := make([]map[string]any, 0, len(symbols))
		for _, sym := range symbols {
			if m, ok := got[sym]; ok {
				entry := parseJsonFields(m)
				entry["symbol"] = sym
				items = append(items, entry)
			} else if row, ok := fallback[sym]; ok {
				items = append(items, dailyKlineToQuoteMap(sym, row))
			}
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"items": items,
			"total": len(items),
		}))
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
			writeError(c, err)
			return
		}

		items := make([]map[string]any, 0, len(rows))
		for _, raw := range rows {
			var obj map[string]any
			if err := json.Unmarshal([]byte(raw), &obj); err != nil {
				slog.Warn("fenbi row decode failed, skip",
					"component", "api", "symbol", symbol, "err", err)
				continue
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
		slog.Warn("query param exceeds max, clamped",
			"component", "api", "key", key, "raw", n, "max", max)
		return max, nil
	}
	return n, nil
}

func dailyKlineToQuoteMap(symbol string, row storage.StockDailyKlineRow) map[string]any {
	return map[string]any{
		"symbol":     symbol,
		"last_price": row.Close.String(),
		"prev_close": row.Open.String(),
		"open_price": row.Open.String(),
		"high_price": row.High.String(),
		"low_price":  row.Low.String(),
		"volume":     strconv.FormatInt(row.Volume, 10),
		"turnover":   row.Turnover.String(),
		"trade_date": row.TradeDate,
		"_source":    "pg_daily_kline",
	}
}
