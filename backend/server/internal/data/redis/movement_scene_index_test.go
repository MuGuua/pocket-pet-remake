package redis

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"pocket-pet-remake/server/internal/module/world"
)

// 仅连接显式指定的本机临时 Redis，使用独立前缀并只清理本次前缀，禁止连接正式实例。
func localMovementRedis(t *testing.T) (*MovementStateRepository, *goredis.Client) {
	t.Helper()
	addr := os.Getenv("POCKET_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("set POCKET_REDIS_TEST_ADDR for isolated real Redis regressions")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("Redis test requires a dedicated loopback instance")
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("pocket_scene_test_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		keys, _ := client.Keys(context.Background(), prefix+":*").Result()
		if len(keys) > 0 {
			client.Del(context.Background(), keys...)
		}
		client.Close()
	})
	return NewMovementStateRepository(&RuntimeClient{inner: client}, prefix), client
}

// 覆盖初始化、跨仓储实例读取、CAS、切图、旧连接迟到清理、断开、重连与租约过期。
func TestMovementSceneIndexRealRedis(t *testing.T) {
	repo, _ := localMovementRedis(t)
	ctx := context.Background()
	state := movementStateFixture()
	if err := repo.Initialize(ctx, state); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSceneOnline(ctx, state.PlayerID, state.SceneID, state.SessionID, "conn_old", time.Second); err != nil {
		t.Fatal(err)
	}
	remoteClient := goredis.NewClient(&goredis.Options{Addr: os.Getenv("POCKET_REDIS_TEST_ADDR")})
	t.Cleanup(func() { remoteClient.Close() })
	remote := NewMovementStateRepository(&RuntimeClient{inner: remoteClient}, repo.keyPrefix)
	ids, err := remote.ListSceneOnlinePlayerIDs(ctx, 3)
	if err != nil || !reflect.DeepEqual(ids, []uint64{state.PlayerID}) {
		t.Fatalf("shared index=%v err=%v", ids, err)
	}
	oldSeq := state.LastMoveSeq
	state.LastMoveSeq++
	if err := repo.CompareAndSet(ctx, oldSeq, state); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompareAndSet(ctx, oldSeq, state); err != world.ErrMovementSequenceStale {
		t.Fatalf("stale CAS=%v", err)
	}
	state.SceneID = 4
	state.SceneVersion++
	if err := repo.Initialize(ctx, state); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSceneOnline(ctx, state.PlayerID, 4, state.SessionID, "conn_new", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := repo.UnmarkSceneOnline(ctx, state.PlayerID, "conn_old"); err != nil {
		t.Fatal(err)
	}
	old, _ := remote.ListSceneOnlinePlayerIDs(ctx, 3)
	current, _ := remote.ListSceneOnlinePlayerIDs(ctx, 4)
	if len(old) != 0 || !reflect.DeepEqual(current, ids) {
		t.Fatalf("transfer/reconnect index old=%v new=%v", old, current)
	}
	if err := repo.UnmarkSceneOnline(ctx, state.PlayerID, "conn_new"); err != nil {
		t.Fatal(err)
	}
	current, _ = remote.ListSceneOnlinePlayerIDs(ctx, 4)
	if len(current) != 0 {
		t.Fatal("disconnect retained online member")
	}
	if _, err := repo.Load(ctx, state.PlayerID); err != nil {
		t.Fatal("disconnect deleted reconnect authority state")
	}
	if err := repo.MarkSceneOnline(ctx, state.PlayerID, 4, state.SessionID, "conn_reconnect", 40*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		current, err = repo.ListSceneOnlinePlayerIDs(ctx, 4)
		if err != nil {
			t.Fatal(err)
		}
		if len(current) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired owner remained online")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := repo.Load(ctx, state.PlayerID); err != nil {
		t.Fatal("lease expiration removed persistent reconnect state")
	}
	if err := repo.MarkSceneOnline(ctx, state.PlayerID, 4, state.SessionID, "conn_final", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, state.PlayerID); err != nil {
		t.Fatal(err)
	}
	current, err = repo.ListSceneOnlinePlayerIDs(ctx, 4)
	if err != nil || len(current) != 0 {
		t.Fatal("explicit delete leaked scene index")
	}
}

// 网络故障必须原样返回，不能用旧数据假装成功；恢复连接后同一仓储可继续读取当前权威状态。
func TestMovementRedisTemporaryFailureAndRecovery(t *testing.T) {
	repo, client := localMovementRedis(t)
	ctx := context.Background()
	state := movementStateFixture()
	if err := repo.Initialize(ctx, state); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.Load(canceled, state.PlayerID); err == nil {
		t.Fatal("canceled Redis request unexpectedly succeeded")
	}
	if err := repo.CompareAndSet(canceled, state.LastMoveSeq, state); err == nil {
		t.Fatal("failed Redis update was hidden")
	}
	latest, err := repo.Load(ctx, state.PlayerID)
	if err != nil || *latest != state {
		t.Fatalf("authority changed after failed operation: %+v %v", latest, err)
	}
	if err := client.Del(ctx, repo.playerKey(state.PlayerID)).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Load(ctx, state.PlayerID); err != world.ErrMovementStateNotFound {
		t.Fatal("lost state did not require authority recovery")
	}
	if err := repo.Initialize(ctx, state); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.Load(ctx, state.PlayerID); err != nil || *got != state {
		t.Fatal("reinitialization did not restore authority")
	}
}
