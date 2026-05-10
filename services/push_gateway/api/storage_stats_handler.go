package api


import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"push_gateway/model"
)

const (
	storageStatsTimeout = 3 * time.Second
	storageStatsScanCap = 50000
)

var mysqlStatTables = []struct {
	Label string
	Table string
}{
	{"证券基础信息", "stock_info"},
	{"日K线", "stock_daily_kline"},
	{"5分钟K线", "stock_5min_kline"},
	{"分时线", "stock_minute_kline"},
	{"除权除息", "stock_power"},
	{"财务数据", "stock_finance"},
	{"日度资金流", "daily_capital_flow"},
	{"信号历史", "signal_history"},
}

func handleStorageStats(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), storageStatsTimeout)
		defer cancel()

		mysqlItems := collectMySQLStats(ctx, d)
		redisItems := collectRedisStats(ctx, d)

		c.JSON(http.StatusOK, model.Ok(gin.H{
			"postgres":   mysqlItems,
			"redis":      redisItems,
			"scan_limit": storageStatsScanCap,
			"ts":         time.Now().UnixMilli(),
		}))
	}
}

func collectMySQLStats(ctx context.Context, d Deps) []gin.H {
	out := make([]gin.H, len(mysqlStatTables))
	if d.MySQL == nil {
		for i, t := range mysqlStatTables {
			out[i] = gin.H{"label": t.Label, "table": t.Table, "count": 0, "error": "postgres not initialized"}
		}
		return out
	}
	var wg sync.WaitGroup
	for i, t := range mysqlStatTables {
		wg.Add(1)
		go func(i int, label, table string) {
			defer wg.Done()
			n, err := d.MySQL.CountTable(ctx, table)
			item := gin.H{"label": label, "table": table, "count": n}
			if err != nil {
				item["error"] = err.Error()
			}
			out[i] = item
		}(i, t.Label, t.Table)
	}
	wg.Wait()
	return out
}

func collectRedisStats(ctx context.Context, d Deps) []gin.H {
	type entry struct {
		label      string
		prefix     string
		singleHash string
	}
	rc := d.Cfg.Redis
	entries := []entry{
		{"实时行情快照", rc.QuotePrefix, ""},
		{"技术指标快照", rc.IndicatorPrefix, ""},
		{"技术分析", rc.TechPrefix, ""},
		{"资金流快照", rc.CapitalPrefix, ""},
		{"财务快照", rc.FinancePrefix, ""},
		{"分笔列表", rc.FenbiPrefix, ""},
		{"日K列表", rc.HistDailyPrefix, ""},
		{"证券名单(Hash字段)", "", rc.StockListKey},
	}
	out := make([]gin.H, len(entries))
	if d.Redis == nil {
		for i, e := range entries {
			out[i] = gin.H{"label": e.label, "prefix": e.prefix, "count": 0, "error": "redis not initialized"}
		}
		return out
	}
	var wg sync.WaitGroup
	for i, e := range entries {
		wg.Add(1)
		go func(i int, e entry) {
			defer wg.Done()
			item := gin.H{"label": e.label, "prefix": e.prefix}
			if e.singleHash != "" {
				item["key"] = e.singleHash
				n, err := d.Redis.HLen(ctx, e.singleHash)
				item["count"] = n
				if err != nil {
					item["error"] = err.Error()
				}
			} else if e.prefix == "" {
				item["count"] = 0
				item["error"] = "prefix not configured"
			} else {
				n, truncated, err := d.Redis.CountByPrefix(ctx, e.prefix, storageStatsScanCap)
				item["count"] = n
				item["truncated"] = truncated
				if err != nil {
					item["error"] = err.Error()
				}
			}
			out[i] = item
		}(i, e)
	}
	wg.Wait()
	return out
}
