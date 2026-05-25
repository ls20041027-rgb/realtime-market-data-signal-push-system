package latency

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	log "push_gateway/internal/log"
)

const (
	RedisLatencyLatestKey  = "stream:latency:latest"
	RedisLatencySamplesKey = "stream:latency:samples"
	RedisLatencyStatsKey   = "stream:latency:stats"
	SamplesMaxLen          = 200
	queueCapacity          = 4096
	// percentileWindow is the rolling window size for percentile (P50/P90/P99) calculation.
	percentileWindow = 2048
)

// stageNames lists the duration field keys we maintain percentile windows for.
// Order matches the field extraction in pickStage().
var stageNames = []string{
	"ingest_proc_ns",
	"kafka1_ns",
	"engine_ns",
	"kafka2_ns",
	"gateway_ns",
	"end_to_end_ns",
}

// pickStage returns the duration value for a given stage key from a Sample.
func pickStage(s Sample, stage string) int64 {
	switch stage {
	case "ingest_proc_ns":
		return s.IngestProcNs
	case "kafka1_ns":
		return s.Kafka1Ns
	case "engine_ns":
		return s.EngineNs
	case "kafka2_ns":
		return s.Kafka2Ns
	case "gateway_ns":
		return s.GatewayNs
	case "end_to_end_ns":
		return s.EndToEndNs
	}
	return 0
}

// ringWindow is a fixed-capacity ring buffer of int64 latency samples,
// used by the reporter goroutine (single-writer, no locking needed).
type ringWindow struct {
	buf  []int64
	next int
	size int
}

func newRingWindow(cap int) *ringWindow {
	return &ringWindow{buf: make([]int64, cap)}
}

func (w *ringWindow) push(v int64) {
	w.buf[w.next] = v
	w.next = (w.next + 1) % len(w.buf)
	if w.size < len(w.buf) {
		w.size++
	}
}

// percentiles returns the P50/P90/P99 of currently held samples (>=0 only).
// It allocates and sorts a copy; callers should invoke after a flush batch,
// not per-sample, to amortize cost.
func (w *ringWindow) percentiles() (p50, p90, p99 int64) {
	if w.size == 0 {
		return 0, 0, 0
	}
	tmp := make([]int64, 0, w.size)
	for i := 0; i < w.size; i++ {
		v := w.buf[i]
		if v > 0 {
			tmp = append(tmp, v)
		}
	}
	if len(tmp) == 0 {
		return 0, 0, 0
	}
	sort.Slice(tmp, func(i, j int) bool { return tmp[i] < tmp[j] })
	pick := func(p float64) int64 {
		idx := int(float64(len(tmp)-1) * p)
		return tmp[idx]
	}
	return pick(0.50), pick(0.90), pick(0.99)
}

// Sample represents a full end-to-end latency record for a single signal.
// All time fields are nanosecond wall-clock timestamps; *_ns suffixes denote durations.
type Sample struct {
	Symbol     string `json:"symbol"`
	SignalType string `json:"signal_type"`

	T0IngestInNs  int64 `json:"t0_ingest_in_ns"`
	T1IngestOutNs int64 `json:"t1_ingest_out_ns"`
	T2EngineInNs  int64 `json:"t2_engine_in_ns"`
	T3EngineOutNs int64 `json:"t3_engine_out_ns"`
	T4GwInNs      int64 `json:"t4_gw_in_ns"`
	T5GwOutNs     int64 `json:"t5_gw_out_ns"`

	IngestProcNs int64 `json:"ingest_proc_ns"`
	Kafka1Ns     int64 `json:"kafka1_ns"`
	EngineNs     int64 `json:"engine_ns"`
	Kafka2Ns     int64 `json:"kafka2_ns"`
	GatewayNs    int64 `json:"gateway_ns"`
	EndToEndNs   int64 `json:"end_to_end_ns"`

	TsMs int64 `json:"ts_ms"`
}

type Reporter struct {
	cli     *redis.Client
	queue   chan Sample
	stop    chan struct{}
	dropped atomic.Int64

	// windows holds rolling samples per stage; only accessed by run() goroutine.
	windows map[string]*ringWindow
}

var (
	defaultReporter *Reporter
	once            sync.Once
)

// Init creates the singleton reporter and starts the background flush loop.
func Init(cli *redis.Client) {
	once.Do(func() {
		windows := make(map[string]*ringWindow, len(stageNames))
		for _, name := range stageNames {
			windows[name] = newRingWindow(percentileWindow)
		}
		defaultReporter = &Reporter{
			cli:     cli,
			queue:   make(chan Sample, queueCapacity),
			stop:    make(chan struct{}),
			windows: windows,
		}
		go defaultReporter.run()
		log.Infof("latency reporter started")
	})
}

func Submit(s Sample) {
	if defaultReporter == nil {
		return
	}
	select {
	case defaultReporter.queue <- s:
	default:
		dropped := defaultReporter.dropped.Add(1)
		if dropped%1000 == 1 {
			log.Warnf("latency reporter queue full, dropped=%d", dropped)
		}
	}
}

// Close stops the background goroutine.
func Close() {
	if defaultReporter == nil {
		return
	}
	close(defaultReporter.stop)
}

func (r *Reporter) run() {
	ctx := context.Background()
	flushTimer := time.NewTimer(500 * time.Millisecond)
	defer flushTimer.Stop()
	batch := make([]Sample, 0, 128)

	for {
		select {
		case <-r.stop:
			return
		case s := <-r.queue:
			batch = append(batch, s)
			if len(batch) >= 128 {
				r.flush(ctx, batch)
				batch = batch[:0]
			}
		case <-flushTimer.C:
			if len(batch) > 0 {
				r.flush(ctx, batch)
				batch = batch[:0]
			}
			flushTimer.Reset(500 * time.Millisecond)
		}
	}
}

func (r *Reporter) flush(ctx context.Context, batch []Sample) {
	pipe := r.cli.Pipeline()
	for _, s := range batch {
		buf, err := json.Marshal(s)
		if err != nil {
			continue
		}
		pipe.RPush(ctx, RedisLatencySamplesKey, buf)
	}
	pipe.LTrim(ctx, RedisLatencySamplesKey, -SamplesMaxLen, -1)

	last := batch[len(batch)-1]
	pipe.HSet(ctx, RedisLatencyLatestKey, sampleToMap(last))

	for _, s := range batch {
		if s.T0IngestInNs <= 0 {
			continue
		}
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "ingest_proc_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "ingest_proc_ns:sum_ns", s.IngestProcNs)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "kafka1_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "kafka1_ns:sum_ns", s.Kafka1Ns)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "engine_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "engine_ns:sum_ns", s.EngineNs)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "kafka2_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "kafka2_ns:sum_ns", s.Kafka2Ns)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "gateway_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "gateway_ns:sum_ns", s.GatewayNs)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "end_to_end_ns:count", 1)
		pipe.HIncrBy(ctx, RedisLatencyStatsKey, "end_to_end_ns:sum_ns", s.EndToEndNs)

		// feed rolling windows for percentile computation
		for _, name := range stageNames {
			if w, ok := r.windows[name]; ok {
				if v := pickStage(s, name); v > 0 {
					w.push(v)
				}
			}
		}
	}

	// compute & write percentiles per stage (overwrite each flush)
	for _, name := range stageNames {
		w, ok := r.windows[name]
		if !ok {
			continue
		}
		p50, p90, p99 := w.percentiles()
		pipe.HSet(ctx, RedisLatencyStatsKey,
			name+":p50_ns", strconv.FormatInt(p50, 10),
			name+":p90_ns", strconv.FormatInt(p90, 10),
			name+":p99_ns", strconv.FormatInt(p99, 10),
		)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Errorf("latency reporter flush failed batch_size=%d err=%v", len(batch), err)
	}
}

func sampleToMap(s Sample) map[string]string {
	return map[string]string{
		"symbol":           s.Symbol,
		"signal_type":      s.SignalType,
		"t0_ingest_in_ns":  strconv.FormatInt(s.T0IngestInNs, 10),
		"t1_ingest_out_ns": strconv.FormatInt(s.T1IngestOutNs, 10),
		"t2_engine_in_ns":  strconv.FormatInt(s.T2EngineInNs, 10),
		"t3_engine_out_ns": strconv.FormatInt(s.T3EngineOutNs, 10),
		"t4_gw_in_ns":      strconv.FormatInt(s.T4GwInNs, 10),
		"t5_gw_out_ns":     strconv.FormatInt(s.T5GwOutNs, 10),
		"ingest_proc_ns":   strconv.FormatInt(s.IngestProcNs, 10),
		"kafka1_ns":        strconv.FormatInt(s.Kafka1Ns, 10),
		"engine_ns":        strconv.FormatInt(s.EngineNs, 10),
		"kafka2_ns":        strconv.FormatInt(s.Kafka2Ns, 10),
		"gateway_ns":       strconv.FormatInt(s.GatewayNs, 10),
		"end_to_end_ns":    strconv.FormatInt(s.EndToEndNs, 10),
		"ts_ms":            strconv.FormatInt(s.TsMs, 10),
	}
}
