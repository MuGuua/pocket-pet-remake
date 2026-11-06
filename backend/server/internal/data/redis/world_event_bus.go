package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/telemetry"
)

// 每实例独立消费同一有界 Stream，保留短断网事件；超过保留窗必须使观察端重建世界快照。
type SceneEventBus struct {
	client *goredis.Client
	key    string
}

// 复用现有连接池；不强迫旧测试 Redis 适配器实现 Streams。
func NewSceneEventBus(client Client, prefix string) world.SceneEventBus {
	runtime, ok := client.(*RuntimeClient)
	if !ok {
		return nil
	}
	return &SceneEventBus{client: runtime.inner, key: prefix + ":world:events"}
}
func (b *SceneEventBus) Publish(ctx context.Context, event world.SceneEvent) (err error) {
	started := time.Now()
	defer func() { telemetry.Duration("world.relay_publish", time.Since(started), err != nil) }()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Eval(ctx, `local seq=redis.call('INCR',KEYS[2])
return redis.call('XADD',KEYS[1],'MAXLEN',20000,'*','event',ARGV[1],'seq',seq)`, []string{b.key, b.key + ":sequence"}, data).Err()
}

// 每次读取前用同一 Lua 请求检查截断/重启边界；拿不到连续历史时不静默延续旧实体。
func (b *SceneEventBus) Consume(ctx context.Context, receive func(world.SceneEvent), reset func()) error {
	latest, err := b.client.XRevRangeN(ctx, b.key, "+", "-", 1).Result()
	if err != nil {
		return err
	}
	cursor := "0-0"
	var sequence uint64
	if len(latest) > 0 {
		cursor = latest[0].ID
		sequence, _ = strconv.ParseUint(fmt.Sprint(latest[0].Values["seq"]), 10, 64)
	}
	for ctx.Err() == nil {
		// XPENDING 不适用广播：每实例独立游标，不能用消费组把事件分摊给不同节点。
		result, err := b.client.Eval(ctx, `local first=redis.call('XRANGE',KEYS[1],'-','+','COUNT',1)
local last=redis.call('XREVRANGE',KEYS[1],'+','-','COUNT',1)
if #last==0 then return {'0-0','0-0'} end
return {first[1][1],last[1][1]}`, []string{b.key}).StringSlice()
		if err != nil {
			return err
		}
		if len(result) != 2 {
			return fmt.Errorf("invalid world event stream range")
		}
		if cursor != "0-0" && (streamIDLess(cursor, result[0]) || streamIDLess(result[1], cursor)) {
			reset()
			cursor = result[1]
		}
		streams, err := b.client.XRead(ctx, &goredis.XReadArgs{Streams: []string{b.key, cursor}, Count: 256, Block: 500 * time.Millisecond}).Result()
		if errors.Is(err, goredis.Nil) {
			continue
		}
		if err != nil {
			return err
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				raw, ok := message.Values["event"].(string)
				if !ok {
					return fmt.Errorf("invalid world event payload")
				}
				var event world.SceneEvent
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					return err
				}
				next, err := strconv.ParseUint(fmt.Sprint(message.Values["seq"]), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid world event sequence: %w", err)
				}
				if sequence != 0 && next != sequence+1 {
					reset()
				}
				receive(event)
				sequence = next
				cursor = message.ID
			}
		}
	}
	return ctx.Err()
}

// Stream 编号按毫秒/同毫秒序号比较，不能用字典序比较不同位数。
func streamIDLess(a, b string) bool {
	var am, as, bm, bs uint64
	if _, err := fmt.Sscanf(a, "%d-%d", &am, &as); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(b, "%d-%d", &bm, &bs); err != nil {
		return false
	}
	return am < bm || am == bm && as < bs
}
