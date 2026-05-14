package wstransport

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"pocket-pet-remake/server/internal/protocol"
)

type Connection struct {
	id        string
	ws        *websocket.Conn
	sendMu    sync.Mutex
	closeOnce sync.Once
}

func NewConnection(id string, wsConn *websocket.Conn) *Connection {
	return &Connection{id: id, ws: wsConn}
}

func (c *Connection) ID() string {
	return c.id
}

func (c *Connection) SendPacket(packet *protocol.Packet) error {
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		return err
	}

	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if err := c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	return c.ws.WriteMessage(websocket.BinaryMessage, encoded)
}

func (c *Connection) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		closeErr = c.ws.Close()
	})
	return closeErr
}
