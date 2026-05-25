package api

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	log "push_gateway/internal/log"
	resourcepkg "push_gateway/internal/resource"
	"push_gateway/model"
)

const statusPingTimeout = 200 * time.Millisecond

var resourceServices = []string{"data_ingestion", "stream_engine", "push_gateway"}

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

	out := gin.H{
		"ws": wsPart,
		"runtime": gin.H{
			"pid":            os.Getpid(),
			"goroutines":     runtime.NumGoroutine(),
			"uptime_seconds": uptimeSeconds(d.StartAt),
		},
		"resources": collectResources(ctx, d),
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

// collectResources reads CPU/RSS samples for all services from Redis.
// Each service writes its own hash `stream:resource:<service>` periodically;
// here we just aggregate them so the frontend has a single endpoint.
func collectResources(parent context.Context, d Deps) []gin.H {
	out := make([]gin.H, 0, len(resourceServices))
	if d.Redis == nil {
		for _, name := range resourceServices {
			out = append(out, gin.H{"service": name, "up": false})
		}
		return out
	}
	ctx, cancel := context.WithTimeout(parent, statusPingTimeout)
	defer cancel()
	nowMs := time.Now().UnixMilli()
	for _, name := range resourceServices {
		item := gin.H{"service": name, "up": false}
		key := "stream:resource:" + name
		m, err := d.Redis.HGetAllMap(ctx, key)
		if err != nil || len(m) == 0 {
			out = append(out, item)
			continue
		}
		updatedMs := parseInt64Default(m["updated_at_ms"], 0)
		item["pid"] = parseInt64Default(m["pid"], 0)
		item["cpu_percent"] = parseFloatDefault(m["cpu_percent"], 0)
		item["rss_bytes"] = parseInt64Default(m["rss_bytes"], 0)
		item["num_threads"] = parseInt64Default(m["num_threads"], 0)
		item["updated_at_ms"] = updatedMs
		item["stale_ms"] = nowMs - updatedMs
		// Treat samples within 10s as live.
		item["up"] = updatedMs > 0 && (nowMs-updatedMs) < 10_000
		out = append(out, item)
	}
	return out
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func parseFloatDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

// localResourceMonitor is held on package-level so main can Stop() it cleanly.
var localResourceMonitor *resourcepkg.Monitor

func StartResourceMonitor(d Deps) {
	if d.Redis == nil {
		log.Warnf("resource monitor disabled: redis is nil")
		return
	}
	m, err := resourcepkg.NewMonitor(d.Redis.Client())
	if err != nil {
		log.Errorf("resource monitor init failed: %v", err)
		return
	}
	localResourceMonitor = m
	m.Start()
}

func StopResourceMonitor() {
	if localResourceMonitor != nil {
		localResourceMonitor.Stop()
		localResourceMonitor = nil
	}
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
		log.Debugf("status ping failed: %v", err)
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
		log.Infof("metrics loop started interval=%s", interval.String())
		for {
			select {
			case <-ctx.Done():
				log.Infof("metrics loop stopped")
				return
			case <-t.C:
				snap := buildStatusSnapshot(ctx, d, false)
				log.Infof("runtime metrics ws=%v runtime=%v", snap["ws"], snap["runtime"])
			}
		}
	}()
}
