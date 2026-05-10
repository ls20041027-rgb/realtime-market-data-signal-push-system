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
	"RCV_FILEDATA":    "文件数据(FILEDATA)",
}

var fileDataTypeLabels = map[string]string{
	"FILE_HISTORY_EX": "日K线",
	"FILE_MINUTE_EX":  "分时线",
	"FILE_5MINUTE_EX": "5分钟K线",
	"FILE_POWER_EX":   "除权除息",
	"FILE_BASE_EX":    "基础信息(忽略)",
	"FILE_NEWS_EX":    "新闻(忽略)",
	"FILE_HTML_EX":    "HTML(忽略)",
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
		fileDataTypes := make([]gin.H, 0, 8)
		totalCount := int64(0)

		for field, valueStr := range fields {
			if strings.HasPrefix(field, "__") {
				continue
			}
			count := parseInt64(valueStr)
			switch {
			case strings.HasPrefix(field, "mt:"):
				key := strings.TrimPrefix(field, "mt:")
				messageTypes = append(messageTypes, gin.H{
					"message_type": key,
					"label":        labelOr(messageTypeLabels, key),
					"count":        count,
				})
				totalCount += count
			case strings.HasPrefix(field, "ft:"):
				key := strings.TrimPrefix(field, "ft:")
				fileDataTypes = append(fileDataTypes, gin.H{
					"file_data_type": key,
					"label":          labelOr(fileDataTypeLabels, key),
					"count":          count,
				})
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
		sort.Slice(fileDataTypes, func(i, j int) bool {
			ci, _ := fileDataTypes[i]["count"].(int64)
			cj, _ := fileDataTypes[j]["count"].(int64)
			if ci != cj {
				return ci > cj
			}
			return fileDataTypes[i]["file_data_type"].(string) < fileDataTypes[j]["file_data_type"].(string)
		})

		out := gin.H{
			"message_types":   messageTypes,
			"file_data_types": fileDataTypes,
			"total_count":     totalCount,
			"updated_at_ms":   updatedAtMs,
			"started_at_ms":   startedAtMs,
			"now_ms":          time.Now().UnixMilli(),
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
