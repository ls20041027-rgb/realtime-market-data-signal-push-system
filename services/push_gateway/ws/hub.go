package ws

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"push_gateway/config"
	"push_gateway/model"
)

type BroadcastMsg struct {
	Channel string
	Push    model.ServerPush
}

type subscribeReq struct {
	client   *Client
	channels []string
	reply    chan []string
}

type HubStats struct {
	Clients      int   `json:"clients"`
	Channels     int   `json:"channels"`
	QuoteClients int   `json:"quote_clients"`
	DroppedSlow  int64 `json:"dropped_slow"`
}

type Hub struct {
	cfg config.WSConfig

	clients  map[*Client]struct{}
	channels map[string]map[*Client]struct{}
	pending  map[*Client]map[string]model.ServerPush

	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscribeReq
	unsubscribe chan subscribeReq
	broadcast   chan BroadcastMsg
	statsReq    chan chan HubStats
	closeOnce   chan struct{}

	droppedSlow atomic.Int64
}

func NewHub(cfg config.WSConfig) *Hub {
	return &Hub{
		cfg:         cfg,
		clients:     make(map[*Client]struct{}),
		channels:    make(map[string]map[*Client]struct{}),
		pending:     make(map[*Client]map[string]model.ServerPush),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		subscribe:   make(chan subscribeReq, 256),
		unsubscribe: make(chan subscribeReq, 256),
		broadcast:   make(chan BroadcastMsg, 4096),
		statsReq:    make(chan chan HubStats, 8),
		closeOnce:   make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	flushInterval := h.cfg.QuoteFlushInterval
	if flushInterval <= 0 {
		flushInterval = 200 * time.Millisecond
	}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	slog.Info("hub started", "component", "ws", "flush_interval_ms", flushInterval.Milliseconds())

	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return
		case <-h.closeOnce:
			h.shutdown()
			return
		case c := <-h.register:
			h.doRegister(c)
		case c := <-h.unregister:
			h.doUnregister(c)
		case req := <-h.subscribe:
			h.doSubscribe(req)
		case req := <-h.unsubscribe:
			h.doUnsubscribe(req)
		case msg := <-h.broadcast:
			h.doBroadcast(msg)
		case reply := <-h.statsReq:
			reply <- h.snapshotStats()
		case <-ticker.C:
			h.flushQuotes()
		}
	}
}


func (h *Hub) Register(c *Client) { h.register <- c }

func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	default:
	}
}

func (h *Hub) Subscribe(c *Client, channels []string) []string {
	reply := make(chan []string, 1)
	h.subscribe <- subscribeReq{client: c, channels: channels, reply: reply}
	return <-reply
}

func (h *Hub) Unsubscribe(c *Client, channels []string) {
	h.unsubscribe <- subscribeReq{client: c, channels: channels}
}

func (h *Hub) Broadcast(channel string, push model.ServerPush) {
	select {
	case h.broadcast <- BroadcastMsg{Channel: channel, Push: push}:
	default:
		h.droppedSlow.Add(1)
		slog.Warn("hub broadcast queue full, drop",
			"component", "ws", "channel", channel)
	}
}

func (h *Hub) Stats() HubStats {
	reply := make(chan HubStats, 1)
	select {
	case h.statsReq <- reply:
		return <-reply
	case <-time.After(200 * time.Millisecond):
		return HubStats{DroppedSlow: h.droppedSlow.Load()}
	}
}

func (h *Hub) Close() {
	select {
	case <-h.closeOnce:
	default:
		close(h.closeOnce)
	}
}

func IsValidChannel(ch string) bool {
	switch {
	case ch == model.WSChannelSignalAll:
		return true
	case ch == model.WSChannelSystemEvents:
		return true
	case strings.HasPrefix(ch, model.WSChannelQuotePrefix) && len(ch) > len(model.WSChannelQuotePrefix):
		return true
	case strings.HasPrefix(ch, model.WSChannelSignalPrefix) && len(ch) > len(model.WSChannelSignalPrefix):
		return true
	default:
		return false
	}
}


func (h *Hub) doRegister(c *Client) {
	h.clients[c] = struct{}{}
	slog.Info("ws client registered",
		"component", "ws", "client_id", c.ID(), "total_clients", len(h.clients))
}

func (h *Hub) doUnregister(c *Client) {
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	for ch := range c.Subscriptions() {
		if set, ok := h.channels[ch]; ok {
			delete(set, c)
			if len(set) == 0 {
				delete(h.channels, ch)
			}
		}
	}
	delete(h.pending, c)
	c.closeSend()
	slog.Info("ws client unregistered",
		"component", "ws", "client_id", c.ID(), "total_clients", len(h.clients))
}

func (h *Hub) doSubscribe(req subscribeReq) {
	accepted := make([]string, 0, len(req.channels))
	for _, ch := range req.channels {
		if !IsValidChannel(ch) {
			req.client.sendNonBlocking(model.ServerPush{
				Channel: ch,
				Type:    model.WSTypeError,
				Data: model.ServerError{
					Type: model.WSTypeError, Code: model.CodeInvalidChannel,
					Message: "invalid channel: " + ch,
				},
				Ts: time.Now().UnixMilli(),
			})
			continue
		}
		set, ok := h.channels[ch]
		if !ok {
			set = make(map[*Client]struct{})
			h.channels[ch] = set
		}
		set[req.client] = struct{}{}
		req.client.AddSub(ch)
		accepted = append(accepted, ch)
	}
	req.reply <- accepted
}

func (h *Hub) doUnsubscribe(req subscribeReq) {
	for _, ch := range req.channels {
		if set, ok := h.channels[ch]; ok {
			delete(set, req.client)
			if len(set) == 0 {
				delete(h.channels, ch)
			}
		}
		req.client.RemoveSub(ch)
	}
}

func (h *Hub) doBroadcast(msg BroadcastMsg) {
	set, ok := h.channels[msg.Channel]
	if !ok || len(set) == 0 {
		return
	}
	if strings.HasPrefix(msg.Channel, model.WSChannelQuotePrefix) {
		for c := range set {
			m, exists := h.pending[c]
			if !exists {
				m = make(map[string]model.ServerPush, 4)
				h.pending[c] = m
			}
			m[msg.Channel] = msg.Push
		}
		return
	}
	for c := range set {
		h.tryDeliver(c, msg.Push)
	}
}

func (h *Hub) flushQuotes() {
	if len(h.pending) == 0 {
		return
	}
	snapshot := h.pending
	h.pending = make(map[*Client]map[string]model.ServerPush, len(snapshot))

	for c, pushes := range snapshot {
		if _, alive := h.clients[c]; !alive {
			continue
		}
		for _, p := range pushes {
			h.tryDeliver(c, p)
		}
	}
}

func (h *Hub) tryDeliver(c *Client, p model.ServerPush) {
	if c.sendNonBlocking(p) {
		return
	}
	h.droppedSlow.Add(1)
	slog.Warn("slow consumer, drop frame",
		"component", "ws", "client_id", c.ID(), "channel", p.Channel)
}

func (h *Hub) snapshotStats() HubStats {
	quoteClients := 0
	seen := make(map[*Client]struct{}, len(h.clients))
	for ch, set := range h.channels {
		if !strings.HasPrefix(ch, model.WSChannelQuotePrefix) {
			continue
		}
		for c := range set {
			if _, ok := seen[c]; ok {
				continue
			}
			seen[c] = struct{}{}
			quoteClients++
		}
	}
	return HubStats{
		Clients:      len(h.clients),
		Channels:     len(h.channels),
		QuoteClients: quoteClients,
		DroppedSlow:  h.droppedSlow.Load(),
	}
}

func (h *Hub) shutdown() {
	slog.Info("hub shutting down",
		"component", "ws", "remaining_clients", len(h.clients))
	for c := range h.clients {
		c.closeSend()
	}
	h.clients = map[*Client]struct{}{}
	h.channels = map[string]map[*Client]struct{}{}
	h.pending = map[*Client]map[string]model.ServerPush{}
}
