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

// 广播读取只获得当前连接，断线/重连/顶号不保留旧引用，热读取不能逐接收者分配会话。
func TestBroadcastConnectionLifecycle(t *testing.T) {
	service := NewService(log.New(io.Discard, "", 0), time.Second, 3*time.Second)
	if service.ConnectionByPlayerID(1) != nil {
		t.Fatal("unknown player has connection")
	}
	first := &sessionConnStub{id: "first"}
	current, err := service.Bind(1, first)
	if err != nil {
		t.Fatal(err)
	}
	if service.ConnectionByPlayerID(1) != first {
		t.Fatal("wrong bound connection")
	}
	if count := testing.AllocsPerRun(1000, func() { _ = service.ConnectionByPlayerID(1) }); count != 0 {
		t.Fatalf("broadcast lookup allocates: %v", count)
	}
	service.Disconnect(first.ID())
	if service.ConnectionByPlayerID(1) != nil {
		t.Fatal("disconnected connection retained")
	}
	second := &sessionConnStub{id: "second"}
	if _, err := service.Reconnect(current.ReconnectToken, second); err != nil {
		t.Fatal(err)
	}
	if service.ConnectionByPlayerID(1) != second {
		t.Fatal("wrong reconnect connection")
	}
	third := &sessionConnStub{id: "third"}
	if _, err := service.Bind(1, third); err != nil {
		t.Fatal(err)
	}
	service.Disconnect(second.ID())
	if service.ConnectionByPlayerID(1) != third {
		t.Fatal("late disconnect removed new connection")
	}
}
