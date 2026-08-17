package wstransport

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/session"
)

// hubConnectionStub 模拟一个会在关闭后结束读取循环的 WebSocket 连接。
type hubConnectionStub struct {
	id        string
	closeOnce sync.Once
	closed    chan struct{}
}

// ID 返回测试连接编号。
func (c *hubConnectionStub) ID() string {
	return c.id
}

// Close 通知模拟读取循环退出。
func (c *hubConnectionStub) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

// TestHubShutdownClosesAndWaitsForActiveConnections 验证停服会关闭活动连接，
// 并等待读取循环完成注销后再允许后续移动 worker 排空。
func TestHubShutdownClosesAndWaitsForActiveConnections(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	hub := NewHub(logger, nil, sessionService)
	conn := &hubConnectionStub{id: "shutdown-conn", closed: make(chan struct{})}
	if !hub.registerConnection(conn) {
		t.Fatal("registerConnection() = false, want true")
	}
	go func() {
		<-conn.closed
		hub.unregisterConnection(conn.ID())
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := hub.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	hub.connectionMu.Lock()
	connectionCount := len(hub.connections)
	closing := hub.closing
	hub.connectionMu.Unlock()
	if !closing || connectionCount != 0 {
		t.Fatalf("hub closing=%t connections=%d, want true/0", closing, connectionCount)
	}
	select {
	case <-conn.closed:
	default:
		t.Fatal("active connection was not closed")
	}
	if hub.registerConnection(&hubConnectionStub{id: "late-conn", closed: make(chan struct{})}) {
		t.Fatal("registerConnection() = true after shutdown, want false")
	}
}
