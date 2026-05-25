package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/internal/latency"
	"push_gateway/model"
)

const (
	latencyStatsTimeout = 500 * time.Millisecond
	latencySampleTake   = 50
)

// LatencyStageStat 单个阶段的累计 + 平均 + 分位数统计。
type LatencyStageStat struct {
	Stage string  `json:"stage"`
	Label string  `json:"label"`
	Count int64   `json:"count"`
	SumNs int64   `json:"sum_ns"`
	AvgNs float64 `json:"avg_ns"`
	AvgMs float64 `json:"avg_ms"`
	P50Ns int64   `json:"p50_ns"`
	P90Ns int64   `json:"p90_ns"`
	P99Ns int64   `json:"p99_ns"`
	P50Ms float64 `json:"p50_ms"`
	P90Ms float64 `json:"p90_ms"`
	P99Ms float64 `json:"p99_ms"`
}

var latencyStages = []struct {
	Key   string
	Label string
}{
	{"ingest_proc_ns", "接入层处理"},
	{"kafka1_ns", "Kafka1 (接入→引擎)"},
	{"engine_ns", "Engine 流计算处理"},
	{"kafka2_ns", "Kafka2 (引擎→网关)"},
	{"gateway_ns", "网关推送"},
	{"end_to_end_ns", "端到端"},
}

func handleLatencyStats(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), latencyStatsTimeout)
		defer cancel()

		out := gin.H{
			"latest":  gin.H{},
			"stages":  []LatencyStageStat{},
			"samples": []map[string]interface{}{},
			"now_ms":  time.Now().UnixMilli(),
		}

		if d.Redis == nil {
			c.JSON(http.StatusOK, model.Ok(out))
			return
		}

		// latest
		if latest, err := d.Redis.HGetAllMap(ctx, latency.RedisLatencyLatestKey); err == nil {
			out["latest"] = mapToInterface(latest)
		}

		// stages: count + sum_ns -> avg
		stages := make([]LatencyStageStat, 0, len(latencyStages))
		if statsMap, err := d.Redis.HGetAllMap(ctx, latency.RedisLatencyStatsKey); err == nil {
			for _, s := range latencyStages {
				count := parseInt64(statsMap[s.Key+":count"])
				sumNs := parseInt64(statsMap[s.Key+":sum_ns"])
				avgNs := 0.0
				if count > 0 {
					avgNs = float64(sumNs) / float64(count)
				}
				p50 := parseInt64(statsMap[s.Key+":p50_ns"])
				p90 := parseInt64(statsMap[s.Key+":p90_ns"])
				p99 := parseInt64(statsMap[s.Key+":p99_ns"])
				stages = append(stages, LatencyStageStat{
					Stage: s.Key,
					Label: s.Label,
					Count: count,
					SumNs: sumNs,
					AvgNs: avgNs,
					AvgMs: avgNs / 1e6,
					P50Ns: p50,
					P90Ns: p90,
					P99Ns: p99,
					P50Ms: float64(p50) / 1e6,
					P90Ms: float64(p90) / 1e6,
					P99Ms: float64(p99) / 1e6,
				})
			}
		}
		out["stages"] = stages

		// samples: 最近 N 条
		client := d.Redis.Client()
		if client != nil {
			items, err := client.LRange(ctx, latency.RedisLatencySamplesKey, -latencySampleTake, -1).Result()
			if err == nil {
				samples := make([]map[string]interface{}, 0, len(items))
				for _, raw := range items {
					var m map[string]interface{}
					if jerr := json.Unmarshal([]byte(raw), &m); jerr == nil {
						samples = append(samples, m)
					}
				}
				out["samples"] = samples
			}
		}

		c.JSON(http.StatusOK, model.Ok(out))
	}
}

func mapToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			out[k] = n
		} else {
			out[k] = v
		}
	}
	return out
}
