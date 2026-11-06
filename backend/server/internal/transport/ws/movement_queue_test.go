package wstransport

import (
	"bytes"
	"io"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
)

// 构造带源序号/时间与精细坐标的移动状态，全部数据只用于测试。
func movementSample(seq uint32, entity uint64, x, y int32) protocol.EntityMovePush {
	return protocol.EntityMovePush{SceneID: 7, SceneVersion: 1, EntityID: entity, MoveSeq: seq, ClientTick: int64(seq) * 100, ServerTick: int64(seq)*100 + 123, PrecisePos: protocol.Vec2i{X: x, Y: y}, Facing: protocol.Vec2i{X: 1}, Moving: true, Speed: 1000, ToPos: protocol.Vec2i{X: x / 1000, Y: y / 1000}}
}
func encodedMovement(t *testing.T, state protocol.EntityMovePush) []byte {
	t.Helper()
	packet, err := protocol.NewJSONPacket(protocol.CmdEntityMovePush, 0, 0, state)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
func enqueueMovement(t *testing.T, c *Connection, state protocol.EntityMovePush) {
	t.Helper()
	if !c.TryQueueMovementPacket(encodedMovement(t, state), state) {
		t.Fatal("movement unexpectedly rejected")
	}
}
func readMovement(t *testing.T, encoded []byte) protocol.EntityMovePush {
	t.Helper()
	packet, err := protocol.DecodePacket(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var state protocol.EntityMovePush
	if err = protocol.UnmarshalBody(packet.Body, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

// 阻塞第一帧，确保后续坐标仍在队列中；释放是幂等的，失败路径也不遗留写协程。
func blockedMovementConnection(t *testing.T) (*Connection, *queueTestWriter, func()) {
	t.Helper()
	block := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(block) }) }
	writer := newQueueTestWriter()
	writer.blockWrite = block
	c := newOutboundConnection("move-test", writer)
	cleanupQueue(t, c)
	t.Cleanup(release)
	if !c.TryQueueEncodedPacket(queueTestEncoded(t, 999)) {
		t.Fatal("block frame missing")
	}
	waitQueueClosed(t, writer.entered)
	return c, writer, release
}

// 一百个积压的直线样本保留首帧/最新帧及停步，不覆盖原时间、序号与坐标。
func TestMovementQueueRetainsAnchorLatestAndStop(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	for seq := uint32(1); seq <= 100; seq++ {
		enqueueMovement(t, c, movementSample(seq, 44, int32(seq)*100, 0))
	}
	if len(c.outbound) != 2 {
		t.Fatalf("queued=%d, want anchor+latest", len(c.outbound))
	}
	stop := movementSample(101, 44, 10000, 0)
	stop.Moving = false
	enqueueMovement(t, c, stop)
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 4 })
	frames := writer.snapshot()
	var seqs []uint32
	for _, frame := range frames[1:] {
		seqs = append(seqs, readMovement(t, frame).MoveSeq)
	}
	if !reflect.DeepEqual(seqs, []uint32{1, 100, 101}) {
		t.Fatalf("movement order=%v", seqs)
	}
	latest := readMovement(t, frames[2])
	if latest.ClientTick != 10000 || latest.ServerTick != 10123 || latest.PrecisePos.X != 10000 {
		t.Fatal("latest payload rewritten")
	}
	if readMovement(t, frames[3]).Moving {
		t.Fatal("stop lost")
	}
}

// 同 facing 的几何弯折、实际转向和重新起步保留独立锚点，避免插值跨墙抄近路。
func TestMovementQueuePreservesCornersAndTransitions(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	samples := []protocol.EntityMovePush{
		movementSample(1, 44, 0, 0), movementSample(2, 44, 100, 0), movementSample(3, 44, 200, 0),
		movementSample(4, 44, 200, 100), movementSample(5, 44, 200, 200), movementSample(6, 44, 200, 300),
		movementSample(7, 44, 200, 400), movementSample(8, 44, 200, 500), movementSample(9, 44, 200, 600),
		movementSample(10, 44, 200, 600), movementSample(11, 44, 200, 700), movementSample(12, 44, 200, 800), movementSample(13, 44, 200, 900),
	}
	for i := 6; i < len(samples); i++ {
		samples[i].Facing = protocol.Vec2i{Y: 1}
	}
	samples[9].Moving = false
	for _, state := range samples {
		enqueueMovement(t, c, state)
	}
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 10 })
	var seqs []uint32
	for _, frame := range writer.snapshot()[1:] {
		seqs = append(seqs, readMovement(t, frame).MoveSeq)
	}
	if !reflect.DeepEqual(seqs, []uint32{1, 3, 4, 6, 7, 9, 10, 11, 13}) {
		t.Fatalf("turn/start/stop order=%v", seqs)
	}
}

// 不同角色可独立合并，但业务回包成功入队后不能再修改其之前的样本。
func TestMovementQueueBusinessBarrierAndIndependentEntities(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	enqueueMovement(t, c, movementSample(1, 44, 0, 0))
	enqueueMovement(t, c, movementSample(2, 44, 100, 0))
	enqueueMovement(t, c, movementSample(1, 55, 0, 0))
	enqueueMovement(t, c, movementSample(2, 55, 100, 0))
	enqueueMovement(t, c, movementSample(3, 44, 200, 0))
	result := make(chan error, 1)
	go func() { result <- c.SendPacket(queueTestPacket(777)) }()
	waitQueueCondition(t, func() bool { return len(c.outbound) == 5 })
	enqueueMovement(t, c, movementSample(4, 44, 300, 0))
	enqueueMovement(t, c, movementSample(5, 44, 400, 0))
	enqueueMovement(t, c, movementSample(6, 44, 500, 0))
	release()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("business receipt missing")
	}
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 8 })
	frames := writer.snapshot()
	if readMovement(t, frames[2]).MoveSeq != 3 || readMovement(t, frames[6]).MoveSeq != 4 || readMovement(t, frames[7]).MoveSeq != 6 {
		t.Fatal("movement crossed business barrier")
	}
	packet, err := protocol.DecodePacket(frames[5])
	if err != nil || packet.Seq != 777 {
		t.Fatal("business frame order changed")
	}
}

// 生命周期和不可信采样元数据不合并；极端坐标差值不会使叉积溢出后误判直线。
func TestMovementContinuationBoundaries(t *testing.T) {
	base := movementSample(2, 44, 100, 0)
	for _, tc := range []struct {
		name   string
		change func(*protocol.EntityMovePush)
	}{
		{"entity", func(s *protocol.EntityMovePush) { s.EntityID++ }},
		{"scene", func(s *protocol.EntityMovePush) { s.SceneID++ }},
		{"version", func(s *protocol.EntityMovePush) { s.SceneVersion++ }},
		{"turn", func(s *protocol.EntityMovePush) { s.Facing = protocol.Vec2i{Y: 1} }},
		{"speed", func(s *protocol.EntityMovePush) { s.Speed++ }},
		{"stop", func(s *protocol.EntityMovePush) { s.Moving = false }},
		{"old-seq", func(s *protocol.EntityMovePush) { s.MoveSeq = 1 }},
		{"old-clock", func(s *protocol.EntityMovePush) { s.ClientTick = 100 }},
		{"bend", func(s *protocol.EntityMovePush) { s.PrecisePos.Y = 100 }},
		{"reverse", func(s *protocol.EntityMovePush) { s.PrecisePos.X = 0 }},
		{"stationary", func(s *protocol.EntityMovePush) { s.PrecisePos.X = 100 }},
		{"extreme", func(s *protocol.EntityMovePush) { s.PrecisePos.X = -1 << 31 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := movementSample(3, 44, 200, 0)
			tc.change(&next)
			if _, _, ok := movementContinues(&pendingMovement{state: base, dx: 100}, next); ok {
				t.Fatal("unsafe samples merged")
			}
		})
	}
}

// 满队列仍能更新已经存在的可合并样本；新的独立样本则沿用断开背压策略。
func TestMovementQueueFullCanStillCoalesce(t *testing.T) {
	c, _, _ := blockedMovementConnection(t)
	for entity := uint64(1); entity <= uint64(outboundQueueCapacity/2); entity++ {
		enqueueMovement(t, c, movementSample(1, entity, 0, 0))
		enqueueMovement(t, c, movementSample(2, entity, 100, 0))
	}
	if len(c.outbound) != outboundQueueCapacity {
		t.Fatal("queue not saturated")
	}
	enqueueMovement(t, c, movementSample(3, 1, 200, 0))
	select {
	case <-c.closed:
		t.Fatal("coalescible update closed full queue")
	default:
	}
	if c.TryQueueMovementPacket(encodedMovement(t, movementSample(1, uint64(outboundQueueCapacity+1), 0, 0)), movementSample(1, uint64(outboundQueueCapacity+1), 0, 0)) {
		t.Fatal("unbounded new movement accepted")
	}
	waitQueueClosed(t, c.closed)
}

// 共享的原始编码不改写，一个连接合并最新状态不影响另一个接收者的待发内容。
func TestMovementQueueDoesNotMutateSharedFrames(t *testing.T) {
	first, w1, release1 := blockedMovementConnection(t)
	second, w2, release2 := blockedMovementConnection(t)
	for seq := uint32(1); seq <= 2; seq++ {
		state := movementSample(seq, 44, int32(seq)*100, 0)
		raw := encodedMovement(t, state)
		copy := append([]byte(nil), raw...)
		first.TryQueueMovementPacket(raw, state)
		second.TryQueueMovementPacket(raw, state)
		if !bytes.Equal(raw, copy) {
			t.Fatal("shared bytes mutated")
		}
	}
	enqueueMovement(t, first, movementSample(3, 44, 300, 0))
	release1()
	release2()
	waitQueueCondition(t, func() bool { return len(w1.snapshot()) == 3 && len(w2.snapshot()) == 3 })
	if readMovement(t, w1.snapshot()[2]).MoveSeq != 3 || readMovement(t, w2.snapshot()[2]).MoveSeq != 2 {
		t.Fatal("recipient states contaminated")
	}
}

// 世界移动广播实际接入合并能力，其他地图仍收不到任何状态。
func TestWorldMovementBroadcastUsesCoalescing(t *testing.T) {
	slow, writer, release := blockedMovementConnection(t)
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, 30*time.Second)
	if _, err := service.Bind(1, slow); err != nil {
		t.Fatal(err)
	}
	otherWriter := newQueueTestWriter()
	other := newOutboundConnection("other-map", otherWriter)
	cleanupQueue(t, other)
	if _, err := service.Bind(2, other); err != nil {
		t.Fatal(err)
	}
	handler := &WorldHandler{sessionService: service}
	handler.setPlayerScenePresence(1, 7)
	handler.setPlayerScenePresence(2, 8)
	handler.setPlayerScenePresence(44, 7)
	for seq := uint32(1); seq <= 100; seq++ {
		handler.broadcastEntityMove(7, 1, 44, seq, world.Vec2i{}, world.Vec2i{X: int32(seq) / 10}, protocol.Vec2i{X: int32(seq) * 100}, protocol.Vec2i{X: 1}, true, 1000, int64(seq)*100+123, int64(seq)*100)
	}
	if len(slow.outbound) != 2 {
		t.Fatal("world broadcast did not use coalescing")
	}
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 3 })
	if readMovement(t, writer.snapshot()[2]).MoveSeq != 100 || len(otherWriter.snapshot()) != 0 {
		t.Fatal("broadcast metadata or scene boundary changed")
	}
}

// 进入/离开通知冻结先前位置；新生命周期的样本不能覆盖旧生命周期的队列内容。
func TestMovementQueueEntityLifecycleBarriers(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	enqueueMovement(t, c, movementSample(1, 44, 0, 0))
	enqueueMovement(t, c, movementSample(2, 44, 100, 0))
	entered, err := protocol.EncodePacket(protocol.NewPacket(protocol.CmdEntityEnterPush, 701, 0, []byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if !c.TryQueueEncodedPacket(entered) {
		t.Fatal("enter notification rejected")
	}
	for seq := uint32(3); seq <= 5; seq++ {
		enqueueMovement(t, c, movementSample(seq, 44, int32(seq)*100, 0))
	}
	left, err := protocol.EncodePacket(protocol.NewPacket(protocol.CmdEntityLeavePush, 702, 0, []byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if !c.TryQueueEncodedPacket(left) {
		t.Fatal("leave notification rejected")
	}
	for seq := uint32(1); seq <= 3; seq++ {
		state := movementSample(seq, 44, int32(seq)*100, 0)
		state.SceneVersion = 2
		enqueueMovement(t, c, state)
	}
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 9 })
	frames := writer.snapshot()
	if readMovement(t, frames[2]).MoveSeq != 2 || readMovement(t, frames[4]).MoveSeq != 3 || readMovement(t, frames[5]).MoveSeq != 5 || readMovement(t, frames[7]).SceneVersion != 2 || readMovement(t, frames[8]).MoveSeq != 3 {
		t.Fatal("lifecycle crossed movement samples")
	}
	for index, seq := range map[int]uint32{3: 701, 6: 702} {
		packet, err := protocol.DecodePacket(frames[index])
		if err != nil || packet.Seq != seq {
			t.Fatal("lifecycle notification order changed")
		}
	}
}

// 已经交给网络的帧不可再被替换；后续积压样本在新的队列节点上合并。
func TestMovementQueueDoesNotReplaceInflightFrame(t *testing.T) {
	block := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(block) }) }
	writer := newQueueTestWriter()
	writer.blockWrite = block
	c := newOutboundConnection("inflight-move", writer)
	cleanupQueue(t, c)
	t.Cleanup(release)
	enqueueMovement(t, c, movementSample(1, 44, 100, 0))
	waitQueueClosed(t, writer.entered)
	for seq := uint32(2); seq <= 4; seq++ {
		enqueueMovement(t, c, movementSample(seq, 44, int32(seq)*100, 0))
	}
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 3 })
	var seqs []uint32
	for _, raw := range writer.snapshot() {
		seqs = append(seqs, readMovement(t, raw).MoveSeq)
	}
	if !reflect.DeepEqual(seqs, []uint32{1, 2, 4}) {
		t.Fatalf("inflight frame changed: %v", seqs)
	}
}

// 业务发送等待满队列的空间，写协程释放空间后继续发送，不能丢唤醒或改变 FIFO。
func TestMovementQueueSpaceWakesBusinessSender(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	for i := 0; i < outboundQueueCapacity; i++ {
		if !c.TryQueueEncodedPacket(queueTestEncoded(t, uint32(i+1))) {
			t.Fatal("queue fill failed")
		}
	}
	result := make(chan error, 1)
	go func() { result <- c.SendPacket(queueTestPacket(777)) }()
	release()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sender was not woken")
	}
	frames := writer.snapshot()
	if len(frames) != outboundQueueCapacity+2 {
		t.Fatal("business send returned before all earlier frames were written")
	}
	packet, err := protocol.DecodePacket(frames[len(frames)-1])
	if err != nil || packet.Seq != 777 {
		t.Fatal("business moved ahead of existing frames")
	}
}

// 合并、写出、可靠发送和关闭同时发生时，不出现数据竞态、重复关闭或无限等待。
func TestMovementQueueConcurrentProducersAndClose(t *testing.T) {
	writer := newQueueTestWriter()
	c := newOutboundConnection("movement-race", writer)
	cleanupQueue(t, c)
	state := movementSample(1, 44, 100, 0)
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func(entity uint64) {
			defer group.Done()
			for seq := uint32(1); seq < 20; seq++ {
				next := state
				next.EntityID = entity
				next.MoveSeq = seq
				next.ClientTick = int64(seq) * 100
				next.PrecisePos.X = int32(seq) * 100
				packet, err := protocol.NewJSONPacket(protocol.CmdEntityMovePush, 0, 0, next)
				if err != nil {
					t.Error(err)
					return
				}
				raw, err := protocol.EncodePacket(packet)
				if err != nil {
					t.Error(err)
					return
				}
				c.TryQueueMovementPacket(raw, next)
			}
		}(uint64(i + 1))
	}
	group.Add(1)
	go func() { defer group.Done(); _ = c.SendPacket(queueTestPacket(888)); _ = c.Close() }()
	group.Wait()
	waitQueueClosed(t, c.writerDone)
	if writer.closes.Load() != 1 {
		t.Fatal("connection close not idempotent")
	}
}

// 服务端传送/旧客户端缺少有效采样代次时，不合并坐标，也不复用先前的移动索引。
func TestMovementQueueServerAndLegacySamplesStayOrdered(t *testing.T) {
	c, writer, release := blockedMovementConnection(t)
	for seq := uint32(1); seq <= 3; seq++ {
		enqueueMovement(t, c, movementSample(seq, 44, int32(seq)*100, 0))
	}
	for seq := uint32(4); seq <= 8; seq++ {
		state := movementSample(seq, 44, int32(seq)*100, 0)
		if seq < 6 {
			state.ClientTick = 0
		} else {
			state.SceneVersion = 0
		}
		enqueueMovement(t, c, state)
	}
	release()
	waitQueueCondition(t, func() bool { return len(writer.snapshot()) == 8 })
	var seqs []uint32
	for _, frame := range writer.snapshot()[1:] {
		seqs = append(seqs, readMovement(t, frame).MoveSeq)
	}
	if !reflect.DeepEqual(seqs, []uint32{1, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("server/legacy samples merged: %v", seqs)
	}
}
