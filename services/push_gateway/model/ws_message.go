package model

type ClientRequest struct {
	Action   string   `json:"action"`
	Channels []string `json:"channels,omitempty"`
}

type ServerPush struct {
	Channel string      `json:"channel"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Ts      int64       `json:"ts"`
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
	WSTypeSystemEvent    = "SYSTEM_EVENT"
	WSTypePong           = "pong"
	WSTypeError          = "error"
)

const (
	WSChannelQuotePrefix  = "quote:"
	WSChannelSignalPrefix = "signal:"
	WSChannelSignalAll    = "signal:ALL"
	WSChannelSystemEvents = "system:events"
)
