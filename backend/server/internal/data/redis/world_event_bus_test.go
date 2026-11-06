package redis

import (
	"context"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

// 独立消费游标必须广播给两个实例，不能像消费组那样把消息分走；截断后必须触发恢复。
func TestSceneEventBusRealRedis(t *testing.T) {
	repo, client := localMovementRedis(t)
	first := NewSceneEventBus(&RuntimeClient{inner: client}, repo.keyPrefix)
	second := NewSceneEventBus(&RuntimeClient{inner: client}, repo.keyPrefix)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := []chan world.SceneEvent{make(chan world.SceneEvent, 64), make(chan world.SceneEvent, 64)}
	reset := make(chan struct{}, 8)
	done := make(chan error, 2)
	for index, bus := range []world.SceneEventBus{first, second} {
		go func(index int, bus world.SceneEventBus) {
			done <- bus.Consume(ctx, func(event world.SceneEvent) { received[index] <- event }, func() { reset <- struct{}{} })
		}(index, bus)
	}
	ready := []bool{false, false}
	deadline := time.Now().Add(3 * time.Second)
	for !ready[0] || !ready[1] {
		if time.Now().After(deadline) {
			t.Fatal("consumers not ready")
		}
		if err := first.Publish(ctx, world.SceneEvent{Origin: "handshake"}); err != nil {
			t.Fatal(err)
		}
		for index, ch := range received {
			select {
			case <-ch:
				ready[index] = true
			default:
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	actual := world.SceneEvent{Origin: "actual", Owner: "owner", SceneID: 7, PlayerID: 1, Command: 2013, Body: []byte(`{"test":1}`)}
	if err := first.Publish(ctx, actual); err != nil {
		t.Fatal(err)
	}
	for _, ch := range received {
		for {
			select {
			case event := <-ch:
				if event.Origin == actual.Origin {
					if event.Owner != actual.Owner || string(event.Body) != string(actual.Body) {
						t.Fatal("event changed")
					}
					goto next
				}
			case <-time.After(time.Second):
				t.Fatal("broadcast missing")
			}
		}
	next:
	}
	// 删除并创建较新的流会使旧 cursor 落在保留窗之外。
	key := repo.keyPrefix + ":world:events"
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Incr(ctx, key+":sequence").Err(); err != nil {
		t.Fatal(err)
	}
	if err := first.Publish(ctx, world.SceneEvent{Origin: "after_reset"}); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		select {
		case <-reset:
		case <-time.After(2 * time.Second):
			t.Fatal("stream gap was silent")
		}
	}
	cancel()
	for index := 0; index < 2; index++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("consumer leaked")
		}
	}
}

// 数值编号边界不能用字符串字典序比较。
func TestStreamIDOrdering(t *testing.T) {
	if !streamIDLess("9-99", "10-0") || !streamIDLess("10-9", "10-10") || streamIDLess("10-10", "10-9") || streamIDLess("10-1", "10-1") {
		t.Fatal("invalid stream ordering")
	}
}
