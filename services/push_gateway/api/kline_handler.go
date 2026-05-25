package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"

	log "push_gateway/internal/log"
	"push_gateway/model"
)

// In-memory caches for kline queries (short TTL to reduce PG pressure under high concurrency)
var (
	klineDailyCache = NewMemCache(5*time.Minute, 2000)  // daily kline rarely changes
	kline1MinCache  = NewMemCache(30*time.Second, 5000) // 1min kline updates every minute
	klineDailySF    singleflight.Group
	kline1MinSF     singleflight.Group
)

const (
	klineLimitMax     = 1000
	klineLimitDefault = 300

	dateLayout     = "2006-01-02"
	datetimeLayout = "2006-01-02T15:04:05"
)

// 上海时区，避免依赖容器系统 TZ
var shanghaiLoc, _ = time.LoadLocation("Asia/Shanghai")

func handleKlineDaily(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		start, end, err := parseDateRange(c, "start", "end", dateLayout)
		if err != nil {
			writeError(c, err)
			return
		}
		limit, err := parseLimitQuery(c, "limit", klineLimitDefault, klineLimitMax)
		if err != nil {
			writeError(c, err)
			return
		}
		cacheKey := fmt.Sprintf("%s:%v:%v:%d", symbol, start.Unix(), end.Unix(), limit)
		if cached, ok := klineDailyCache.Get(cacheKey); ok {
			c.Data(http.StatusOK, "application/json; charset=utf-8", cached.([]byte))
			return
		}

		// singleflight: only one goroutine queries PG per cacheKey
		result, sfErr, _ := klineDailySF.Do(cacheKey, func() (any, error) {
			// Double-check cache after acquiring singleflight slot
			if cached, ok := klineDailyCache.Get(cacheKey); ok {
				return cached, nil
			}
			sfCtx, sfCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer sfCancel()
			rows, err := d.MySQL.QueryDailyKline(sfCtx, symbol, start, end, limit)
			if err != nil {
				return nil, err
			}
			log.Debugf("daily kline rows fetched symbol=%s rows=%d", symbol, len(rows))
			items := make([]gin.H, 0, len(rows))
			for _, r := range rows {
				open, _ := r.Open.Float64()
				high, _ := r.High.Float64()
				low, _ := r.Low.Float64()
				cls, _ := r.Close.Float64()
				turnover, _ := r.Turnover.Float64()
				items = append(items, gin.H{
					"symbol":     r.Symbol,
					"trade_date": time.Unix(r.TradeDate, 0).In(shanghaiLoc).Format("2006-01-02"),
					"open":       open / 10000,
					"high":       high / 10000,
					"low":        low / 10000,
					"close":      cls / 10000,
					"volume":     r.Volume,
					"turnover":   turnover / 10000,
				})
			}
			resp := model.Ok(gin.H{
				"symbol": symbol,
				"items":  items,
				"total":  len(items),
			})
			jsonBytes, err := json.Marshal(resp)
			if err != nil {
				return nil, err
			}
			klineDailyCache.Set(cacheKey, jsonBytes)
			return jsonBytes, nil
		})
		if sfErr != nil {
			log.Errorf("query daily kline failed symbol=%s err=%v", symbol, sfErr)
			writeError(c, sfErr)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", result.([]byte))
	}
}

func handleKline5Min(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		start, end, err := parseFlexibleRange(c, "start", "end")
		if err != nil {
			writeError(c, err)
			return
		}
		limit, err := parseLimitQuery(c, "limit", klineLimitDefault, klineLimitMax)
		if err != nil {
			writeError(c, err)
			return
		}
		rows, err := d.MySQL.Query5MinKline(c.Request.Context(), symbol, start, end, limit)
		if err != nil {
			log.Errorf("query 5min kline failed symbol=%s start=%v end=%v limit=%d err=%v",
				symbol, start, end, limit, err)
			writeError(c, err)
			return
		}
		log.Debugf("5min kline rows fetched symbol=%s rows=%d", symbol, len(rows))
		items := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			open, _ := r.Open.Float64()
			high, _ := r.High.Float64()
			low, _ := r.Low.Float64()
			cls, _ := r.Close.Float64()
			turnover, _ := r.Turnover.Float64()
			items = append(items, gin.H{
				"symbol":     r.Symbol,
				"trade_time": time.Unix(r.TradeTime, 0).In(shanghaiLoc).Format("2006-01-02 15:04"),
				"open":       open / 10000,
				"high":       high / 10000,
				"low":        low / 10000,
				"close":      cls / 10000,
				"volume":     r.Volume,
				"turnover":   turnover / 10000,
			})
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"symbol": symbol,
			"items":  items,
			"total":  len(items),
		}))
	}
}

func handleKline1Min(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		start, end, err := parseFlexibleRange(c, "start", "end")
		if err != nil {
			writeError(c, err)
			return
		}
		limit, err := parseLimitQuery(c, "limit", klineLimitDefault, klineLimitMax)
		if err != nil {
			writeError(c, err)
			return
		}
		// When no start/end specified, default to today 09:30 to get full day data.
		if start.IsZero() && end.IsZero() {
			now := time.Now().In(shanghaiLoc)
			start = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, shanghaiLoc)
			limit = 241
		}

		cacheKey := fmt.Sprintf("1m:%s:%v:%v:%d", symbol, start.Unix(), end.Unix(), limit)
		if cached, ok := kline1MinCache.Get(cacheKey); ok {
			c.Data(http.StatusOK, "application/json; charset=utf-8", cached.([]byte))
			return
		}

		// singleflight: only one goroutine queries PG per cacheKey
		result, sfErr, _ := kline1MinSF.Do(cacheKey, func() (any, error) {
			if cached, ok := kline1MinCache.Get(cacheKey); ok {
				return cached, nil
			}
			sfCtx, sfCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer sfCancel()
			rows, err := d.MySQL.Query1MinKlineRt(sfCtx, symbol, start, end, limit)
			if err != nil {
				return nil, err
			}
			log.Debugf("1min kline rt rows fetched symbol=%s rows=%d", symbol, len(rows))
			items := make([]gin.H, 0, len(rows))
			for _, r := range rows {
				items = append(items, gin.H{
					"symbol":     r.Symbol,
					"trade_time": time.Unix(r.TradeTime, 0).In(shanghaiLoc).Format("2006-01-02 15:04"),
					"close":      float64(r.Close) / 10000,
					"volume":     r.Volume,
					"turnover":   float64(r.Turnover) / 10000,
				})
			}

			var prevClose float64
			if d.MySQL != nil {
				if snaps, qerr := d.MySQL.QueryQuoteSnapshots(sfCtx, []string{symbol}); qerr == nil {
					if snap, ok := snaps[symbol]; ok {
						pc, _ := snap.PrevClose.Float64()
						prevClose = pc / 10000
					}
				} else {
					log.Warnf("query quote snapshot for prev_close failed symbol=%s err=%v", symbol, qerr)
				}
			}

			resp := model.Ok(gin.H{
				"symbol":     symbol,
				"items":      items,
				"total":      len(items),
				"prev_close": prevClose,
			})
			jsonBytes, err := json.Marshal(resp)
			if err != nil {
				return nil, err
			}
			kline1MinCache.Set(cacheKey, jsonBytes)
			return jsonBytes, nil
		})
		if sfErr != nil {
			log.Errorf("query 1min kline failed symbol=%s err=%v", symbol, sfErr)
			writeError(c, sfErr)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", result.([]byte))
	}
}

func parseDateRange(c *gin.Context, startKey, endKey, layout string) (time.Time, time.Time, error) {
	return parseTimeRange(c, startKey, endKey, false)
}

func parseFlexibleRange(c *gin.Context, startKey, endKey string) (time.Time, time.Time, error) {
	return parseTimeRange(c, startKey, endKey, true)
}

func parseTimeRange(c *gin.Context, startKey, endKey string, flexible bool) (time.Time, time.Time, error) {
	parseOne := func(key, raw string, expandDay bool) (time.Time, error) {
		if raw == "" {
			return time.Time{}, nil
		}
		if flexible {
			if t, err := time.ParseInLocation(datetimeLayout, raw, time.Local); err == nil {
				return t, nil
			}
			if t, err := time.ParseInLocation(dateLayout, raw, time.Local); err == nil {
				if expandDay {
					t = t.Add(24*time.Hour - time.Second)
				}
				return t, nil
			}
			return time.Time{}, NewApiError(model.CodeInvalidParam,
				key+" must be "+dateLayout+" or "+datetimeLayout)
		}
		t, err := time.ParseInLocation(dateLayout, raw, time.Local)
		if err != nil {
			return time.Time{}, NewApiError(model.CodeInvalidParam, key+" must be "+dateLayout)
		}
		return t, nil
	}

	start, err := parseOne(startKey, strings.TrimSpace(c.Query(startKey)), false)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseOne(endKey, strings.TrimSpace(c.Query(endKey)), true)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return start, end, NewApiError(model.CodeInvalidParam, startKey+" must be <= "+endKey)
	}
	return start, end, nil
}
