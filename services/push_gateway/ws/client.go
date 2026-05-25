package ws

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"push_gateway/config"
	"push_gateway/internal/latency"
	log "push_gateway/internal/log"
	"push_gateway/model"

	"github.com/gorilla/websocket"
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
				log.Debugf("ws read loop exit client_id=%s err=%v", c.id, err)
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
				log.Debugf("ws write failed, closing client client_id=%s err=%v", c.id, err)
				return
			}
			// 延迟追踪：WriteJSON 返回后才是真正写完 socket 的时点，这里打 t5。
			if push.Latency != nil {
				latency.Submit(buildSample(push.Latency, time.Now().UnixNano()))
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Debugf("ws ping failed, closing client client_id=%s err=%v", c.id, err)
				return
			}
		}
	}
}

// buildSample 拼装一条端到端延迟样本。t5 在调用点已取过（WriteJSON 返回后）。
func buildSample(tr *model.LatencyTrace, t5 int64) latency.Sample {
	t0 := tr.T0IngestInNs
	t1 := tr.T1IngestOutNs
	t2 := tr.T2EngineInNs
	t3 := tr.T3EngineOutNs
	t4 := tr.T4GwInNs
	log.Infof("build latency sample, t0=%d, t1=%d, t2=%d, t3=%d, t4=%d, t5=%d", t0, t1, t2, t3, t4, t5)

	var ingestProc, kafka1, engine, kafka2, gateway int64
	if t1 > 0 && t0 > 0 {
		ingestProc = t1 - t0
	}
	if t2 > 0 && t1 > 0 {
		kafka1 = t2 - t1
	} else if t3 > 0 && t1 > 0 {
		// 降级：没有 t2 时把 kafka1 + engine 合并计入 kafka1
		kafka1 = t3 - t1
	}
	if t3 > 0 && t2 > 0 {
		engine = t3 - t2
	}
	if t4 > 0 && t3 > 0 {
		kafka2 = t4 - t3
	}
	if t5 > 0 && t4 > 0 {
		gateway = t5 - t4
	}

	var e2e int64
	if t0 > 0 && t5 > 0 {
		e2e = t5 - t0
	}

	return latency.Sample{
		Symbol:        tr.Symbol,
		SignalType:    tr.SignalType,
		T0IngestInNs:  t0,
		T1IngestOutNs: t1,
		T2EngineInNs:  t2,
		T3EngineOutNs: t3,
		T4GwInNs:      t4,
		T5GwOutNs:     t5,
		IngestProcNs:  ingestProc,
		Kafka1Ns:      kafka1,
		EngineNs:      engine,
		Kafka2Ns:      kafka2,
		GatewayNs:     gateway,
		EndToEndNs:    e2e,
		TsMs:          time.Now().UnixMilli(),
	}
}
