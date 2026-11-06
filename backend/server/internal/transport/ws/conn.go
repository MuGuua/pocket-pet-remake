package wstransport

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"pocket-pet-remake/server/internal/platform/telemetry"
	"pocket-pet-remake/server/internal/protocol"
)

// 100 人同屏一轮不可合并状态已超过 64 帧；256 帧为两轮突发及业务边界留出有界余量。
// 仍由满队列关闭和 5 秒写超时隔离持续慢连接，不无限缓存，也不静默丢弃业务或起停事件。
const outboundQueueCapacity = 256

// 写连接边界与 Gorilla 一致，便于确定性验证慢写、关闭和写失败。
type outboundWriter interface {
	SetWriteDeadline(time.Time) error
	WriteMessage(int, []byte) error
	Close() error
}

// 同一 FIFO 中的不可变编码帧；业务发送附带回执，广播不等待网络写入。
type outboundFrame struct {
	encoded  []byte
	result   chan error
	movement *pendingMovement
	queuedAt time.Time
}

// 尚未发送的移动样本；首帧固定，后续直线样本仅替换最新帧，不修改共享编码字节。
// 所有字段与待合并索引均受连接的 enqueueMu 保护，网络写入使用取出的不可变帧。
type pendingMovement struct {
	encoded     []byte
	state       protocol.EntityMovePush
	replaceable bool
	dx, dy      int64
}

type Connection struct {
	id     string
	ws     *websocket.Conn
	writer outboundWriter
	// 单写协程独占网络写入，保留每连接队列中的帧顺序。
	outbound         chan outboundFrame
	enqueueMu        sync.Mutex
	pendingMovements map[uint64]*pendingMovement
	queueSpace       chan struct{}
	closed           chan struct{}
	writerDone       chan struct{}
	ioClosed         chan struct{}
	closeErr         error
	closeOnce        sync.Once
}

func NewConnection(id string, wsConn *websocket.Conn) *Connection {
	conn := newOutboundConnection(id, wsConn)
	conn.ws = wsConn
	return conn
}

// 初始化有界队列；只有生产构造器再绑定读取使用的真实 WebSocket。
func newOutboundConnection(id string, writer outboundWriter) *Connection {
	conn := &Connection{id: id, writer: writer, pendingMovements: make(map[uint64]*pendingMovement), queueSpace: make(chan struct{}, 1), outbound: make(chan outboundFrame, outboundQueueCapacity), closed: make(chan struct{}), writerDone: make(chan struct{}), ioClosed: make(chan struct{})}
	go conn.writeLoop()
	return conn
}

func (c *Connection) ID() string { return c.id }

// 业务回包保留同步语义：成功代表实际写入完成，关闭/失败会唤醒等待者。
func (c *Connection) SendPacket(packet *protocol.Packet) error {
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		return err
	}
	result := make(chan error, 1)
	for !c.tryQueueOrderedFrame(outboundFrame{encoded: encoded, result: result}) {
		select {
		case <-c.closed:
			return net.ErrClosed
		case <-c.queueSpace:
		}
	}
	select {
	case err := <-result:
		return err
	case <-c.closed:
		// 写失败时优先保留写协程已经投递的原始错误。
		select {
		case err := <-result:
			return err
		default:
			return net.ErrClosed
		}
	}
}

// 广播帧只入队；调用方必须保持 encoded 不可变，才能安全供同场景连接共享。
// 队列已满时关闭该慢连接，避免静默丢弃进入/离开通知或阻塞其他玩家。
func (c *Connection) TryQueueEncodedPacket(encoded []byte) bool {
	if c.tryQueueOrderedFrame(outboundFrame{encoded: encoded}) {
		return true
	}
	c.startClose("outbound_queue_full")
	return false
}

// 队列入队与移动边界更新在同一个短锁中完成；不能持锁等待空间或网络写入。
func (c *Connection) tryQueueOrderedFrame(frame outboundFrame) bool {
	c.enqueueMu.Lock()
	defer c.enqueueMu.Unlock()
	select {
	case <-c.closed:
		return false
	default:
	}
	frame.queuedAt = time.Now()
	select {
	case c.outbound <- frame:
		// 进入/离开、停步、传送和业务回包都是屏障，禁止把后续位置改写到它们之前。
		clear(c.pendingMovements)
		c.notifyQueueSpace()
		return true
	default:
		return false
	}
}

// 用容量一的通知唤醒等待中的业务发送者；有剩余空间时传递通知，避免多个等待者停滞。
func (c *Connection) notifyQueueSpace() {
	telemetry.Default.Record("ws.queue_depth_sample", "frames", int64(len(c.outbound)), false)
	if len(c.outbound) < cap(c.outbound) {
		select {
		case c.queueSpace <- struct{}{}:
		default:
		}
	}
}

// 只合并正常同代次行走状态，保留首帧、方向/速度变化、弯折及所有非移动消息的顺序。
// encoded 必须与 state 对应且保持不可变；世界广播使用同一个服务端载荷同时构造两者。
func (c *Connection) TryQueueMovementPacket(encoded []byte, state protocol.EntityMovePush) bool {
	if !state.Moving || state.SceneID == 0 || state.SceneVersion == 0 || state.EntityID == 0 || state.MoveSeq == 0 || state.ClientTick <= 0 {
		return c.TryQueueEncodedPacket(encoded)
	}
	c.enqueueMu.Lock()
	select {
	case <-c.closed:
		c.enqueueMu.Unlock()
		return false
	default:
	}
	previous := c.pendingMovements[state.EntityID]
	dx, dy, continued := movementContinues(previous, state)
	if continued && previous.replaceable {
		previous.encoded = encoded
		previous.state = state
		c.enqueueMu.Unlock()
		return true
	}
	pending := &pendingMovement{encoded: encoded, state: state, replaceable: continued, dx: dx, dy: dy}
	select {
	case c.outbound <- outboundFrame{movement: pending, queuedAt: time.Now()}:
		c.pendingMovements[state.EntityID] = pending
		c.notifyQueueSpace()
		c.enqueueMu.Unlock()
		return true
	default:
		c.enqueueMu.Unlock()
		c.startClose("outbound_queue_full")
		return false
	}
}

// 精细坐标必须沿同一直线向前；即使 facing 没变，几何拐弯也不能被合并。
// 差值限制保证叉积/点积的 int64 计算不溢出；异常大位移保留独立样本。
func movementContinues(previous *pendingMovement, next protocol.EntityMovePush) (int64, int64, bool) {
	if previous == nil {
		return 0, 0, false
	}
	old := previous.state
	if old.EntityID != next.EntityID || !next.Moving || old.ClientTick <= 0 || old.SceneID != next.SceneID || old.SceneVersion != next.SceneVersion || old.Facing != next.Facing || old.Speed != next.Speed || !old.Moving || next.MoveSeq <= old.MoveSeq || next.ClientTick <= old.ClientTick {
		return 0, 0, false
	}
	dx := int64(next.PrecisePos.X) - int64(old.PrecisePos.X)
	dy := int64(next.PrecisePos.Y) - int64(old.PrecisePos.Y)
	const limit int64 = 1<<31 - 1
	if dx > limit || dx < -limit || dy > limit || dy < -limit || (dx == 0 && dy == 0) {
		return 0, 0, false
	}
	if previous.dx != 0 || previous.dy != 0 {
		if previous.dx*dy != previous.dy*dx || previous.dx*dx+previous.dy*dy <= 0 {
			return 0, 0, false
		}
		return previous.dx, previous.dy, true
	}
	return dx, dy, true
}

// 保留原 5 秒写超时；关闭底层连接会同时中断正在进行的网络读写。
func (c *Connection) writeLoop() {
	defer func() {
		c.enqueueMu.Lock()
		clear(c.pendingMovements)
		c.enqueueMu.Unlock()
		close(c.writerDone)
	}()
	for {
		select {
		case <-c.closed:
			return
		case frame := <-c.outbound:
			select {
			case <-c.closed:
				return
			default:
			}
			c.enqueueMu.Lock()
			encoded := frame.encoded
			if frame.movement != nil {
				pending := frame.movement
				encoded = pending.encoded
				pending.encoded = nil
				if c.pendingMovements[pending.state.EntityID] == pending {
					delete(c.pendingMovements, pending.state.EntityID)
				}
			}
			c.notifyQueueSpace()
			c.enqueueMu.Unlock()
			telemetry.Duration("ws.queue_wait", time.Since(frame.queuedAt), false)
			writeStarted := time.Now()
			err := c.writer.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err == nil {
				err = c.writer.WriteMessage(websocket.BinaryMessage, encoded)
			}
			telemetry.Duration("ws.write", time.Since(writeStarted), err != nil)
			if frame.result != nil {
				frame.result <- err
			}
			if err != nil {
				_ = c.Close()
				return
			}
		}
	}
}

// 显式关闭仍等待底层关闭完成；广播背压只触发异步关闭，不等待 TLS/网络关闭。
func (c *Connection) Close() error {
	c.startClose("")
	<-c.ioClosed
	return c.closeErr
}

// 不关闭队列 channel，防止并发广播触发 send-on-closed；每连接至多一个关闭协程。
func (c *Connection) startClose(reason string) {
	c.closeOnce.Do(func() {
		close(c.closed)
		if reason != "" {
			telemetry.Default.Record("ws.backpressure_close", "events", 1, true)
		}
		go func() {
			c.closeErr = c.writer.Close()
			if reason != "" {
				log.Printf("[ws-backpressure] conn_id=%s reason=%s queue_capacity=%d", c.id, reason, outboundQueueCapacity)
			}
			close(c.ioClosed)
		}()
	})
}
