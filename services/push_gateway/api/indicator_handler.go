package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

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
		for k, v := range indicator {
			merged[k] = v
		}
		for k, v := range tech {
			merged[k] = v
		}
		c.JSON(http.StatusOK, model.Ok(merged))
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
			writeError(c, err)
			return
		}

		result := gin.H{
			"symbol":   symbol,
			"snapshot": snapshot,
		}

		if c.Query("history") == "1" {
			end := time.Now()
			start := end.AddDate(0, 0, -capitalHistoryDays)
			rows, err := d.MySQL.QueryDailyCapitalFlow(c.Request.Context(), symbol, start, end)
			if err != nil {
				writeError(c, err)
				return
			}
			result["history"] = rows
			result["history_days"] = capitalHistoryDays
		}

		c.JSON(http.StatusOK, model.Ok(result))
	}
}
