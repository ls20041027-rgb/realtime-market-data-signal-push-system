package ws

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"push_gateway/config"
	log "push_gateway/internal/log"
)

var connCounter atomic.Int64

func HandleWS(hub *Hub, cfg config.WSConfig) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
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
			log.Warnf("ws upgrade failed remote=%s err=%v", gc.Request.RemoteAddr, err)
			return
		}
		connCounter.Add(1)
		defer connCounter.Add(-1)

		client := NewClient(conn, hub, cfg)
		hub.Register(client)

		log.Infof("ws client connected client_id=%s remote=%s active=%d",
			client.ID(), client.addr, connCounter.Load())

		go client.WritePump(gc.Request.Context())
		client.ReadPump(gc.Request.Context())
	}
}

func ActiveConnections() int64 { return connCounter.Load() }
