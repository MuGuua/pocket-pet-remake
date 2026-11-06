package wstransport

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
)

// 显式启用本机真实网络压测，避免日常单测被大量 socket 与计时扰动拖慢。
// 只运行现有广播/编码/队列/网络/解码路径，不连接数据库或 Redis，也不代表移动真机延迟。
func TestSceneBroadcastLoad(t *testing.T) {
	if os.Getenv("POCKET_WS_SCENE_LOAD_TEST") != "1" {
		t.Skip("set POCKET_WS_SCENE_LOAD_TEST=1 for local WebSocket broadcast load regression")
	}
	for _, players := range []int{20, 50, 100} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("players_%d/blocked_%t", players, blocked), func(t *testing.T) {
				runSceneBroadcastLoad(t, players, blocked)
			})
		}
	}
}

// 同样人数分布在相隔足够远的十人邻域，验证实际网格筛选和健康/阻塞接收者隔离。
func TestSceneAOIBroadcastLoad(t *testing.T) {
	if os.Getenv("POCKET_WS_SCENE_LOAD_TEST") != "1" {
		t.Skip("set POCKET_WS_SCENE_LOAD_TEST=1")
	}
	for _, players := range []int{20, 50, 100} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("players_%d/blocked_%t", players, blocked), func(t *testing.T) { runSceneBroadcastLoad(t, players, blocked, 10) })
		}
	}
}

// 每个观察端独占结果切片，汇总阶段才合并，避免测量锁竞争成为广播瓶颈。
type sceneLoadResult struct {
	latencies []int64
	bytes     int64
	err       error
}

// 按客户端当前 100ms 采样周期错峰上报，所有观察端使用真实 Gorilla 连接接收。
// 每帧改变朝向以压测不可合并边界；最终停步必须送达，避免合并掩盖消息丢失。
func runSceneBroadcastLoad(t *testing.T, players int, blocked bool, groupSizes ...int) {
	t.Helper()
	const samples = 20
	const interval = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	handler := NewWorldHandler(service, nil, nil, nil, nil, nil, nil, nil)
	groupSize := 0
	if len(groupSizes) > 0 {
		groupSize = groupSizes[0]
		handler.aoi = newWorldAOI(world.AOIConfig{Enabled: true, CellSizeMilli: 1000, Radius: 1}, handler)
	}

	accepted := make(chan *Connection, players)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		accepted <- NewConnection(r.URL.Query().Get("id"), ws)
	}))
	clients := make([]*websocket.Conn, 0, players)
	connections := make([]*Connection, 0, players+1)
	// 即使中途失败，也先中断所有 IO，再等待协程退出，避免压测遗留连接。
	t.Cleanup(func() {
		cancel()
		for _, client := range clients {
			_ = client.Close()
		}
		for _, conn := range connections {
			_ = conn.Close()
		}
		for _, conn := range connections {
			waitQueueClosed(t, conn.writerDone)
		}
		server.Close()
	})
	for index := 0; index < players; index++ {
		id := uint64(index + 1)
		client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+fmt.Sprintf("?id=%d", id), nil)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
		var conn *Connection
		select {
		case conn = <-accepted:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		connections = append(connections, conn)
		if _, err := service.Bind(id, conn); err != nil {
			t.Fatal(err)
		}
		handler.setPlayerScenePresence(id, 7)
		if groupSize > 0 {
			handler.aoi.join(id, 7, protocol.EntityBrief{EntityID: id, PlayerID: id, EntityType: 1, Pos: protocol.Vec2i{X: int32(index/groupSize) * 100}}, nil, false, nil)
		}

	}
	var slow *Connection
	if blocked {
		writer := newQueueTestWriter()
		writer.blockWrite = make(chan struct{})
		slow = newOutboundConnection("load-blocked-peer", writer)
		connections = append(connections, slow)
		// 慢连接已有半个队列积压，再注入本轮广播，确保小邻域场景也会触发容量背压。
		slow.TryQueueEncodedPacket(queueTestEncoded(t, 1))
		waitQueueClosed(t, writer.entered)
		for index := 0; index < outboundQueueCapacity/2; index++ {
			slow.TryQueueEncodedPacket(queueTestEncoded(t, uint32(index+2)))
		}

		id := uint64(players + 1)
		if _, err := service.Bind(id, slow); err != nil {
			t.Fatal(err)
		}
		handler.setPlayerScenePresence(id, 7)
		if groupSize > 0 {
			handler.aoi.join(id, 7, protocol.EntityBrief{EntityID: id, PlayerID: id, EntityType: 1}, nil, false, nil)
		}
	}
	// 本进程单调时间测量广播调用前到接收解码后的耗时，不做跨端绝对时钟相减。
	epoch := time.Now()
	stamps := make([][]atomic.Int64, players)
	for index := range stamps {
		stamps[index] = make([]atomic.Int64, samples+1)
	}
	results := make(chan sceneLoadResult, players)
	for index, client := range clients {
		go func(self int, client *websocket.Conn) {
			result := receiveSceneLoad(ctx, client, uint64(self+1), players, samples, epoch, stamps, groupSize)
			results <- result
		}(index, client)
	}
	start := time.Now().Add(25 * time.Millisecond)
	var senders sync.WaitGroup
	for index := 0; index < players; index++ {
		senders.Add(1)
		go func(index int) {
			defer senders.Done()
			target := start.Add(time.Duration(index) * interval / time.Duration(players))
			for seq := 1; seq <= samples; seq++ {
				timer := time.NewTimer(time.Until(target))
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return
				}
				stamps[index][seq].Store(time.Since(epoch).Nanoseconds())
				direction := int32(1)
				if seq%2 == 0 {
					direction = -1
				}
				offset := int32(0)
				if groupSize > 0 {
					offset = int32(index/groupSize) * 100
				}
				handler.broadcastEntityMove(7, 1, uint64(index+1), uint32(seq), world.Vec2i{}, world.Vec2i{X: offset + int32(seq%2)}, protocol.Vec2i{X: (offset + int32(seq%2)) * 1000}, protocol.Vec2i{X: direction}, seq != samples, 1000, int64(seq)*100, int64(seq)*100)
				// 宿主调度停顿后不追赶补发历史输入，和游戏逐帧采样只发送最新状态保持一致。
				target = time.Now().Add(interval)
			}
		}(index)
	}
	senders.Wait()
	var latencies []int64
	var bytes int64
	for index := 0; index < players; index++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatal(result.err)
			}
			latencies = append(latencies, result.latencies...)
			bytes += result.bytes
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	elapsed := time.Since(start)
	neighbors := players - 1
	if groupSize > 0 {
		neighbors = groupSize - 1
	}
	expected := players * neighbors * samples
	if len(latencies) != expected {
		t.Fatalf("received %d frames, want %d", len(latencies), expected)
	}
	for _, conn := range connections[:players] {
		select {
		case <-conn.closed:
			t.Fatalf("healthy connection %s closed under load", conn.ID())
		default:
		}
	}
	if slow != nil {
		waitQueueClosed(t, slow.closed)
	}
	slices.Sort(latencies)
	t.Logf("players=%d blocked=%t received=%d elapsed=%s decoded_p50=%s decoded_p95=%s decoded_max=%s payload_mib_per_sec=%.2f", players, blocked, len(latencies), elapsed.Round(time.Millisecond), time.Duration(latencies[len(latencies)/2]), time.Duration(latencies[(len(latencies)*95+99)/100-1]), time.Duration(latencies[len(latencies)-1]), float64(bytes)/(1024*1024)/elapsed.Seconds())
}

// 每个实体的顺序、场景与终止状态均检查；本用例刻意禁用合并条件，所以必须逐包完整到达。
func receiveSceneLoad(ctx context.Context, client *websocket.Conn, self uint64, players, samples int, epoch time.Time, stamps [][]atomic.Int64, groupSize int) sceneLoadResult {
	result := sceneLoadResult{latencies: make([]int64, 0, (players-1)*samples)}
	deadline, _ := ctx.Deadline()
	if err := client.SetReadDeadline(deadline); err != nil {
		result.err = err
		return result
	}
	last := make([]uint32, players+1)
	completed := 0
	neighbors := players - 1
	if groupSize > 0 {
		neighbors = groupSize - 1
	}
	for completed < neighbors {
		kind, raw, err := client.ReadMessage()
		if err != nil {
			result.err = fmt.Errorf("observer %d read: %w", self, err)
			return result
		}
		packet, err := protocol.DecodePacket(raw)
		if err == nil && groupSize > 0 && packet.Cmd == protocol.CmdEntityEnterPush {
			continue
		}
		if err != nil || kind != websocket.BinaryMessage || packet.Cmd != protocol.CmdEntityMovePush {
			result.err = fmt.Errorf("observer %d invalid movement packet: %v", self, err)
			return result
		}
		var state protocol.EntityMovePush
		if err := protocol.UnmarshalBody(packet.Body, &state); err != nil {
			result.err = err
			return result
		}
		if state.SceneID != 7 || state.SceneVersion != 1 || state.EntityID == 0 || state.EntityID > uint64(players) || state.EntityID == self || state.MoveSeq == 0 || state.MoveSeq > uint32(samples) {
			result.err = fmt.Errorf("observer %d invalid movement identity", self)
			return result
		}
		if groupSize > 0 && (int(state.EntityID)-1)/groupSize != (int(self)-1)/groupSize {
			result.err = fmt.Errorf("movement leaked outside AOI neighborhood")
			return result
		}
		if state.MoveSeq != last[state.EntityID]+1 || state.Moving != (state.MoveSeq != uint32(samples)) {
			result.err = fmt.Errorf("observer %d entity %d sequence/state mismatch: last=%d next=%d", self, state.EntityID, last[state.EntityID], state.MoveSeq)
			return result
		}
		stamp := stamps[state.EntityID-1][state.MoveSeq].Load()
		if stamp <= 0 {
			result.err = fmt.Errorf("observer %d missing send timestamp", self)
			return result
		}
		result.latencies = append(result.latencies, time.Since(epoch).Nanoseconds()-stamp)
		result.bytes += int64(len(raw))
		last[state.EntityID] = state.MoveSeq
		if state.MoveSeq == uint32(samples) {
			completed++
		}
	}
	return result
}
