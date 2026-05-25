// WebSocket benchmark tool for push_gateway.
// Measures signal push latency under high connection counts.
//
// It embeds a Kafka producer that sends simulated market data at a configurable rate,
// so all subscribed WS clients receive pushes and we can measure broadcast latency.
//
// Usage:
//
//	go run ./cmd/wsbench -c 1000 -d 30s
//	go run ./cmd/wsbench -c 5000 -d 30s -ramp 10s
//	go run ./cmd/wsbench -c 10000 -d 30s -ramp 15s
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

// ServerPush mirrors the server's push message format.
type ServerPush struct {
	Channel string          `json:"channel"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Ts      int64           `json:"ts"`
}

// MarketSnapshotPayload for Kafka producer
type MarketSnapshotPayload struct {
	Symbol    string `json:"symbol"`
	DataType  string `json:"data_type"`
	Exchange  string `json:"exchange"`
	EventTime int64  `json:"event_time"`
	LastPrice int64  `json:"last_price"`
	PrevClose int64  `json:"prev_close"`
	OpenPrice int64  `json:"open_price"`
	HighPrice int64  `json:"high_price"`
	LowPrice  int64  `json:"low_price"`
	Volume    int64  `json:"volume"`
	Turnover  int64  `json:"turnover"`
}

// TradingSignalPayload for Kafka producer (signal mode)
type TradingSignalPayload struct {
	Symbol       string `json:"symbol"`
	SignalID     string `json:"signal_id"`
	SignalType   string `json:"signal_type"`
	Action       string `json:"action"`
	StrategyName string `json:"strategy_name"`
	Confidence   string `json:"confidence"`
	SignalTime   int64  `json:"signal_time"`
	TriggerPrice int64  `json:"trigger_price"`
	Reason       string `json:"reason"`
}

// StatusResponse for /api/status
type StatusResponse struct {
	Data struct {
		WS struct {
			Clients     int   `json:"clients"`
			Channels    int   `json:"channels"`
			DroppedSlow int64 `json:"dropped_slow"`
		} `json:"ws"`
		Runtime struct {
			Goroutines int   `json:"goroutines"`
			PID        int   `json:"pid"`
			Uptime     int64 `json:"uptime_seconds"`
		} `json:"runtime"`
		Resources []struct {
			Service    string  `json:"service"`
			Up         bool    `json:"up"`
			CPUPercent float64 `json:"cpu_percent"`
			RSSBytes   int64   `json:"rss_bytes"`
		} `json:"resources"`
	} `json:"data"`
}

func main() {
	addr := flag.String("addr", "ws://127.0.0.1:8080/ws/market", "WebSocket server address")
	statusURL := flag.String("status", "http://127.0.0.1:8080/api/status", "Status API URL")
	kafkaBroker := flag.String("kafka", "127.0.0.1:9092", "Kafka broker address")
	kafkaTopic := flag.String("topic", "", "Kafka topic (auto-detected by mode if empty)")
	conns := flag.Int("c", 1000, "Number of concurrent WebSocket connections")
	duration := flag.Duration("d", 30*time.Second, "Test duration after all connections established")
	rampUp := flag.Duration("ramp", 5*time.Second, "Ramp-up time to establish all connections")
	pushRate := flag.Int("rate", 5, "Messages per second to push via Kafka")
	symbol := flag.String("symbol", "SZ000001", "Symbol to subscribe and push")
	mode := flag.String("mode", "quote", "Benchmark mode: quote (with 200ms flush) or signal (instant push)")
	flag.Parse()

	// Determine topic and channels based on mode
	var channels []string
	if *mode == "signal" {
		channels = []string{"signal:ALL"}
		if *kafkaTopic == "" {
			*kafkaTopic = "trading_signals"
		}
	} else {
		channels = []string{"quote:" + *symbol}
		if *kafkaTopic == "" {
			*kafkaTopic = "market_data_normalized"
		}
	}

	fmt.Printf("╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║   WebSocket Benchmark - Push Gateway              ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Mode:        %s\n", *mode)
	fmt.Printf("║ Target:      %s\n", *addr)
	fmt.Printf("║ Connections: %d\n", *conns)
	fmt.Printf("║ Duration:    %s\n", *duration)
	fmt.Printf("║ Ramp-up:     %s\n", *rampUp)
	fmt.Printf("║ Channels:    %v\n", channels)
	fmt.Printf("║ Push Rate:   %d msg/s via Kafka\n", *pushRate)
	fmt.Printf("║ Kafka:       %s → %s\n", *kafkaBroker, *kafkaTopic)
	fmt.Printf("╚══════════════════════════════════════════════════╝\n\n")

	// Metrics
	var (
		connected  atomic.Int64
		connErrors atomic.Int64
		msgCount   atomic.Int64
		readErrors atomic.Int64
		latencies  []int64 // microseconds
		latencyMu  sync.Mutex
		done       = make(chan struct{})
	)

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\n⚠️  Interrupted, collecting results...")
		close(done)
	}()

	// Ramp-up connections
	interval := *rampUp / time.Duration(*conns)
	if interval < 50*time.Microsecond {
		interval = 50 * time.Microsecond
	}

	fmt.Printf("⏳ Establishing %d connections (interval=%s)...\n", *conns, interval)
	startTime := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < *conns; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runClient(id, *addr, channels, done, &connected, &connErrors, &msgCount, &readErrors, &latencies, &latencyMu)
		}(i)
		time.Sleep(interval)

		// Progress report every 10%
		step := *conns / 10
		if step == 0 {
			step = 1
		}
		if (i+1)%step == 0 {
			fmt.Printf("  ... %d/%d connected (errors: %d)\n", connected.Load(), *conns, connErrors.Load())
		}
	}

	rampDone := time.Since(startTime)
	fmt.Printf("\n✅ Ramp-up complete in %s: %d/%d connected, %d errors\n",
		rampDone.Round(time.Millisecond), connected.Load(), *conns, connErrors.Load())

	// Collect resource snapshot at start
	fmt.Printf("\n📊 Server status at test start:\n")
	printStatus(*statusURL)

	// Start Kafka producer
	fmt.Printf("\n🚀 Starting Kafka producer (%d msg/s, mode=%s)...\n", *pushRate, *mode)
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		if *mode == "signal" {
			runSignalProducer(*kafkaBroker, *kafkaTopic, *symbol, *pushRate, done)
		} else {
			runProducer(*kafkaBroker, *kafkaTopic, *symbol, *pushRate, done)
		}
	}()

	// Wait for test duration
	fmt.Printf("⏱️  Running test for %s...\n", *duration)
	select {
	case <-time.After(*duration):
		close(done)
	case <-done:
	}

	// Wait for producer to stop
	<-producerDone
	time.Sleep(1 * time.Second)

	// Collect resource snapshot at end
	fmt.Printf("\n📊 Server status at test end:\n")
	printStatus(*statusURL)

	// Calculate results
	totalMsgs := msgCount.Load()
	elapsed := time.Since(startTime)

	latencyMu.Lock()
	samples := make([]int64, len(latencies))
	copy(samples, latencies)
	latencyMu.Unlock()

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║              RESULTS                              ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Duration:        %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("║ Connections:     %d established / %d target\n", connected.Load(), *conns)
	fmt.Printf("║ Conn Errors:     %d\n", connErrors.Load())
	fmt.Printf("║ Messages Recv:   %d (%.0f msg/s total across all clients)\n", totalMsgs, float64(totalMsgs)/(*duration).Seconds())
	fmt.Printf("║ Read Errors:     %d\n", readErrors.Load())
	fmt.Printf("╠══════════════════════════════════════════════════╣\n")

	if len(samples) > 0 {
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		n := len(samples)
		var sum int64
		for _, v := range samples {
			sum += v
		}
		avg := float64(sum) / float64(n) / 1000.0 // µs to ms
		p50 := float64(samples[n*50/100]) / 1000.0
		p90 := float64(samples[n*90/100]) / 1000.0
		p99 := float64(samples[n*99/100]) / 1000.0
		minL := float64(samples[0]) / 1000.0
		maxL := float64(samples[n-1]) / 1000.0

		fmt.Printf("║ Push Latency (server ts → client recv):           \n")
		fmt.Printf("║   Samples:  %d\n", n)
		fmt.Printf("║   Avg:      %.2f ms\n", avg)
		fmt.Printf("║   P50:      %.2f ms\n", p50)
		fmt.Printf("║   P90:      %.2f ms\n", p90)
		fmt.Printf("║   P99:      %.2f ms\n", p99)
		fmt.Printf("║   Min:      %.2f ms\n", minL)
		fmt.Printf("║   Max:      %.2f ms\n", maxL)
	} else {
		fmt.Printf("║ ⚠️  No latency samples collected (no messages received)\n")
	}
	fmt.Printf("╚══════════════════════════════════════════════════╝\n")

	// Wait for goroutines to exit
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️  Some goroutines did not exit in time")
	}
}

func runClient(
	id int,
	addr string,
	channels []string,
	done <-chan struct{},
	connected, connErrors, msgCount, readErrors *atomic.Int64,
	latencies *[]int64,
	latencyMu *sync.Mutex,
) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(addr, nil)
	if err != nil {
		connErrors.Add(1)
		return
	}
	connected.Add(1)
	defer func() {
		conn.Close()
		connected.Add(-1)
	}()

	// Subscribe to channels
	subMsg := map[string]interface{}{
		"action":   "subscribe",
		"channels": channels,
	}
	if err := conn.WriteJSON(subMsg); err != nil {
		connErrors.Add(1)
		return
	}

	// Read loop
	for {
		select {
		case <-done:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-done:
				return
			default:
			}
			readErrors.Add(1)
			return
		}

		recvTime := time.Now().UnixMilli()
		msgCount.Add(1)

		// Parse ts field for latency measurement
		var push ServerPush
		if err := json.Unmarshal(raw, &push); err != nil {
			continue
		}

		// Only measure latency for actual data pushes (not pong etc)
		if push.Ts > 0 && push.Type != "" && push.Type != "pong" && push.Type != "error" {
			latencyMs := recvTime - push.Ts
			latencyUs := latencyMs * 1000
			if latencyUs >= 0 && latencyUs < 60_000_000 { // sanity: < 60s
				latencyMu.Lock()
				*latencies = append(*latencies, latencyUs)
				latencyMu.Unlock()
			}
		}
	}
}

func runProducer(broker, topic, symbol string, rate int, done <-chan struct{}) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 5 * time.Millisecond,
		BatchSize:    1,
	}
	defer w.Close()

	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	seq := int64(0)
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			seq++
			payload := MarketSnapshotPayload{
				Symbol:    symbol,
				DataType:  "snapshot",
				Exchange:  "SZ",
				EventTime: time.Now().UnixMilli(),
				LastPrice: 100000 + seq%1000, // simulate price fluctuation (in 1/10000 yuan)
				PrevClose: 100000,
				OpenPrice: 100100,
				HighPrice: 101000,
				LowPrice:  99500,
				Volume:    1000000 + seq*100,
				Turnover:  500000000 + seq*50000,
			}
			data, _ := json.Marshal(payload)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := w.WriteMessages(ctx, kafka.Message{
				Key:   []byte(symbol),
				Value: data,
			})
			cancel()
			if err != nil {
				// Silently ignore producer errors during shutdown
				select {
				case <-done:
					return
				default:
					fmt.Printf("  ⚠️  Kafka write error: %v\n", err)
				}
			}
		}
	}
}

func runSignalProducer(broker, topic, symbol string, rate int, done <-chan struct{}) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 5 * time.Millisecond,
		BatchSize:    1,
	}
	defer w.Close()

	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	seq := int64(0)
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			seq++
			payload := TradingSignalPayload{
				Symbol:       symbol,
				SignalID:     fmt.Sprintf("bench-%d", seq),
				SignalType:   "BENCH_TEST",
				Action:       "BUY",
				StrategyName: "bench_strategy",
				Confidence:   "0.95",
				SignalTime:   time.Now().UnixMilli(),
				TriggerPrice: 100000 + seq%1000,
				Reason:       "benchmark test signal",
			}
			data, _ := json.Marshal(payload)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := w.WriteMessages(ctx, kafka.Message{
				Key:   []byte(symbol),
				Value: data,
			})
			cancel()
			if err != nil {
				select {
				case <-done:
					return
				default:
					fmt.Printf("  ⚠️  Kafka write error: %v\n", err)
				}
			}
		}
	}
}

func printStatus(url string) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("  ⚠️  Cannot reach status API: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var status StatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		fmt.Printf("  ⚠️  Cannot parse status: %v\n", err)
		return
	}

	fmt.Printf("  WS Clients:    %d\n", status.Data.WS.Clients)
	fmt.Printf("  WS Channels:   %d\n", status.Data.WS.Channels)
	fmt.Printf("  Dropped Slow:  %d\n", status.Data.WS.DroppedSlow)
	fmt.Printf("  Goroutines:    %d\n", status.Data.Runtime.Goroutines)
	for _, r := range status.Data.Resources {
		if r.Service == "push_gateway" && r.Up {
			fmt.Printf("  CPU:           %.1f%%\n", r.CPUPercent)
			fmt.Printf("  RSS:           %.1f MB\n", float64(r.RSSBytes)/1024/1024)
		}
	}
}

// Suppress unused import
var _ = strings.Split
