package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"push_gateway/config"
	"push_gateway/model"
)

var clientIDSeq atomic.Uint64

type Client struct {
	conn *websocket.Conn
	hub  *Hub
	cfg  config.WSConfig

	id   string
	addr string

	send chan model.ServerPush

	subs map[string]struct{}

	sendClosed atomic.Bool
}

func NewClient(conn *websocket.Conn, hub *Hub, cfg config.WSConfig) *Client {
	id := "c" + strconv.FormatUint(clientIDSeq.Add(1), 10)
	bufSize := cfg.ClientSendBuffer
	if bufSize <= 0 {
		bufSize = 256
	}
	return &Client{
		conn: conn,
		hub:  hub,
		cfg:  cfg,
		id:   id,
		addr: conn.RemoteAddr().String(),
		send: make(chan model.ServerPush, bufSize),
		subs: make(map[string]struct{}),
	}
}

func (c *Client) ID() string { return c.id }

func (c *Client) Subscriptions() map[string]struct{} {
	return c.subs
}

func (c *Client) AddSub(ch string) {
	c.subs[ch] = struct{}{}
}

func (c *Client) RemoveSub(ch string) {
	delete(c.subs, ch)
}

func (c *Client) sendNonBlocking(p model.ServerPush) bool {
	if c.sendClosed.Load() {
		return false
	}
	select {
	case c.send <- p:
		return true
	default:
		return false
	}
}

func (c *Client) closeSend() {
	if c.sendClosed.CompareAndSwap(false, true) {
		close(c.send)
	}
}

func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	readTimeout := c.cfg.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if !errors.Is(err, context.Canceled) && !websocket.IsCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Debug("ws read loop exit",
					"component", "ws", "client_id", c.id, "err", err)
			}
			return
		}

		var req model.ClientRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			c.sendNonBlocking(model.ServerPush{
				Channel: "",
				Type:    model.WSTypeError,
				Data: model.ServerError{
					Type: model.WSTypeError, Code: model.CodeInvalidParam,
					Message: "malformed request json",
				},
				Ts: time.Now().UnixMilli(),
			})
			continue
		}

		switch req.Action {
		case "subscribe":
			_ = c.hub.Subscribe(c, req.Channels)
		case "unsubscribe":
			c.hub.Unsubscribe(c, req.Channels)
		case "ping":
			c.sendNonBlocking(model.ServerPush{
				Type: model.WSTypePong,
				Data: model.ServerPong{Type: model.WSTypePong},
				Ts:   time.Now().UnixMilli(),
			})
		default:
			c.sendNonBlocking(model.ServerPush{
				Type: model.WSTypeError,
				Data: model.ServerError{
					Type: model.WSTypeError, Code: model.CodeInvalidParam,
					Message: "unknown action: " + req.Action,
				},
				Ts: time.Now().UnixMilli(),
			})
		}
	}
}

func (c *Client) WritePump(ctx context.Context) {
	pingInterval := c.cfg.PingInterval
	if pingInterval <= 0 {
		pingInterval = 25 * time.Second
	}
	writeTimeout := c.cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 5 * time.Second
	}

	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case push, ok := <-c.send:
			if !ok {
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				_ = c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutdown"))
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteJSON(push); err != nil {
				slog.Debug("ws write failed, closing client",
					"component", "ws", "client_id", c.id, "err", err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Debug("ws ping failed, closing client",
					"component", "ws", "client_id", c.id, "err", err)
				return
			}
		}
	}
}
