package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

type WatchlistAddRequest struct {
	Symbol string `json:"symbol" binding:"required"`
}

// handleGetWatchlist returns the current user's watchlist.
func handleGetWatchlist(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireUserID(c)
		if !ok {
			return
		}
		symbols, err := d.MySQL.GetWatchlist(c.Request.Context(), userID)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(symbols))
	}
}

// handleAddToWatchlist adds a symbol to the user's watchlist.
func handleAddToWatchlist(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireUserID(c)
		if !ok {
			return
		}
		var req WatchlistAddRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, model.Err(model.CodeInvalidParam, "invalid params: "+err.Error()))
			return
		}
		if err := d.MySQL.AddToWatchlist(c.Request.Context(), userID, req.Symbol); err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{"symbol": req.Symbol}))
	}
}

// handleRemoveFromWatchlist removes a symbol from the user's watchlist.
func handleRemoveFromWatchlist(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := requireUserID(c)
		if !ok {
			return
		}
		symbol := c.Param("symbol")
		if symbol == "" {
			c.JSON(http.StatusBadRequest, model.Err(model.CodeInvalidParam, "symbol is required"))
			return
		}
		if err := d.MySQL.RemoveFromWatchlist(c.Request.Context(), userID, symbol); err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(gin.H{"symbol": symbol}))
	}
}

// requireUserID extracts user_id from JWT context; writes error response and returns false if missing.
func requireUserID(c *gin.Context) (int64, bool) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "unauthorized"))
		return 0, false
	}
	return uid.(int64), true
}
