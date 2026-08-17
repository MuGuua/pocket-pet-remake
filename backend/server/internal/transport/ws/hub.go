package wstransport

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/idgen"
	"pocket-pet-remake/server/internal/protocol"
)

// hubConnection 是停服阶段只需要识别和关闭的最小连接边界，便于独立验证连接排空逻辑。
type hubConnection interface {
	ID() string
	Close() error
}

type Hub struct {
	logger         *log.Logger
	router         *Router
	sessionService *session.Service
	upgrader       websocket.Upgrader

	connectionMu sync.Mutex
	connections  map[string]hubConnection
	connectionWG sync.WaitGroup
	closing      bool
}

func NewHub(logger *log.Logger, router *Router, sessionService *session.Service) *Hub {
	return &Hub{
		logger:         logger,
		router:         router,
		sessionService: sessionService,
		connections:    make(map[string]hubConnection),
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
	if !h.registerConnection(conn) {
		_ = conn.Close()
		return
	}
	defer h.unregisterConnection(conn.ID())
	h.serveConn(conn)
}

// registerConnection 在进入读取循环前登记连接，使停服流程可以关闭并等待所有 WebSocket 请求处理结束。
// closing 一旦置位就不再接受新连接，避免停服排空期间产生新的移动写入。
func (h *Hub) registerConnection(conn hubConnection) bool {
	if h == nil || conn == nil {
		return false
	}
	h.connectionMu.Lock()
	defer h.connectionMu.Unlock()
	if h.closing {
		return false
	}
	h.connections[conn.ID()] = conn
	h.connectionWG.Add(1)
	return true
}

// unregisterConnection 在连接读取循环和断线生命周期回调全部结束后移除登记。
func (h *Hub) unregisterConnection(connID string) {
	if h == nil {
		return
	}
	h.connectionMu.Lock()
	delete(h.connections, connID)
	h.connectionMu.Unlock()
	h.connectionWG.Done()
}

// Shutdown 停止接收新 WebSocket，关闭所有现有连接，并等待正在处理的消息及断线回调结束。
// 应用只有在该方法返回后才停止移动持久化 worker，确保最后一个已接收移动包仍能标记 dirty 并完成断线写回。
func (h *Hub) Shutdown(ctx context.Context) error {
	if h == nil {
		return nil
	}
	h.connectionMu.Lock()
	h.closing = true
	connections := make([]hubConnection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}
	h.connectionMu.Unlock()

	for _, conn := range connections {
		_ = conn.Close()
	}
	done := make(chan struct{})
	go func() {
		h.connectionWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
			packet, decodeErr := protocol.DecodePacket(payload)
			if decodeErr == nil {
				h.logger.Printf(
					"handle websocket message conn_id=%s cmd=%s(%d) seq=%d err=%v",
					conn.ID(),
					wsCmdName(packet.Cmd),
					packet.Cmd,
					packet.Seq,
					err,
				)
			} else {
				h.logger.Printf("handle websocket message conn_id=%s err=%v", conn.ID(), err)
			}
			return
		}
	}
}
