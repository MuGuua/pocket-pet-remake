package wstransport

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/idgen"
)

type Hub struct {
	logger         *log.Logger
	router         *Router
	sessionService *session.Service
	upgrader       websocket.Upgrader
}

func NewHub(logger *log.Logger, router *Router, sessionService *session.Service) *Hub {
	return &Hub{
		logger:         logger,
		router:         router,
		sessionService: sessionService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Printf("upgrade websocket: %v", err)
		return
	}

	connID, err := idgen.RandomHex(12)
	if err != nil {
		h.logger.Printf("generate conn id: %v", err)
		_ = wsConn.Close()
		return
	}

	conn := NewConnection(connID, wsConn)
	wsConn.SetReadLimit(1 << 20)
	h.serveConn(conn)
}

func (h *Hub) serveConn(conn *Connection) {
	defer func() {
		h.sessionService.Disconnect(conn.ID())
		_ = conn.Close()
	}()

	for {
		messageType, payload, err := conn.ws.ReadMessage()
		if err != nil {
			h.logger.Printf("read websocket message conn_id=%s err=%v", conn.ID(), err)
			return
		}
		if messageType != websocket.BinaryMessage {
			h.logger.Printf("ignore non-binary websocket message conn_id=%s", conn.ID())
			continue
		}
		if err := h.router.Handle(conn, payload); err != nil {
			h.logger.Printf("handle websocket message conn_id=%s err=%v", conn.ID(), err)
			return
		}
	}
}
