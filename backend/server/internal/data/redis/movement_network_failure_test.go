package redis

import (
	"context"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// 本机代理主动断开测试连接，验证真实 Redis 网络故障后的重新连接；不停止或清空 Redis 实例。
type redisFaultProxy struct {
	listener net.Listener
	target   string
	mu       sync.Mutex
	blocked  bool
	closing  bool
	links    map[net.Conn]struct{}
	group    sync.WaitGroup
}

// 所有复制协程都受关闭控制，避免故障测试遗留连接或后台 IO。
func newRedisFaultProxy(t *testing.T, target string) *redisFaultProxy {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxy := &redisFaultProxy{listener: listener, target: target, links: make(map[net.Conn]struct{})}
	proxy.group.Add(1)
	go func() {
		defer proxy.group.Done()
		for {
			incoming, err := listener.Accept()
			if err != nil {
				return
			}
			outgoing, err := net.DialTimeout("tcp", target, time.Second)
			if err != nil {
				incoming.Close()
				continue
			}
			proxy.mu.Lock()
			if proxy.blocked || proxy.closing {
				proxy.mu.Unlock()
				incoming.Close()
				outgoing.Close()
				continue
			}
			proxy.links[incoming] = struct{}{}
			proxy.links[outgoing] = struct{}{}
			proxy.group.Add(2)
			proxy.mu.Unlock()
			copyLink := func(to, from net.Conn) {
				defer proxy.group.Done()
				io.Copy(to, from)
				to.Close()
				from.Close()
				proxy.mu.Lock()
				delete(proxy.links, to)
				delete(proxy.links, from)
				proxy.mu.Unlock()
			}
			go copyLink(incoming, outgoing)
			go copyLink(outgoing, incoming)
		}
	}()
	t.Cleanup(func() {
		proxy.mu.Lock()
		proxy.closing = true
		for conn := range proxy.links {
			conn.Close()
		}
		proxy.mu.Unlock()
		listener.Close()
		proxy.group.Wait()
	})
	return proxy
}

// 阻断后读/CAS 必须失败且不改变权威值，恢复网络后同一仓储实例可自动重新连接。
func TestMovementRedisNetworkFailureRecovery(t *testing.T) {
	repo, _ := localMovementRedis(t)
	state := movementStateFixture()
	ctx := context.Background()
	if err := repo.Initialize(ctx, state); err != nil {
		t.Fatal(err)
	}
	proxy := newRedisFaultProxy(t, os.Getenv("POCKET_REDIS_TEST_ADDR"))
	client := goredis.NewClient(&goredis.Options{Addr: proxy.listener.Addr().String(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond, WriteTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { client.Close() })
	throughProxy := NewMovementStateRepository(&RuntimeClient{inner: client}, repo.keyPrefix)
	if _, err := throughProxy.Load(ctx, state.PlayerID); err != nil {
		t.Fatal(err)
	}
	proxy.mu.Lock()
	proxy.blocked = true
	for conn := range proxy.links {
		conn.Close()
	}
	proxy.mu.Unlock()
	if _, err := throughProxy.Load(ctx, state.PlayerID); err == nil {
		t.Fatal("network failure hidden by stale data")
	}
	next := state
	next.LastMoveSeq++
	if err := throughProxy.CompareAndSet(ctx, state.LastMoveSeq, next); err == nil {
		t.Fatal("CAS succeeded through disabled network")
	}
	proxy.mu.Lock()
	proxy.blocked = false
	proxy.mu.Unlock()
	latest, err := throughProxy.Load(ctx, state.PlayerID)
	if err != nil || *latest != state {
		t.Fatalf("network recovery lost authority: %+v %v", latest, err)
	}
}
