// Package resource samples the local push_gateway process's CPU% / RSS and
// publishes the latest values to Redis hash `stream:resource:push_gateway`.
//
// Used by the ops dashboard StatusGrid to show resource usage of all services.
package resource

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/process"

	log "push_gateway/internal/log"
)

const (
	ServiceName    = "push_gateway"
	RedisKey       = "stream:resource:push_gateway"
	sampleInterval = 2 * time.Second
)

type Monitor struct {
	cli     *redis.Client
	proc    *process.Process
	stopped atomic.Bool
	stop    chan struct{}
}

func NewMonitor(cli *redis.Client) (*Monitor, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}
	// Prime CPUPercent so first call returns a meaningful value.
	_, _ = p.CPUPercent()
	return &Monitor{cli: cli, proc: p, stop: make(chan struct{})}, nil
}

// Start spawns the sampling goroutine. Safe to call only once.
func (m *Monitor) Start() {
	go m.run()
	log.Infof("resource monitor started, key=%s", RedisKey)
}

// Stop terminates the sampling goroutine.
func (m *Monitor) Stop() {
	if m.stopped.Swap(true) {
		return
	}
	close(m.stop)
}

func (m *Monitor) run() {
	t := time.NewTicker(sampleInterval)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			m.sampleOnce()
		}
	}
}

func (m *Monitor) sampleOnce() {
	cpu, err := m.proc.CPUPercent()
	if err != nil {
		log.Debugf("resource monitor cpu_percent failed: %v", err)
		return
	}
	memInfo, err := m.proc.MemoryInfo()
	if err != nil {
		log.Debugf("resource monitor memory_info failed: %v", err)
		return
	}
	threads, _ := m.proc.NumThreads()

	if m.cli == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := m.cli.HSet(ctx, RedisKey, map[string]any{
		"service":       ServiceName,
		"pid":           strconv.Itoa(os.Getpid()),
		"cpu_percent":   strconv.FormatFloat(cpu, 'f', 2, 64),
		"rss_bytes":     strconv.FormatUint(memInfo.RSS, 10),
		"num_threads":   strconv.FormatInt(int64(threads), 10),
		"updated_at_ms": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}).Err(); err != nil {
		log.Debugf("resource monitor flush failed: %v", err)
	}
}
