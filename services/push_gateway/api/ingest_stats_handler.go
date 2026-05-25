package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

const (
	ingestStatsTimeout  = 500 * time.Millisecond
	ingestStatsRedisKey = "stream:ingest:counters"
)

var messageTypeLabels = map[string]string{
	"RCV_REPORT":      "实时行情(REPORT)",
	"RCV_FENBIDATA":   "分笔成交(FENBI)",
	"RCV_MKTTBLDATA":  "行情快照(MKTTBL)",
	"RCV_FINANCEDATA": "财务数据(FINANCE)",
}

func handleIngestStats(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), ingestStatsTimeout)
		defer cancel()

		fields := map[string]string{}
		var readErr error
		if d.Redis != nil {
			fields, readErr = d.Redis.HGetAllMap(ctx, ingestStatsRedisKey)
		}

		updatedAtMs := parseInt64(fields["__updated_at_ms"])
		startedAtMs := parseInt64(fields["__started_at_ms"])

		messageTypes := make([]gin.H, 0, 8)
		totalCount := int64(0)

		for field, valueStr := range fields {
			if strings.HasPrefix(field, "__") {
				continue
			}
			count := parseInt64(valueStr)
			if strings.HasPrefix(field, "mt:") {
				key := strings.TrimPrefix(field, "mt:")
				messageTypes = append(messageTypes, gin.H{
					"message_type": key,
					"label":        labelOr(messageTypeLabels, key),
					"count":        count,
				})
				totalCount += count
			}
		}

		sort.Slice(messageTypes, func(i, j int) bool {
			ci, _ := messageTypes[i]["count"].(int64)
			cj, _ := messageTypes[j]["count"].(int64)
			if ci != cj {
				return ci > cj
			}
			return messageTypes[i]["message_type"].(string) < messageTypes[j]["message_type"].(string)
		})

		out := gin.H{
			"message_types": messageTypes,
			"total_count":   totalCount,
			"updated_at_ms": updatedAtMs,
			"started_at_ms": startedAtMs,
			"now_ms":        time.Now().UnixMilli(),
		}
		if readErr != nil {
			out["error"] = readErr.Error()
		}
		c.JSON(http.StatusOK, model.Ok(out))
	}
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func labelOr(m map[string]string, key string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return key
}
