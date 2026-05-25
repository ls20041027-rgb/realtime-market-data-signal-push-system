package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"push_gateway/model"
	"push_gateway/storage"
)

const (
	signalPageSizeDefault = 20
	signalPageSizeMax     = 200
)

// signalScale 与全网关其他价格/金额接口保持一致：DB 中存储的 ×10000 缩放整数，
// 对外暴露时除以 10000 还原为人类可读的小数。
var signalScale = decimal.NewFromInt(10000)

// signalDTO 是 /api/signals 对前端的输出结构：
//   - trigger_price 由 stock_signal.trigger_price 缩放还原（÷10000）
//   - signal_time 统一格式化为 "YYYY-MM-DD HH:mm:ss"（兼容 epoch 秒 / RFC3339 / 已是日期字符串）
type signalDTO struct {
	SignalType   string `json:"signal_type"`
	Symbol       string `json:"symbol"`
	Severity     string `json:"severity"`
	Action       string `json:"action"`
	StrategyName string `json:"strategy_name"`
	TriggerPrice string `json:"trigger_price"`
	Reason       string `json:"reason"`
	SignalTime   string `json:"signal_time"`
}

func handleSignals(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := storage.SignalFilter{
			Symbol:     strings.TrimSpace(c.Query("symbol")),
			SignalType: strings.TrimSpace(c.Query("signal_type")),
			Severity:   strings.TrimSpace(c.Query("severity")),
			Action:     strings.TrimSpace(c.Query("action")),
		}

		if raw := strings.TrimSpace(c.Query("from")); raw != "" {
			t, err := time.ParseInLocation(datetimeLayout, raw, time.Local)
			if err != nil {
				writeError(c, NewApiError(model.CodeInvalidParam, "from must be "+datetimeLayout))
				return
			}
			filter.From = &t
		}
		if raw := strings.TrimSpace(c.Query("to")); raw != "" {
			t, err := time.ParseInLocation(datetimeLayout, raw, time.Local)
			if err != nil {
				writeError(c, NewApiError(model.CodeInvalidParam, "to must be "+datetimeLayout))
				return
			}
			filter.To = &t
		}

		page, pageSize, err := parsePagination(c, signalPageSizeDefault, signalPageSizeMax)
		if err != nil {
			writeError(c, err)
			return
		}
		filter.Page = page
		filter.PageSize = pageSize

		rows, total, err := d.MySQL.QuerySignals(c.Request.Context(), filter)
		if err != nil {
			writeError(c, err)
			return
		}

		items := make([]signalDTO, 0, len(rows))
		for _, r := range rows {
			ts, _ := strconv.ParseInt(strings.TrimSpace(r.SignalTime), 10, 64)
			items = append(items, signalDTO{
				SignalType:   r.SignalType,
				Symbol:       r.Symbol,
				Severity:     r.Severity,
				Action:       r.Action,
				StrategyName: r.StrategyName,
				TriggerPrice: r.TriggerPrice.Div(signalScale).String(),
				Reason:       r.Reason,
				SignalTime:   time.Unix(ts, 0).In(time.Local).Format(datetimeLayout),
			})
		}

		c.JSON(http.StatusOK, model.Ok(model.PagedData{
			Items:    items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}))
	}
}

func parsePagination(c *gin.Context, sizeDefault, sizeMax int) (int, int, error) {
	page := 1
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return 0, 0, NewApiError(model.CodeInvalidParam, "page must be a positive integer")
		}
		page = n
	}
	size, err := parseLimitQuery(c, "page_size", sizeDefault, sizeMax)
	if err != nil {
		return 0, 0, err
	}
	return page, size, nil
}
