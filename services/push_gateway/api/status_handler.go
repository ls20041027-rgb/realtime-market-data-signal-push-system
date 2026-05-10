package api


import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

const statusPingTimeout = 200 * time.Millisecond

func handleStatus(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, model.Ok(buildStatusSnapshot(c.Request.Context(), d, true)))
	}
}

func buildStatusSnapshot(ctx context.Context, d Deps, withPing bool) gin.H {
	wsPart := gin.H{"clients": 0, "channels": 0, "dropped_slow": int64(0)}
	if d.Hub != nil {
		s := d.Hub.Stats()
		wsPart = gin.H{
			"clients":       s.Clients,
			"channels":      s.Channels,
			"quote_clients": s.QuoteClients,
			"dropped_slow":  s.DroppedSlow,
		}
	}

	kafkaPart := gin.H{"topics": []any{}}
	if d.Consumer != nil {
		kafkaPart = gin.H{"topics": d.Consumer.Stats()}
	}

	out := gin.H{
		"ws":    wsPart,
		"kafka": kafkaPart,
		"runtime": gin.H{
			"pid":            os.Getpid(),
			"goroutines":     runtime.NumGoroutine(),
			"uptime_seconds": uptimeSeconds(d.StartAt),
		},
	}
	if withPing {
		out["redis"] = pingStore(ctx, d.Redis != nil, func(cc context.Context) error {
			if d.Redis == nil {
				return nil
			}
			return d.Redis.Ping(cc)
		})
		out["postgres"] = pingStore(ctx, d.MySQL != nil, func(cc context.Context) error {
			if d.MySQL == nil {
				return nil
			}
			return d.MySQL.Ping(cc)
		})
	}
	return out
}

func uptimeSeconds(startAt time.Time) int64 {
	if startAt.IsZero() {
		return 0
	}
	return int64(time.Since(startAt).Seconds())
}

func pingStore(parent context.Context, enabled bool, pingFn func(context.Context) error) gin.H {
	if !enabled {
		return gin.H{"up": false, "latency_ms": 0}
	}
	ctx, cancel := context.WithTimeout(parent, statusPingTimeout)
	defer cancel()
	start := time.Now()
	err := pingFn(ctx)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Debug("status ping failed", "component", "api", "err", err)
		return gin.H{"up": false, "latency_ms": latency, "error": err.Error()}
	}
	return gin.H{"up": true, "latency_ms": latency}
}

func StartMetricsLoop(ctx context.Context, d Deps) {
	if d.Cfg == nil || !d.Cfg.Runtime.MetricsEnabled {
		return
	}
	interval := d.Cfg.Runtime.MetricsInterval
	if interval <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		slog.Info("metrics loop started", "component", "api", "interval", interval.String())
		for {
			select {
			case <-ctx.Done():
				slog.Info("metrics loop stopped", "component", "api")
				return
			case <-t.C:
				snap := buildStatusSnapshot(ctx, d, false)
				slog.Info("runtime metrics",
					"component", "api",
					"ws", snap["ws"],
					"kafka", snap["kafka"],
					"runtime", snap["runtime"],
				)
			}
		}
	}()
}
