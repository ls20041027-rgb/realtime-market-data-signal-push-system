package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
	"push_gateway/storage"
)

const (
	signalPageSizeDefault = 20
	signalPageSizeMax     = 200
)

func handleSignals(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := storage.SignalFilter{
			Symbol:     strings.TrimSpace(c.Query("symbol")),
			SignalType: strings.TrimSpace(c.Query("signal_type")),
			Severity:   strings.TrimSpace(c.Query("severity")),
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

		c.JSON(http.StatusOK, model.Ok(model.PagedData{
			Items:    rows,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}))
	}
}

func handleSignalByID(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("id"))
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(c, NewApiError(model.CodeInvalidParam, "id must be a positive integer"))
			return
		}
		row, err := d.MySQL.QuerySignalByID(c.Request.Context(), id)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.Ok(row))
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
