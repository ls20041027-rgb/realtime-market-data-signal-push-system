package ws

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"push_gateway/config"
)

var connCounter atomic.Int64

func HandleWS(hub *Hub, cfg config.WSConfig) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return func(gc *gin.Context) {
		if cfg.MaxConnections > 0 && connCounter.Load() >= int64(cfg.MaxConnections) {
			gc.JSON(http.StatusServiceUnavailable, gin.H{
				"code": 50003, "message": "too many ws connections",
			})
			return
		}

		conn, err := upgrader.Upgrade(gc.Writer, gc.Request, nil)
		if err != nil {
			slog.Warn("ws upgrade failed",
				"component", "ws", "err", err, "remote", gc.Request.RemoteAddr)
			return
		}
		connCounter.Add(1)
		defer connCounter.Add(-1)

		client := NewClient(conn, hub, cfg)
		hub.Register(client)

		slog.Info("ws client connected",
			"component", "ws", "client_id", client.ID(), "remote", client.addr,
			"active", connCounter.Load())

		go client.WritePump(gc.Request.Context())
		client.ReadPump(gc.Request.Context())
	}
}

func ActiveConnections() int64 { return connCounter.Load() }
