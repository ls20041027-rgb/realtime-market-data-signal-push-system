package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

const (
	financeLimitDefault = 8
	financeLimitMax     = 40
)

func handleStockList(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.MySQL.QueryStockList(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		items := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			items = append(items, gin.H{"symbol": r.Symbol, "name": r.Name})
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"items": items,
			"total": len(items),
		}))
	}
}

func handleStockInfo(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		row, err := d.MySQL.QueryStockInfo(c.Request.Context(), symbol)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(row))
	}
}

func handleFinance(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := strings.TrimSpace(c.Param("symbol"))
		if symbol == "" {
			writeError(c, NewApiError(model.CodeInvalidParam, "symbol is required"))
			return
		}
		limit, err := parseLimitQuery(c, "limit", financeLimitDefault, financeLimitMax)
		if err != nil {
			writeError(c, err)
			return
		}
		rows, err := d.MySQL.QueryFinance(c.Request.Context(), symbol, limit)
		if err != nil {
			writeError(c, err)
			return
		}
		items := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			eps, _ := r.EPS.Float64()
			bps, _ := r.BPS.Float64()
			netProfit, _ := r.NetProfit.Float64()
			totalShares, _ := r.TotalShares.Float64()
			floatShares, _ := r.FloatShares.Float64()
			items = append(items, gin.H{
				"symbol":       r.Symbol,
				"report_date":  r.ReportDate,
				"total_shares": totalShares / 10000,
				"float_shares": floatShares / 10000,
				"eps":          eps / 10000,
				"bps":          bps / 10000,
				"net_profit":   netProfit / 10000,
			})
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"symbol": symbol,
			"items":  items,
			"total":  len(items),
		}))
	}
}
