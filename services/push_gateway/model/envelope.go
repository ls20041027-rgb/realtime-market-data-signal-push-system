package model

import (
	"encoding/json"
	"fmt"
)

type Envelope struct {
	MessageType string          `json:"message_type"`
	Source      string          `json:"source"`
	Timestamp   string          `json:"timestamp"`
	Symbol      string          `json:"symbol"`
	Payload     json.RawMessage `json:"payload"`
}

type MarketSnapshotPayload struct {
	DataType       string       `json:"data_type"`
	Exchange       string       `json:"exchange"`
	EventTime      string       `json:"event_time"`
	LastPrice      string       `json:"last_price"`
	PrevClose      string       `json:"prev_close,omitempty"`
	OpenPrice      string       `json:"open_price,omitempty"`
	HighPrice      string       `json:"high_price,omitempty"`
	LowPrice       string       `json:"low_price,omitempty"`
	Volume         int64        `json:"volume"`
	Turnover       string       `json:"turnover"`
	BidLevels      []PriceLevel `json:"bid_levels,omitempty"`
	AskLevels      []PriceLevel `json:"ask_levels,omitempty"`
	RawMessageType string       `json:"raw_message_type,omitempty"`
}

type PriceLevel struct {
	Price  string `json:"price"`
	Volume int64  `json:"volume"`
}

type TradingSignalPayload struct {
	SignalID     string                 `json:"signal_id"`
	SignalType   string                 `json:"signal_type"`
	Action       string                 `json:"action"`
	StrategyName string                 `json:"strategy_name"`
	Confidence   string                 `json:"confidence"`
	SignalTime   string                 `json:"signal_time"`
	TriggerPrice string                 `json:"trigger_price"`
	Reason       string                 `json:"reason"`
	Severity     string                 `json:"severity,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	Indicators   map[string]interface{} `json:"indicators,omitempty"`
}

type SystemEventPayload struct {
	EventID      string                 `json:"event_id"`
	Service      string                 `json:"service"`
	Level        string                 `json:"level"`
	EventType    string                 `json:"event_type"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	RetryCount   int                    `json:"retry_count,omitempty"`
	RelatedTopic string                 `json:"related_topic,omitempty"`
}

func ParseMarketSnapshot(raw json.RawMessage) (*MarketSnapshotPayload, error) {
	var p MarketSnapshotPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decode market snapshot payload: %w", err)
	}
	return &p, nil
}

func ParseTradingSignal(raw json.RawMessage) (*TradingSignalPayload, error) {
	var p TradingSignalPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decode trading signal payload: %w", err)
	}
	return &p, nil
}

func ParseSystemEvent(raw json.RawMessage) (*SystemEventPayload, error) {
	var p SystemEventPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decode system event payload: %w", err)
	}
	return &p, nil
}
