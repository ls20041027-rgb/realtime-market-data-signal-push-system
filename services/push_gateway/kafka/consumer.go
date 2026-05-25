package kafkax

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/segmentio/kafka-go"

	"push_gateway/config"
	log "push_gateway/internal/log"
	"push_gateway/model"
)

type Broadcaster interface {
	Broadcast(channel string, push model.ServerPush)
}

type Consumer struct {
	cfg     config.KafkaConfig
	readers map[string]*kafka.Reader
	hub     Broadcaster
}

type TopicStats struct {
	Topic     string `json:"topic"`
	Partition string `json:"partition"`
	Lag       int64  `json:"lag"`
	Offset    int64  `json:"offset"`
	Messages  int64  `json:"messages"`
	Errors    int64  `json:"errors"`
}

func (c *Consumer) Stats() []TopicStats {
	out := make([]TopicStats, 0, len(c.readers))
	for topic, r := range c.readers {
		s := r.Stats()
		// 直接使用Partition字段，它已经是string类型
		partitionStr := s.Partition
		// 如果分区显示为-1，可能是kafka-go的默认值，我们改为显示0
		if partitionStr == "-1" {
			partitionStr = "0"
		}
		lag := s.Lag
		if lag < 0 {
			lag = 0 // 当Lag为-1时，显示为0
		}
		out = append(out, TopicStats{
			Topic:     topic,
			Partition: partitionStr,
			Lag:       lag,
			Offset:    s.Offset,
			Messages:  s.Messages,
			Errors:    s.Errors,
		})
	}
	return out
}

func NewConsumer(cfg config.KafkaConfig, hub Broadcaster) *Consumer {
	newReader := func(topic string) *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.BootstrapServers,
			GroupID:        cfg.GroupID,
			Topic:          topic,
			MinBytes:       cfg.MinBytes,
			MaxBytes:       cfg.MaxBytes,
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		})
	}
	return &Consumer{
		cfg: cfg,
		hub: hub,
		readers: map[string]*kafka.Reader{
			cfg.TopicMarketDataNormalized: newReader(cfg.TopicMarketDataNormalized),
			cfg.TopicTradingSignals:       newReader(cfg.TopicTradingSignals),
		},
	}
}

func (c *Consumer) Run(ctx context.Context) {
	log.Infof("kafka consumer starting group_id=%s brokers=%v topics=[%s,%s]",
		c.cfg.GroupID, c.cfg.BootstrapServers,
		c.cfg.TopicMarketDataNormalized, c.cfg.TopicTradingSignals)
	done := make(chan struct{}, len(c.readers))
	for topic, reader := range c.readers {
		go func(topic string, r *kafka.Reader) {
			defer func() { done <- struct{}{} }()
			c.consumeLoop(ctx, topic, r)
		}(topic, reader)
	}
	for i := 0; i < len(c.readers); i++ {
		<-done
	}
	log.Infof("kafka consumer stopped")
}

func (c *Consumer) Close() error {
	var firstErr error
	for topic, r := range c.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
			log.Errorf("kafka reader close failed topic=%s err=%v", topic, err)
		}
	}
	return firstErr
}

func (c *Consumer) consumeLoop(ctx context.Context, topic string, r *kafka.Reader) {
	backoff := c.cfg.ReconnectWait
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	maxBackoff := c.cfg.ReconnectMaxWait
	if maxBackoff <= 0 {
		maxBackoff = 60 * time.Second
	}

	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return
			}
			log.Errorf("kafka read failed, backing off topic=%s backoff_ms=%d err=%v",
				topic, backoff.Milliseconds(), err)
			if !sleepOrDone(ctx, backoff) {
				return
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		backoff = c.cfg.ReconnectWait
		if backoff <= 0 {
			backoff = 2 * time.Second
		}
		switch topic {
		case c.cfg.TopicMarketDataNormalized:
			c.handleMarketSnapshot(msg)
		case c.cfg.TopicTradingSignals:
			c.handleTradingSignal(msg)
		}
	}
}

func (c *Consumer) handleMarketSnapshot(msg kafka.Message) {
	var payload model.MarketSnapshotPayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		log.Warnf("decode market snapshot failed offset=%d err=%v", msg.Offset, err)
		return
	}
	c.hub.Broadcast(model.WSChannelQuotePrefix+payload.Symbol, model.ServerPush{
		Channel: model.WSChannelQuotePrefix + payload.Symbol,
		Type:    model.WSTypeMarketSnapshot,
		Data:    &payload,
		Ts:      time.Now().UnixMilli(),
	})
}

func (c *Consumer) handleTradingSignal(msg kafka.Message) {
	var payload model.TradingSignalPayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		log.Warnf("decode trading signal failed offset=%d err=%v", msg.Offset, err)
		return
	}
	t4 := time.Now().UnixNano()

	var trace *model.LatencyTrace
	if payload.T0IngestInNs > 0 {
		trace = &model.LatencyTrace{
			Symbol:        payload.Symbol,
			SignalType:    payload.SignalType,
			T0IngestInNs:  payload.T0IngestInNs,
			T1IngestOutNs: payload.T1IngestOutNs,
			T2EngineInNs:  payload.T2EngineInNs,
			T3EngineOutNs: payload.T3EngineOutNs,
			T4GwInNs:      t4,
		}
	}

	ts := time.Now().UnixMilli()
	perSymbol := model.WSChannelSignalPrefix + payload.Symbol
	push := model.ServerPush{Type: model.WSTypeTradingSignal, Data: &payload, Ts: ts, Latency: trace}
	push.Channel = perSymbol
	c.hub.Broadcast(perSymbol, push)
	push.Channel = model.WSChannelSignalAll
	c.hub.Broadcast(model.WSChannelSignalAll, push)
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
