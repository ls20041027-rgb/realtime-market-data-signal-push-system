package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/config"
	kafkax "push_gateway/kafka"
	"push_gateway/model"
	"push_gateway/storage"
	"push_gateway/ws"
)

type Deps struct {
	Cfg      *config.Settings
	Redis    *storage.RedisStore
	MySQL    *storage.PostgresStore
	Hub      HubStatus
	Consumer ConsumerStatus
	StartAt  time.Time
}

type HubStatus interface {
	Stats() ws.HubStats
}

type ConsumerStatus interface {
	Stats() []kafkax.TopicStats
}

type ApiError struct {
	Code int
	Msg  string
}

func (e *ApiError) Error() string { return e.Msg }

func NewApiError(code int, msg string) *ApiError { return &ApiError{Code: code, Msg: msg} }

func NewRouter(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(recoveryMiddleware())
	r.Use(corsMiddleware())
	r.Use(requestIDMiddleware())
	r.Use(slogLoggerMiddleware())

	r.GET("/healthz", healthHandler)

	api := r.Group("/api")

	api.GET("/quote/:symbol", handleQuote(d))
	api.GET("/quotes", handleQuotes(d))
	api.GET("/fenbi/:symbol", handleFenbi(d))

	api.GET("/indicators/:symbol", handleIndicators(d))
	api.GET("/capital/:symbol", handleCapital(d))

	api.GET("/kline/:symbol", handleKlineDaily(d))
	api.GET("/kline5m/:symbol", handleKline5Min(d))
	api.GET("/signals", handleSignals(d))
	api.GET("/signals/:id", handleSignalByID(d))

	api.GET("/stock-list", handleStockList(d))
	api.GET("/stock/:symbol", handleStockInfo(d))
	api.GET("/finance/:symbol", handleFinance(d))

	api.GET("/status", handleStatus(d))
	api.GET("/storage-stats", handleStorageStats(d))
	api.GET("/ingest-stats", handleIngestStats(d))

	return r
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, model.Ok(gin.H{
		"status":  "ok",
		"service": "push_gateway",
	}))
}


func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("api handler panic recovered",
					"component", "api",
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"panic", r,
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError,
						model.Err(model.CodeInternalError, "internal server error"))
				}
			}
		}()
		c.Next()
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = genRequestID()
		}
		c.Set("request_id", rid)
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}

func slogLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		rid, _ := c.Get("request_id")
		attrs := []any{
			"component", "api",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", rid,
		}
		if status >= 500 {
			slog.Warn("api request", attrs...)
		} else {
			slog.Info("api request", attrs...)
		}
	}
}

func writeError(c *gin.Context, err error) {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		c.JSON(httpStatusOfCode(apiErr.Code), model.Err(apiErr.Code, apiErr.Msg))
		return
	}
	if errors.Is(err, storage.ErrNotFound) {
		c.JSON(http.StatusNotFound, model.Err(model.CodeResourceNotFound, "resource not found"))
		return
	}
	slog.Error("unhandled api error", "component", "api", "err", err)
	c.JSON(http.StatusInternalServerError,
		model.Err(model.CodeInternalError, "internal server error"))
}

func httpStatusOfCode(code int) int {
	switch code {
	case model.CodeOK:
		return http.StatusOK
	case model.CodeResourceNotFound:
		return http.StatusNotFound
	case model.CodeInvalidParam, model.CodeInvalidChannel:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func genRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(b[:])
}
