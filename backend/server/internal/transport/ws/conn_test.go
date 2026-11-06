package wstransport

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/protocol"
)

// 可阻塞首帧写入及底层关闭，检测网络慢、TLS 关闭慢时广播是否仍立即返回。
type queueTestWriter struct {
	blockWrite <-chan struct{}
	blockClose <-chan struct{}
	entered    chan struct{}
	closed     chan struct{}
	notified   chan struct{}
	once       sync.Once
	mu         sync.Mutex
	frames     [][]byte
	calls      atomic.Int32
	active     atomic.Int32
	peak       atomic.Int32
	closes     atomic.Int32
	writeErr   error
}

func newQueueTestWriter() *queueTestWriter {
	return &queueTestWriter{entered: make(chan struct{}, 1), closed: make(chan struct{}), notified: make(chan struct{}, outboundQueueCapacity*4)}
}
func (w *queueTestWriter) SetWriteDeadline(time.Time) error { return nil }
func (w *queueTestWriter) WriteMessage(_ int, data []byte) error {
	active := w.active.Add(1)
	defer w.active.Add(-1)
	for previous := w.peak.Load(); active > previous; previous = w.peak.Load() {
		if w.peak.CompareAndSwap(previous, active) {
			break
		}
	}
	if w.calls.Add(1) == 1 {
		w.entered <- struct{}{}
		if w.blockWrite != nil {
			select {
			case <-w.blockWrite:
			case <-w.closed:
				return net.ErrClosed
			}
		}
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	select {
	case <-w.closed:
		return net.ErrClosed
	default:
	}
	w.mu.Lock()
	w.frames = append(w.frames, append([]byte(nil), data...))
	w.mu.Unlock()
	w.notified <- struct{}{}
	return nil
}
func (w *queueTestWriter) Close() error {
	w.once.Do(func() {
		w.closes.Add(1)
		if w.blockClose != nil {
			<-w.blockClose
		}
		close(w.closed)
	})
	return nil
}
func (w *queueTestWriter) snapshot() [][]byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([][]byte(nil), w.frames...)
}

// 只用短截止等待线程事件，不用长时间 sleep 推测发送是否完成。
func waitQueueCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for !condition() {
		select {
		case <-deadline.C:
			t.Fatal("queue condition timed out")
		case <-tick.C:
		}
	}
}
func waitQueueClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("connection worker did not stop")
	}
}
func queueTestPacket(seq uint32) *protocol.Packet {
	return protocol.NewPacket(protocol.CmdEntityMovePush, seq, 0, []byte(`{}`))
}
func queueTestEncoded(t *testing.T, seq uint32) []byte {
	t.Helper()
	encoded, err := protocol.EncodePacket(queueTestPacket(seq))
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
func cleanupQueue(t *testing.T, c *Connection) {
	t.Helper()
	t.Cleanup(func() { _ = c.Close(); waitQueueClosed(t, c.writerDone) })
}

// 广播与业务回包同一 FIFO，业务成功必须等到实际写完；并发业务也只能单协程写入。
func TestConnectionQueueFIFOAndSynchronousReceipt(t *testing.T) {
	block := make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(block) })
	writer := newQueueTestWriter()
	writer.blockWrite = block
	conn := newOutboundConnection("fifo", writer)
	cleanupQueue(t, conn)
	if !conn.TryQueueEncodedPacket(queueTestEncoded(t, 1)) {
		t.Fatal("first enqueue failed")
	}
	waitQueueClosed(t, writer.entered)
	if !conn.TryQueueEncodedPacket(queueTestEncoded(t, 2)) {
		t.Fatal("second enqueue failed")
	}
	result := make(chan error, 1)
	go func() { result <- conn.SendPacket(queueTestPacket(3)) }()
	waitQueueCondition(t, func() bool { return len(conn.outbound) == 2 })
	select {
	case err := <-result:
		t.Fatalf("business returned before actual write: %v", err)
	default:
	}
	release.Do(func() { close(block) })
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receipt missing")
	}
	frames := writer.snapshot()
	if len(frames) != 3 {
		t.Fatalf("frames=%d", len(frames))
	}
	for i, frame := range frames {
		packet, err := protocol.DecodePacket(frame)
		if err != nil || packet.Seq != uint32(i+1) {
			t.Fatal("FIFO order changed")
		}
	}
	var group sync.WaitGroup
	for i := uint32(0); i < 12; i++ {
		group.Add(1)
		go func(seq uint32) {
			defer group.Done()
			if err := conn.SendPacket(queueTestPacket(seq)); err != nil {
				t.Error(err)
			}
		}(i + 10)
	}
	group.Wait()
	if writer.peak.Load() != 1 {
		t.Fatal("concurrent websocket writes")
	}
}

// 队列满立即拒绝并发起关闭；底层关闭也很慢时，广播调用不能等待它。
func TestConnectionQueueBackpressureDoesNotWaitForClose(t *testing.T) {
	writeBlock := make(chan struct{})
	closeBlock := make(chan struct{})
	defer close(closeBlock)
	writer := newQueueTestWriter()
	writer.blockWrite = writeBlock
	writer.blockClose = closeBlock
	conn := newOutboundConnection("slow-close", writer)
	cleanupQueue(t, conn)
	conn.TryQueueEncodedPacket(queueTestEncoded(t, 1))
	waitQueueClosed(t, writer.entered)
	for i := 0; i < outboundQueueCapacity; i++ {
		if !conn.TryQueueEncodedPacket(queueTestEncoded(t, uint32(i+2))) {
			t.Fatal("premature overflow")
		}
	}
	returned := make(chan bool, 1)
	go func() { returned <- conn.TryQueueEncodedPacket(queueTestEncoded(t, 99)) }()
	select {
	case accepted := <-returned:
		if accepted {
			t.Fatal("overflow accepted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast waited for slow close")
	}
	waitQueueClosed(t, conn.closed)
	if len(conn.outbound) > outboundQueueCapacity {
		t.Fatal("unbounded queue")
	}
	if conn.TryQueueEncodedPacket(queueTestEncoded(t, 100)) {
		t.Fatal("closed connection accepted frame")
	}
	if err := conn.SendPacket(queueTestPacket(101)); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("closed send: %v", err)
	}
}

// 关闭能够唤醒已入队等待的业务请求，写失败保留原错误且结束连接。
func TestConnectionQueueCloseAndWriteFailure(t *testing.T) {
	writer := newQueueTestWriter()
	writer.blockWrite = make(chan struct{})
	conn := newOutboundConnection("cancel", writer)
	cleanupQueue(t, conn)
	conn.TryQueueEncodedPacket(queueTestEncoded(t, 1))
	waitQueueClosed(t, writer.entered)
	result := make(chan error, 1)
	go func() { result <- conn.SendPacket(queueTestPacket(2)) }()
	waitQueueCondition(t, func() bool { return len(conn.outbound) == 1 })
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("close result: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("business wait leaked")
	}
	waitQueueClosed(t, conn.writerDone)
	failure := errors.New("forced write failure")
	broken := newQueueTestWriter()
	broken.writeErr = failure
	failed := newOutboundConnection("failed", broken)
	cleanupQueue(t, failed)
	if err := failed.SendPacket(queueTestPacket(3)); !errors.Is(err, failure) {
		t.Fatalf("write error lost: %v", err)
	}
	waitQueueClosed(t, failed.writerDone)
	if broken.closes.Load() != 1 {
		t.Fatal("socket closed more than once")
	}
}

// 慢接收者排满后，健康接收者仍收到广播；其他地图与被排除的发送者不接收。
func TestSceneBroadcastIsolatesSlowConnection(t *testing.T) {
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, 30*time.Second)
	slowWriter := newQueueTestWriter()
	slowWriter.blockWrite = make(chan struct{})
	slow := newOutboundConnection("slow", slowWriter)
	cleanupQueue(t, slow)
	healthyWriter := newQueueTestWriter()
	healthy := newOutboundConnection("healthy", healthyWriter)
	cleanupQueue(t, healthy)
	otherWriter := newQueueTestWriter()
	other := newOutboundConnection("other", otherWriter)
	cleanupQueue(t, other)
	selfWriter := newQueueTestWriter()
	self := newOutboundConnection("self", selfWriter)
	cleanupQueue(t, self)
	for id, conn := range map[uint64]*Connection{1: slow, 2: healthy, 3: other, 4: self} {
		if _, err := service.Bind(id, conn); err != nil {
			t.Fatal(err)
		}
	}
	slow.TryQueueEncodedPacket(queueTestEncoded(t, 1))
	waitQueueClosed(t, slowWriter.entered)
	for i := 0; i < outboundQueueCapacity; i++ {
		slow.TryQueueEncodedPacket(queueTestEncoded(t, uint32(i+2)))
	}
	handler := &WorldHandler{sessionService: service}
	handler.setPlayerScenePresence(1, 7)
	handler.setPlayerScenePresence(2, 7)
	handler.setPlayerScenePresence(3, 8)
	handler.setPlayerScenePresence(4, 7)
	done := make(chan struct{})
	go func() { handler.sendPacketToScene(7, 4, queueTestPacket(200)); close(done) }()
	waitQueueClosed(t, done)
	waitQueueCondition(t, func() bool { return len(healthyWriter.snapshot()) == 1 })
	decoded, err := protocol.DecodePacket(healthyWriter.snapshot()[0])
	if err != nil || decoded.Seq != 200 {
		t.Fatal("healthy broadcast missing")
	}
	waitQueueClosed(t, slow.closed)
	if len(otherWriter.snapshot()) != 0 || len(selfWriter.snapshot()) != 0 {
		t.Fatal("scene/exclusion boundary changed")
	}
}

// 实际 Gorilla WebSocket 验证入队与同步回包同序、编码正确、连接关闭后写协程退出。
func TestConnectionQueueRealWebSocket(t *testing.T) {
	accepted := make(chan *Connection, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn := NewConnection("wire", ws)
		accepted <- conn
		<-conn.closed
		<-conn.writerDone
	}))
	defer server.Close()
	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var conn *Connection
	select {
	case conn = <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("upgrade missing")
	}
	cleanupQueue(t, conn)
	if !conn.TryQueueEncodedPacket(queueTestEncoded(t, 1)) {
		t.Fatal("wire enqueue failed")
	}
	if err := conn.SendPacket(queueTestPacket(2)); err != nil {
		t.Fatal(err)
	}
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	for expected := uint32(1); expected <= 2; expected++ {
		kind, raw, err := client.ReadMessage()
		if err != nil || kind != websocket.BinaryMessage {
			t.Fatalf("wire read: %v", err)
		}
		packet, err := protocol.DecodePacket(raw)
		if err != nil || packet.Seq != expected {
			t.Fatal("wire packet order/checksum changed")
		}
	}
}

// 关闭与多个广播生产者同时运行时，发送只返回失败，不 panic 或重复关闭。
func TestConnectionQueueConcurrentClose(t *testing.T) {
	writer := newQueueTestWriter()
	conn := newOutboundConnection("concurrent", writer)
	cleanupQueue(t, conn)
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 10; j++ {
				conn.TryQueueEncodedPacket([]byte{1, 2, 3})
			}
			_ = conn.Close()
		}()
	}
	group.Wait()
	waitQueueClosed(t, conn.writerDone)
	if writer.closes.Load() != 1 {
		t.Fatal("close was not idempotent")
	}
}

// 通过真实 Hub 建连和收发后停服，注销必须同时等待读循环与写协程退出。
func TestHubShutdownWaitsForQueuedConnectionWriter(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessions := session.NewService(logger, time.Second, 30*time.Second)
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, sessions)
	hub := NewHub(logger, router, sessions)
	server := httptest.NewServer(hub)
	defer server.Close()
	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Shutdown(ctx)
	})
	raw, err := protocol.EncodePacket(protocol.NewPacket(protocol.CmdHeartbeatReq, 21, 0, []byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if err = client.WriteMessage(websocket.BinaryMessage, raw); err != nil {
		t.Fatal(err)
	}
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, reply, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	packet, err := protocol.DecodePacket(reply)
	if err != nil || packet.Cmd != protocol.CmdErrorPush || packet.Seq != 21 {
		t.Fatal("unauthorized response wire contract changed")
	}
	hub.connectionMu.Lock()
	var conn *Connection
	for _, candidate := range hub.connections {
		conn = candidate.(*Connection)
	}
	hub.connectionMu.Unlock()
	if conn == nil {
		t.Fatal("connection not registered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = hub.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-conn.writerDone:
	default:
		t.Fatal("writer still active after shutdown")
	}
	hub.connectionMu.Lock()
	count := len(hub.connections)
	hub.connectionMu.Unlock()
	if count != 0 {
		t.Fatal("connection registry not drained")
	}
}
