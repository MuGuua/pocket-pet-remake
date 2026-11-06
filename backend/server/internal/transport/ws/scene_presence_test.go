package wstransport

import (
	"context"
	"fmt"
	"io"
	"log"
	"slices"
	"sync"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/protocol"
)

// 正式生命周期入口必须同步维护正反索引，重复进入和断开必须幂等。
func TestScenePresenceLifecycle(t *testing.T) {
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	handler := NewWorldHandler(service, nil, nil, nil, nil, nil, nil, nil)
	handler.enterPlayerScene(context.Background(), 1, 7)
	handler.enterPlayerScene(context.Background(), 2, 7)
	handler.enterPlayerScene(context.Background(), 3, 8)
	handler.enterPlayerScene(context.Background(), 1, 7)
	assertSceneMembers(t, handler, 7, 0, []uint64{1, 2})
	assertSceneMembers(t, handler, 7, 1, []uint64{2})
	assertSceneMembers(t, handler, 9, 0, nil)
	handler.movePlayerScenePresence(1, 7, 8)
	handler.movePlayerScenePresence(1, 8, 8)
	assertSceneMembers(t, handler, 7, 0, []uint64{2})
	assertSceneMembers(t, handler, 8, 0, []uint64{1, 3})
	handler.HandleSessionDisconnect(2)
	handler.HandleSessionDisconnect(2)
	if _, exists := handler.scenePlayers[7]; exists {
		t.Fatal("empty old-scene bucket retained")
	}
	handler.enterPlayerScene(context.Background(), 2, 9)
	handler.enterPlayerScene(context.Background(), 2, 8)
	assertSceneMembers(t, handler, 8, 0, []uint64{1, 2, 3})
	for _, id := range []uint64{1, 2, 3} {
		handler.HandleSessionDisconnect(id)
	}
	if len(handler.scenePlayers) != 0 || len(handler.playerScenes) != 0 {
		t.Fatal("disconnect leaked presence")
	}
}

// 其他地图成员不混入结果，调用方修改返回切片不能影响索引。
func TestScenePresenceSnapshotIsolation(t *testing.T) {
	handler := &WorldHandler{}
	for id := uint64(1); id <= 10000; id++ {
		handler.setPlayerScenePresence(id, uint32(id%100)+1)
	}
	ids := handler.playerIDsInScene(8, 7)
	if len(ids) != 99 {
		t.Fatalf("scene members = %d, want 99", len(ids))
	}
	for _, id := range ids {
		if id == 7 || id%100 != 7 {
			t.Fatalf("unexpected scene member %d", id)
		}
	}
	ids[0] = 0
	if slices.Contains(handler.playerIDsInScene(8, 7), uint64(0)) {
		t.Fatal("snapshot mutated index")
	}
}

// 实际广播路径验证切图旧场景通知、目标场景接收者和断开清理。
func TestScenePresenceBroadcastBoundaries(t *testing.T) {
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	handler := NewWorldHandler(service, nil, nil, nil, nil, nil, nil, nil)
	oldPeer := &fakeConn{id: "old-scene-peer"}
	newPeer := &fakeConn{id: "new-scene-peer"}
	mover := &fakeConn{id: "scene-mover"}
	for id, conn := range map[uint64]*fakeConn{1: mover, 2: oldPeer, 3: newPeer} {
		if _, err := service.Bind(id, conn); err != nil {
			t.Fatal(err)
		}
	}
	handler.setPlayerScenePresence(1, 7)
	handler.setPlayerScenePresence(2, 7)
	handler.setPlayerScenePresence(3, 8)
	handler.movePlayerScenePresence(1, 7, 8)
	if len(oldPeer.packets) != 1 || oldPeer.packets[0].Cmd != protocol.CmdEntityLeavePush || len(newPeer.packets) != 0 || len(mover.packets) != 0 {
		t.Fatal("transfer leave reached wrong scene")
	}
	handler.sendPacketToScene(8, 1, queueTestPacket(42))
	if len(newPeer.packets) != 1 || newPeer.packets[0].Seq != 42 || len(oldPeer.packets) != 1 {
		t.Fatal("new-scene broadcast boundary changed")
	}
	handler.HandleSessionDisconnect(1)
	if len(newPeer.packets) != 2 || newPeer.packets[1].Cmd != protocol.CmdEntityLeavePush {
		t.Fatal("disconnect did not notify current scene")
	}
	handler.sendPacketToScene(8, 0, queueTestPacket(43))
	if len(mover.packets) != 0 {
		t.Fatal("disconnected presence received broadcast")
	}
}

// 并发读写与断开后检查双向一致性，配合 race 检查锁保护。
func TestScenePresenceConcurrentLifecycle(t *testing.T) {
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	handler := NewWorldHandler(service, nil, nil, nil, nil, nil, nil, nil)
	var workers sync.WaitGroup
	for id := uint64(1); id <= 16; id++ {
		workers.Add(1)
		go func(id uint64) {
			defer workers.Done()
			for n := uint32(1); n <= 200; n++ {
				handler.setPlayerScenePresence(id, n%7)
				handler.playerIDsInScene(n%7, id)
				if n%3 == 0 {
					handler.HandleSessionDisconnect(id)
				}
			}
		}(id)
	}
	workers.Wait()
	handler.presenceMu.RLock()
	defer handler.presenceMu.RUnlock()
	count := 0
	for sceneID, members := range handler.scenePlayers {
		if len(members) == 0 {
			t.Fatal("empty bucket retained")
		}
		for id := range members {
			if got, exists := handler.playerScenes[id]; !exists || got != sceneID {
				t.Fatalf("reverse mismatch for %d", id)
			}
			count++
		}
	}
	if count != len(handler.playerScenes) {
		t.Fatal("missing reverse entries")
	}
}

// 固定本地图人数，比较其他地图人数增加的影响；此为索引微基准，不是网络压测。
func BenchmarkScenePresenceLookup(b *testing.B) {
	for _, others := range []int{0, 10000} {
		for _, local := range []int{20, 50, 100} {
			b.Run(fmt.Sprintf("local_%d/others_%d", local, others), func(b *testing.B) {
				handler := &WorldHandler{}
				for id := 1; id <= local+others; id++ {
					sceneID := uint32(8)
					if id <= local {
						sceneID = 7
					}
					handler.setPlayerScenePresence(uint64(id), sceneID)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if len(handler.playerIDsInScene(7, 0)) != local {
						b.Fatal("lookup size changed")
					}
				}
			})
		}
	}
}

// 成员返回顺序没有契约含义，因此比较排序后的集合。
func assertSceneMembers(t *testing.T, handler *WorldHandler, sceneID uint32, excluded uint64, want []uint64) {
	t.Helper()
	got := handler.playerIDsInScene(sceneID, excluded)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("scene %d members = %v, want %v", sceneID, got, want)
	}
}
