package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	log "push_gateway/internal/log"
	"push_gateway/model"
	"push_gateway/storage"
)

const capitalHistoryDays = 30

func handleIndicators(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}

		var (
			wg                    sync.WaitGroup
			indicator, tech       map[string]string
			errIndicator, errTech error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			indicator, errIndicator = d.Redis.GetIndicator(c.Request.Context(), symbol)
		}()
		go func() {
			defer wg.Done()
			tech, errTech = d.Redis.GetTech(c.Request.Context(), symbol)
		}()
		wg.Wait()

		indicatorMissing := errors.Is(errIndicator, storage.ErrNotFound)
		techMissing := errors.Is(errTech, storage.ErrNotFound)
		if indicatorMissing && techMissing {
			writeError(c, storage.ErrNotFound)
			return
		}
		if errIndicator != nil && !indicatorMissing {
			writeError(c, errIndicator)
			return
		}
		if errTech != nil && !techMissing {
			writeError(c, errTech)
			return
		}

		merged := map[string]any{"symbol": symbol}
		mergeScaledFields(merged, indicator, indicatorScaledFields)
		mergeScaledFields(merged, tech, techScaledFields)
		c.JSON(http.StatusOK, model.Ok(merged))
	}
}

// indicatorScaledFields 列出 indicator: hash 中需要 ÷10000 还原的字段。
// 约定见 stream_engine/pipeline/build.py：
//   - change_amt: 价格差，×10000
//   - change_pct: 百分比，×1_000_000（×10000 价格 × ×100 百分比），还原为 1.25 表示 1.25%
//   - turnover_rate: 比率，×10000，还原为 0.0125 表示 1.25%
var indicatorScaledFields = map[string]float64{
	"change_amt":    10000,
	"change_pct":    10000,
	"turnover_rate": 10000,
}

// techScaledFields 列出 tech: hash 中需要 ÷10000 还原的字段。
// 约定见 stream_engine/pipeline/tech_indicator.py: PRICE_SCALE = 10000
var techScaledFields = map[string]float64{
	"ma5":        10000,
	"ma10":       10000,
	"ma20":       10000,
	"ma60":       10000,
	"rsi14":      10000,
	"boll_mid":   10000,
	"boll_upper": 10000,
	"boll_lower": 10000,
}

// mergeScaledFields 把 redis hash（值都是 string）合并进 dst：
// 命中 scaled 表的字段尝试 ParseFloat 后 ÷scale 转 float；其他字段保持字符串原样。
func mergeScaledFields(dst map[string]any, src map[string]string, scaled map[string]float64) {
	for k, v := range src {
		if scale, ok := scaled[k]; ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				dst[k] = f / scale
				continue
			}
		}
		dst[k] = v
	}
}

func handleCapital(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}

		snapshot, err := d.Redis.GetCapital(c.Request.Context(), symbol)
		if err != nil {
			log.Errorf("GetCapital error: %v", err)
			writeError(c, err)
			return
		}

		capitalPriceFields := map[string]bool{
			"big_buy": true, "big_sell": true, "net_inflow": true,
			"big_buy_amt": true, "big_sell_amt": true, "med_buy_amt": true, "med_sell_amt": true,
			"small_buy_amt": true, "small_sell_amt": true,
		}
		snapshotOut := make(map[string]any, len(snapshot))
		for k, v := range snapshot {
			if capitalPriceFields[k] {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					snapshotOut[k] = f / 10000
					continue
				}
			}
			snapshotOut[k] = v
		}

		result := gin.H{
			"symbol":   symbol,
			"snapshot": snapshotOut,
		}

		if c.Query("history") == "1" {
			end := time.Now()
			start := end.AddDate(0, 0, -capitalHistoryDays)
			rows, err := d.MySQL.QueryDailyCapitalFlow(c.Request.Context(), symbol, start, end)
			if err != nil {
				writeError(c, err)
				return
			}
			history := make([]gin.H, 0, len(rows))
			for _, r := range rows {
				bigBuy, _ := r.BigBuy.Float64()
				bigSell, _ := r.BigSell.Float64()
				netInflow, _ := r.NetInflow.Float64()
				history = append(history, gin.H{
					"symbol":     r.Symbol,
					"trade_date": r.TradeDate,
					"big_buy":    bigBuy / 10000,
					"big_sell":   bigSell / 10000,
					"net_inflow": netInflow / 10000,
				})
			}
			result["history"] = history
			result["history_days"] = capitalHistoryDays
		}

		c.JSON(http.StatusOK, model.Ok(result))
	}
}
