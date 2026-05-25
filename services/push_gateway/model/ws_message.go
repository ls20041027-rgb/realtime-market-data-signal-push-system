package model

type ClientRequest struct {
	Action   string   `json:"action"`
	Channels []string `json:"channels,omitempty"`
}

// LatencyTrace 透传端到端延迟追踪上下文。为避免循环 import，这里只存必要的原始时间戳与样本元数据，
// 在 WritePump 实际发出后由 ws 层拼装为 latency.Sample 并提交。
type LatencyTrace struct {
	Symbol        string
	SignalType    string
	T0IngestInNs  int64
	T1IngestOutNs int64
	T2EngineInNs  int64
	T3EngineOutNs int64
	T4GwInNs      int64
}

type ServerPush struct {
	Channel string      `json:"channel"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Ts      int64       `json:"ts"`

	// Latency 仅服务端内部透传，不序列化给客户端。
	Latency *LatencyTrace `json:"-"`
}

type ServerPong struct {
	Type string `json:"type"`
}

type ServerError struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	WSTypeMarketSnapshot = "MARKET_SNAPSHOT"
	WSTypeTradingSignal  = "TRADING_SIGNAL"
	WSTypePong           = "pong"
	WSTypeError          = "error"
)

const (
	WSChannelQuotePrefix  = "quote:"
	WSChannelSignalPrefix = "signal:"
	WSChannelSignalAll    = "signal:ALL"
)
