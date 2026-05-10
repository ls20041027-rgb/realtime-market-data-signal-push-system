package kafkax

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"push_gateway/config"
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
		out = append(out, TopicStats{
			Topic:     topic,
			Partition: s.Partition,
			Lag:       s.Lag,
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
			cfg.TopicSystemEvents:         newReader(cfg.TopicSystemEvents),
		},
	}
}

func (c *Consumer) Run(ctx context.Context) {
	slog.Info("kafka consumer starting",
		"component", "kafka", "group_id", c.cfg.GroupID,
		"brokers", c.cfg.BootstrapServers,
		"topics", []string{
			c.cfg.TopicMarketDataNormalized,
			c.cfg.TopicTradingSignals,
			c.cfg.TopicSystemEvents,
		},
	)
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
	slog.Info("kafka consumer stopped", "component", "kafka")
}

func (c *Consumer) Close() error {
	var firstErr error
	for topic, r := range c.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
			slog.Error("kafka reader close failed",
				"component", "kafka", "topic", topic, "err", err)
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
			slog.Error("kafka read failed, backing off",
				"component", "kafka", "topic", topic,
				"backoff_ms", backoff.Milliseconds(), "err", err)
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
		c.dispatch(topic, msg)
	}
}

func (c *Consumer) dispatch(topic string, msg kafka.Message) {
	var env model.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.Warn("kafka envelope decode failed, skip",
			"component", "kafka", "topic", topic,
			"offset", msg.Offset, "err", err)
		return
	}

	ts := time.Now().UnixMilli()

	switch env.MessageType {
	case "MARKET_SNAPSHOT_NORMALIZED":
		payload, err := model.ParseMarketSnapshot(env.Payload)
		if err != nil {
			slog.Warn("decode market snapshot failed",
				"component", "kafka", "symbol", env.Symbol, "err", err)
			return
		}
		c.hub.Broadcast(model.WSChannelQuotePrefix+env.Symbol, model.ServerPush{
			Channel: model.WSChannelQuotePrefix + env.Symbol,
			Type:    model.WSTypeMarketSnapshot,
			Data:    payload,
			Ts:      ts,
		})

	case "TRADING_SIGNAL":
		payload, err := model.ParseTradingSignal(env.Payload)
		if err != nil {
			slog.Warn("decode trading signal failed",
				"component", "kafka", "symbol", env.Symbol, "err", err)
			return
		}
		perSymbol := model.WSChannelSignalPrefix + env.Symbol
		push := model.ServerPush{Type: model.WSTypeTradingSignal, Data: payload, Ts: ts}
		push.Channel = perSymbol
		c.hub.Broadcast(perSymbol, push)
		push.Channel = model.WSChannelSignalAll
		c.hub.Broadcast(model.WSChannelSignalAll, push)

	case "SERVICE_INFO", "SERVICE_WARNING", "SERVICE_ERROR", "SERVICE_CRITICAL":
		payload, err := model.ParseSystemEvent(env.Payload)
		if err != nil {
			slog.Warn("decode system event failed",
				"component", "kafka", "err", err)
			return
		}
		c.hub.Broadcast(model.WSChannelSystemEvents, model.ServerPush{
			Channel: model.WSChannelSystemEvents,
			Type:    model.WSTypeSystemEvent,
			Data:    payload,
			Ts:      ts,
		})

	default:
		slog.Warn("unknown message_type, skip",
			"component", "kafka", "topic", topic,
			"message_type", env.MessageType, "symbol", env.Symbol)
	}
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
