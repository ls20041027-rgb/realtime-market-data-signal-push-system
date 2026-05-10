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
		m, err := d.Redis.GetStockList(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		items := make([]gin.H, 0, len(m))
		for symbol, name := range m {
			items = append(items, gin.H{"symbol": symbol, "name": name})
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
		c.JSON(http.StatusOK, model.Ok(gin.H{
			"symbol": symbol,
			"items":  rows,
			"total":  len(rows),
		}))
	}
}
