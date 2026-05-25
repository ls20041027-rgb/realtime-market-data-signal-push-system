package model

type MarketSnapshotPayload struct {
	Symbol         string       `json:"symbol"`
	DataType       string       `json:"data_type"`
	Exchange       string       `json:"exchange"`
	EventTime      int64        `json:"event_time"`
	LastPrice      int64        `json:"last_price"`
	PrevClose      int64        `json:"prev_close,omitempty"`
	OpenPrice      int64        `json:"open_price,omitempty"`
	HighPrice      int64        `json:"high_price,omitempty"`
	LowPrice       int64        `json:"low_price,omitempty"`
	Volume         int64        `json:"volume"`
	Turnover       int64        `json:"turnover"`
	BidLevels      []PriceLevel `json:"bid_levels,omitempty"`
	AskLevels      []PriceLevel `json:"ask_levels,omitempty"`
	RawMessageType string       `json:"raw_message_type,omitempty"`
}

type PriceLevel struct {
	Price  int64 `json:"price"`
	Volume int64 `json:"volume"`
}

type TradingSignalPayload struct {
	Symbol       string                 `json:"symbol"`
	SignalID     string                 `json:"signal_id"`
	SignalType   string                 `json:"signal_type"`
	Action       string                 `json:"action"`
	StrategyName string                 `json:"strategy_name"`
	Confidence   string                 `json:"confidence"`
	SignalTime   int64                  `json:"signal_time"`
	TriggerPrice int64                  `json:"trigger_price"`
	Reason       string                 `json:"reason"`
	Severity     string                 `json:"severity,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	Indicators   map[string]interface{} `json:"indicators,omitempty"`

	// 延迟追踪用透传字段（仅用于计时，不参与业务逻辑）。
	T0IngestInNs  int64 `json:"_t0_ingest_in_ns,omitempty"`
	T1IngestOutNs int64 `json:"_t1_ingest_out_ns,omitempty"`
	T2EngineInNs  int64 `json:"_t2_engine_in_ns,omitempty"`
	T3EngineOutNs int64 `json:"_t3_engine_out_ns,omitempty"`
}
