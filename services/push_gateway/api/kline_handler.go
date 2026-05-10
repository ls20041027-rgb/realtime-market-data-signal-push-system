package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

const (
	klineLimitMax     = 1000
	klineLimitDefault = 300

	dateLayout     = "2006-01-02"
	datetimeLayout = "2006-01-02T15:04:05"
)

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
		rows, err := d.MySQL.QueryDailyKline(c.Request.Context(), symbol, start, end, limit)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"symbol": symbol,
			"items":  rows,
			"total":  len(rows),
		}))
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
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"symbol": symbol,
			"items":  rows,
			"total":  len(rows),
		}))
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
