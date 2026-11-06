package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"
	"pocket-pet-remake/server/internal/config"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
	wstransport "pocket-pet-remake/server/internal/transport/ws"
)

// 仅打开明确指定的本机独立测试库/Redis，创建临时 schema 和键前缀，清理不触碰其他数据。
func movementPipelineFixture(t *testing.T) (*sql.DB, *world.Service) {
	t.Helper()
	dsn, addr := os.Getenv("POCKET_BAG_PAGE_TEST_DSN"), os.Getenv("POCKET_REDIS_TEST_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("set dedicated local PostgreSQL and Redis test endpoints")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database != "pocket_bag_page_test" || cfg.Host != "127.0.0.1" || !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("pipeline requires isolated loopback dependencies")
	}
	setup := stdlib.OpenDB(*cfg)
	schema := fmt.Sprintf("movement_pipeline_%d", time.Now().UnixNano())
	if _, err := setup.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	db.SetMaxOpenConns(8)
	if _, err := db.Exec(`CREATE TABLE player(id bigint PRIMARY KEY,scene_id bigint,pos_x integer,pos_y integer,position_version bigint NOT NULL DEFAULT 0)`); err != nil {
		t.Fatal(err)
	}
	redisClient, err := redisrepo.Open(config.RedisConfig{Addr: addr})
	if err != nil {
		t.Fatal(err)
	}
	prefix := schema
	movementRepo := redisrepo.NewMovementStateRepository(redisClient, prefix)
	service := world.NewService(nil)
	service.SetMovementStateRepository(movementRepo)
	t.Cleanup(func() {
		redisAdmin := goredis.NewClient(&goredis.Options{Addr: addr})
		keys, _ := redisAdmin.Keys(context.Background(), prefix+":*").Result()
		if len(keys) > 0 {
			redisAdmin.Del(context.Background(), keys...)
		}
		redisAdmin.Close()
		redisClient.Close()
		db.Close()
		if _, err := setup.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
		setup.Close()
	})
	return db, service
}

// 实际 PostgreSQL 锁超时后 dirty 必须重入队，恢复后落库，迟到旧版本不能覆盖新值。
func TestMovementPipelinePostgresFailureRecovery(t *testing.T) {
	db, service := movementPipelineFixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO player VALUES(1,7,0,0,1)`); err != nil {
		t.Fatal(err)
	}
	initial := world.MovementState{PlayerID: 1, SessionID: "pipeline", SceneID: 7, SceneVersion: 1, PositionVersion: 1}
	if err := service.InitializeMovementState(ctx, initial); err != nil {
		t.Fatal(err)
	}
	moved, err := service.MovePlayer(ctx, world.MovePlayerInput{PlayerID: 1, SessionID: "pipeline", SceneID: 7, MoveSeq: 1, HasCandidate: true, CandidatePos: world.Vec2i{X: 5000}, CandidateCell: world.Vec2i{X: 5}, ServerTickMS: time.Now().UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	writer := pgrepo.NewPlayerRepository(db)
	worker, err := newMovementPersistenceWorker(service, writer, log.New(io.Discard, "", 0), time.Second, 100)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback()
	if _, err := locked.Exec(`LOCK TABLE player IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	timeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	result, err := worker.flushOnce(timeout)
	cancel()
	locked.Rollback()
	if err == nil || result.Requeued != 1 {
		t.Fatalf("Postgres failure lost dirty state: %+v %v", result, err)
	}
	result, err = worker.flushOnce(ctx)
	if err != nil || result.Persisted != 1 {
		t.Fatalf("Postgres recovery failed: %+v %v", result, err)
	}
	if applied, err := writer.UpdatePositionIfNewer(ctx, 1, 7, 0, 0, 1); err != nil || applied {
		t.Fatal("old version overwrote recovered state")
	}
	var x int32
	var version uint64
	if err := db.QueryRow(`SELECT pos_x,position_version FROM player WHERE id=1`).Scan(&x, &version); err != nil || x != 5 || version != moved.State.PositionVersion {
		t.Fatalf("wrong recovered position %d/%d: %v", x, version, err)
	}
}

// 显式开启 20/50/100 人真实上报→路由→Redis CAS→广播→PostgreSQL 写回压测，默认单测跳过。
func TestMovementPipelineLoad(t *testing.T) {
	if os.Getenv("POCKET_WS_PIPELINE_LOAD_TEST") != "1" {
		t.Skip("set POCKET_WS_PIPELINE_LOAD_TEST=1 for real dependency/network load regression")
	}
	for _, players := range []int{20, 50, 100} {
		t.Run(fmt.Sprintf("players_%d", players), func(t *testing.T) { runMovementPipelineLoad(t, players) })
	}
}

type pipelineReadResult struct {
	latencies []int64
	err       error
}

// 位置元数据使用正式仓储/领域/worker，完整账号登录、Godot 渲染和公网链路不在此用例范围。
func runMovementPipelineLoad(t *testing.T, players int) {
	t.Helper()
	samples := 20
	if raw := os.Getenv("POCKET_WS_PIPELINE_SAMPLES"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 2 || value > 600 {
			t.Fatal("POCKET_WS_PIPELINE_SAMPLES must be between 2 and 600")
		}
		samples = value
	}
	db, worldService := movementPipelineFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(samples)*100*time.Millisecond+20*time.Second)
	defer cancel()
	logger := log.New(io.Discard, "", 0)
	sessions := session.NewService(logger, time.Second, 10*time.Second)
	handler := wstransport.NewWorldHandler(sessions, nil, nil, nil, nil, worldService, nil, nil)
	router := wstransport.NewRouter(nil, handler, nil, nil, nil, nil, nil, nil, sessions)
	worker, err := newMovementPersistenceWorker(worldService, pgrepo.NewPlayerRepository(db), logger, 100*time.Millisecond, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.SetMovementFinalPersister(worker)
	sessions.SetDisconnectHandler(func(id uint64) {
		persistCtx, end := context.WithTimeout(context.Background(), time.Second)
		defer end()
		_ = worker.PersistPlayerMovement(persistCtx, id)
	})
	sessions.AddDisconnectHandler(handler.HandleOfflineSession)
	accepted := make(chan *wstransport.Connection, players)
	var handlers sync.WaitGroup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.Add(1)
		defer handlers.Done()
		socket, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn := wstransport.NewConnection(r.URL.Query().Get("id"), socket)
		defer func() { sessions.Disconnect(conn.ID()); conn.Close() }()
		accepted <- conn
		for {
			kind, raw, err := socket.ReadMessage()
			if err != nil {
				return
			}
			if kind == websocket.BinaryMessage {
				if err := router.Handle(conn, raw); err != nil {
					return
				}
			}
		}
	}))
	clients := make([]*websocket.Conn, 0, players)
	connections := make([]*wstransport.Connection, 0, players)
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { worker.Run(workerCtx); close(workerDone) }()
	t.Cleanup(func() {
		cancel()
		stopWorker()
		for _, client := range clients {
			client.Close()
		}
		for _, conn := range connections {
			conn.Close()
		}
		server.Close()
		handlers.Wait()
		<-workerDone
	})
	for index := 0; index < players; index++ {
		id := uint64(index + 1)
		if _, err := db.Exec(`INSERT INTO player VALUES($1,7,0,0,1)`, id); err != nil {
			t.Fatal(err)
		}
		client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+fmt.Sprintf("?id=%d", id), nil)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
		var conn *wstransport.Connection
		select {
		case conn = <-accepted:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		connections = append(connections, conn)
		current, err := sessions.Bind(id, conn)
		if err != nil {
			t.Fatal(err)
		}
		state := world.MovementState{PlayerID: id, SessionID: current.ID, SceneID: 7, SceneVersion: 1, PositionVersion: 1, Facing: world.Vec2i{X: 1}, Speed: 1000}
		if err := worldService.InitializeMovementState(ctx, state); err != nil {
			t.Fatal(err)
		}
	}
	epoch := time.Now()
	stamps := make([][]atomic.Int64, players)
	for index := range stamps {
		stamps[index] = make([]atomic.Int64, samples+2)
	}
	results := make(chan pipelineReadResult, players)
	for index, client := range clients {
		go func(self int, client *websocket.Conn) {
			results <- readMovementPipeline(ctx, client, self, players, samples, epoch, stamps)
		}(index, client)
	}
	// 首次普通移动登记同屏成员，暖身帧不计入统计，避免把账号准备时间混入测量。
	for _, conn := range connections {
		packet, _ := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 1, 0, protocol.MoveIntentReq{MoveSeq: 1, SceneID: 7, TargetPos: &protocol.Vec2i{}, Moving: pipelineBool(true)})
		raw, _ := protocol.EncodePacket(packet)
		if err := router.Handle(conn, raw); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now().Add(25 * time.Millisecond)
	var senders sync.WaitGroup
	failures := make(chan error, players)
	for index, client := range clients {
		senders.Add(1)
		go func(index int, client *websocket.Conn) {
			defer senders.Done()
			target := start.Add(time.Duration(index) * 100 * time.Millisecond / time.Duration(players))
			for n := 0; n < samples; n++ {
				timer := time.NewTimer(time.Until(target))
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return
				}
				// 长时间压测复用现有心跳续租，避免只生成移动而虚构永不过期的在线连接。
				if n > 0 && n%50 == 0 {
					heartbeat, _ := protocol.NewJSONPacket(protocol.CmdHeartbeatReq, uint32(100000+n), 0, protocol.HeartbeatReq{ClientTimeMS: time.Now().UnixMilli()})
					encoded, _ := protocol.EncodePacket(heartbeat)
					if err := client.WriteMessage(websocket.BinaryMessage, encoded); err != nil {
						failures <- err
						return
					}
				}
				seq := uint32(n + 2)
				x := int32(n % 2)
				direction := int32(1)
				if n%2 == 0 {
					direction = -1
				}
				packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, seq, 0, protocol.MoveIntentReq{MoveSeq: seq, SceneID: 7, TargetPos: &protocol.Vec2i{X: x}, PrecisePos: &protocol.Vec2i{X: x * 1000}, Facing: &protocol.Vec2i{X: direction}, Moving: pipelineBool(n != samples-1), ClientTick: int64(seq) * 100})
				if err != nil {
					failures <- err
					return
				}
				raw, err := protocol.EncodePacket(packet)
				if err != nil {
					failures <- err
					return
				}
				stamps[index][seq].Store(time.Since(epoch).Nanoseconds())
				client.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := client.WriteMessage(websocket.BinaryMessage, raw); err != nil {
					failures <- err
					return
				}
				target = time.Now().Add(100 * time.Millisecond)
			}
		}(index, client)
	}
	senders.Wait()
	select {
	case err := <-failures:
		t.Fatal(err)
	default:
	}
	var latencies []int64
	for index := 0; index < players; index++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatal(result.err)
			}
			latencies = append(latencies, result.latencies...)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	elapsed := time.Since(start)
	stopWorker()
	<-workerDone
	if _, err := worker.DrainDirtyMovement(ctx); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < players; index++ {
		id := uint64(index + 1)
		state, err := worldService.LoadMovementState(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		var x int32
		var version uint64
		if err := db.QueryRow(`SELECT pos_x,position_version FROM player WHERE id=$1`, id).Scan(&x, &version); err != nil {
			t.Fatal(err)
		}
		if x != 1 || state.LastMoveSeq != uint32(samples+1) || state.Moving || version != state.PositionVersion {
			t.Fatalf("final state not persisted id=%d x=%d seq=%d version=%d/%d", id, x, state.LastMoveSeq, version, state.PositionVersion)
		}
	}
	online, err := worldService.ListSceneOnlinePlayerIDs(ctx, 7)
	if err != nil || len(online) != players {
		t.Fatalf("online leases lost under movement load: %d/%d %v", len(online), players, err)
	}
	if len(latencies) != players*(players-1)*samples {
		t.Fatal("pipeline dropped boundary frames")
	}
	slices.Sort(latencies)
	t.Logf("players=%d movement_reports=%d observer_frames=%d elapsed=%s upstream_to_observer_p50=%s p95=%s all_final_positions_persisted=true", players, players*samples, len(latencies), elapsed.Round(time.Millisecond), time.Duration(latencies[len(latencies)/2]), time.Duration(latencies[(len(latencies)*95+99)/100-1]))
}

// 只接收暖身以后的连续状态，停步与场景/实体顺序不允许被隐藏丢失。
func readMovementPipeline(ctx context.Context, client *websocket.Conn, self, players, samples int, epoch time.Time, stamps [][]atomic.Int64) pipelineReadResult {
	result := pipelineReadResult{}
	deadline, _ := ctx.Deadline()
	client.SetReadDeadline(deadline)
	last := make([]uint32, players+1)
	for index := range last {
		last[index] = 1
	}
	completed := 0
	for completed < players-1 {
		kind, raw, err := client.ReadMessage()
		if err != nil {
			result.err = err
			return result
		}
		packet, err := protocol.DecodePacket(raw)
		if err == nil && packet.Cmd == protocol.CmdHeartbeatResp {
			continue
		}
		if err != nil || kind != websocket.BinaryMessage || packet.Cmd != protocol.CmdEntityMovePush {
			result.err = fmt.Errorf("unexpected pipeline packet: %v", err)
			return result
		}
		var state protocol.EntityMovePush
		if err := protocol.UnmarshalBody(packet.Body, &state); err != nil {
			result.err = err
			return result
		}
		if state.MoveSeq == 1 {
			continue
		}
		if state.EntityID == 0 || state.EntityID > uint64(players) || state.EntityID == uint64(self+1) || state.SceneID != 7 || state.SceneVersion != 1 || state.MoveSeq != last[state.EntityID]+1 || state.MoveSeq > uint32(samples+1) || state.Moving != (state.MoveSeq != uint32(samples+1)) {
			result.err = fmt.Errorf("pipeline identity/sequence/stop mismatch: observer=%d state=%+v", self+1, state)
			return result
		}
		stamp := stamps[state.EntityID-1][state.MoveSeq].Load()
		if stamp <= 0 {
			result.err = fmt.Errorf("pipeline timestamp missing")
			return result
		}
		result.latencies = append(result.latencies, time.Since(epoch).Nanoseconds()-stamp)
		last[state.EntityID] = state.MoveSeq
		if state.MoveSeq == uint32(samples+1) {
			completed++
		}
	}
	return result
}
func pipelineBool(value bool) *bool { return &value }

// 只有显式指定本任务的临时 Docker Redis 容器才运行重启测试，禁止对任意运行实例执行重启。
func TestMovementPipelineRedisRestartRecovery(t *testing.T) {
	container := os.Getenv("POCKET_REDIS_TEST_CONTAINER")
	if container == "" {
		t.Skip("set the disposable pocket-redis-latency-* container for an isolated restart test")
	}
	if !strings.HasPrefix(container, "pocket-redis-latency-") || os.Getenv("POCKET_REDIS_TEST_ADDR") != "127.0.0.1:56379" {
		t.Fatal("restart test is restricted to the disposable task Redis container/port")
	}
	db, service := movementPipelineFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := db.Exec(`INSERT INTO player VALUES(1,7,0,0,1)`); err != nil {
		t.Fatal(err)
	}
	initial := world.MovementState{PlayerID: 1, SessionID: "before_restart", SceneID: 7, SceneVersion: 1, PositionVersion: 1}
	if err := service.InitializeMovementState(ctx, initial); err != nil {
		t.Fatal(err)
	}
	if err := service.MarkSceneOnline(ctx, 1, 7, initial.SessionID, "owner_before", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	moved, err := service.MovePlayer(ctx, world.MovePlayerInput{PlayerID: 1, SessionID: initial.SessionID, SceneID: 7, MoveSeq: 1, HasCandidate: true, CandidatePos: world.Vec2i{X: 5000}, CandidateCell: world.Vec2i{X: 5}, ServerTickMS: time.Now().UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := newMovementPersistenceWorker(service, pgrepo.NewPlayerRepository(db), log.New(io.Discard, "", 0), time.Second, 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.flushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.CommandContext(ctx, "docker", "restart", container).CombinedOutput(); err != nil {
		t.Fatalf("restart disposable Redis: %v %s", err, output)
	}
	for {
		_, err := service.LoadMovementState(ctx, 1)
		if errors.Is(err, world.ErrMovementStateNotFound) {
			break
		}
		if err == nil {
			t.Fatal("disposable Redis unexpectedly retained volatile authority after restart")
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	// 从真正持久化档案恢复，而不是从客户端坐标或测试中保存的移动对象复制回去。
	restored := world.MovementState{PlayerID: 1, SessionID: "after_restart", SceneVersion: 2}
	if err := db.QueryRow(`SELECT scene_id,pos_x,pos_y,position_version FROM player WHERE id=1`).Scan(&restored.SceneID, &restored.PersistedPos.X, &restored.PersistedPos.Y, &restored.PositionVersion); err != nil {
		t.Fatal(err)
	}
	if restored.PersistedPos.X != 5 || restored.PositionVersion != moved.State.PositionVersion {
		t.Fatal("checkpoint missing before restart")
	}
	restored.PrecisePos = world.Vec2i{X: restored.PersistedPos.X * 1000, Y: restored.PersistedPos.Y * 1000}
	if err := service.InitializeMovementState(ctx, restored); err != nil {
		t.Fatal(err)
	}
	if _, err := service.MovePlayer(ctx, world.MovePlayerInput{PlayerID: 1, SessionID: "before_restart", SceneID: 7, MoveSeq: 2, HasCandidate: true}); !errors.Is(err, world.ErrMovementSessionMismatch) {
		t.Fatal("old session overwrote recovered authority")
	}
	if err := service.MarkSceneOnline(ctx, 1, 7, restored.SessionID, "owner_after", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	ids, err := service.ListSceneOnlinePlayerIDs(ctx, 7)
	if err != nil || len(ids) != 1 || ids[0] != 1 {
		t.Fatal("reconnect failed to rebuild scene index")
	}
	next, err := service.MovePlayer(ctx, world.MovePlayerInput{PlayerID: 1, SessionID: restored.SessionID, SceneID: 7, MoveSeq: 1, HasCandidate: true, CandidatePos: world.Vec2i{X: 6000}, CandidateCell: world.Vec2i{X: 6}, ServerTickMS: time.Now().UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.flushOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var x int32
	var version uint64
	if err := db.QueryRow(`SELECT pos_x,position_version FROM player WHERE id=1`).Scan(&x, &version); err != nil || x != 6 || version != next.State.PositionVersion || version <= restored.PositionVersion {
		t.Fatalf("recovered persistence failed: %d/%d %v", x, version, err)
	}
}
