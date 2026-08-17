package session

import (
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/protocol"
)

// sessionConnStub 提供会话生命周期测试所需的最小连接行为。
type sessionConnStub struct {
	id     string
	closed bool
}

// ID 返回测试连接的稳定编号。
func (c *sessionConnStub) ID() string { return c.id }

// SendPacket 接受服务端推送；生命周期测试只关心连接关闭与回调顺序。
func (c *sessionConnStub) SendPacket(_ *protocol.Packet) error { return nil }

// Close 记录旧连接已经被顶号流程关闭。
func (c *sessionConnStub) Close() error {
	c.closed = true
	return nil
}

// TestServiceBindInvokesReplacementHandler 验证同一玩家新登录会同步触发顶号回调，而普通重连不会重复触发。
func TestServiceBindInvokesReplacementHandler(t *testing.T) {
	service := NewService(log.New(io.Discard, "", 0), time.Second, 3*time.Second)
	oldConn := &sessionConnStub{id: "old-conn"}
	if _, err := service.Bind(10001, oldConn); err != nil {
		t.Fatalf("Bind(old) error = %v", err)
	}
	var replacedPlayerIDs []uint64
	service.SetReplacementHandler(func(playerID uint64) {
		replacedPlayerIDs = append(replacedPlayerIDs, playerID)
	})

	newConn := &sessionConnStub{id: "new-conn"}
	newSession, err := service.Bind(10001, newConn)
	if err != nil {
		t.Fatalf("Bind(new) error = %v", err)
	}
	if len(replacedPlayerIDs) != 1 || replacedPlayerIDs[0] != 10001 {
		t.Fatalf("replacement callbacks = %v, want [10001]", replacedPlayerIDs)
	}
	if !oldConn.closed {
		t.Fatal("old connection closed = false, want true")
	}

	reconnectConn := &sessionConnStub{id: "reconnect-conn"}
	if _, err := service.Reconnect(newSession.ReconnectToken, reconnectConn); err != nil {
		t.Fatalf("Reconnect() error = %v", err)
	}
	if len(replacedPlayerIDs) != 1 {
		t.Fatalf("replacement callbacks after reconnect = %v, want unchanged", replacedPlayerIDs)
	}
}
