// Package telemetry 保存有界的进程运行指标，不包含玩家数据、SQL、Redis key 或请求载荷。
package telemetry

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// 每类指标只保留最近 256 次样本，总类型数量也受限，避免随玩家或命令输入无限增长。
const sampleCapacity = 256
const seriesCapacity = 128

// Snapshot 的分位数仅来自近期有界样本；计数/失败/最大值为该进程启动以来累计值。
type Snapshot struct {
	Key           string `json:"key"`
	Unit          string `json:"unit"`
	Count         uint64 `json:"count"`
	Failures      uint64 `json:"failures"`
	RecentSamples int    `json:"recent_samples"`
	Last          int64  `json:"last"`
	P50           int64  `json:"p50"`
	P95           int64  `json:"p95"`
	Max           int64  `json:"max"`
}

// 环形槽各自拥有短锁；不同发送协程通常写不同槽，避免所有高频连接争用同一写锁。
// 票号防止被调度暂停的旧记录覆盖已经发布的新样本，累计计数单独用原子操作维护。
type sampleSlot struct {
	mu     sync.Mutex
	ticket uint64
	value  int64
}
type series struct {
	values   [sampleCapacity]sampleSlot
	count    atomic.Uint64
	failures atomic.Uint64
	max      atomic.Int64
	unit     string
}

// Registry 是有界并发指标容器；独立实例也可用于测试，避免修改运行服务的全局状态。
type Registry struct {
	mu     sync.RWMutex
	series map[string]*series
}

// Default 是当前服务进程共享的观测容器，服务重启后自然重新计数。
var Default = &Registry{series: make(map[string]*series)}

// Record 记录非负整数样本；key 必须是调用方固定的运维维度，不得拼接玩家 ID 或敏感内容。
func (r *Registry) Record(key, unit string, value int64, failed bool) {
	if value < 0 {
		value = 0
	}
	r.mu.RLock()
	current := r.series[key]
	r.mu.RUnlock()
	if current == nil {
		r.mu.Lock()
		current = r.series[key]
		if current == nil {
			if len(r.series) >= seriesCapacity {
				r.mu.Unlock()
				return
			}
			if r.series == nil {
				r.series = make(map[string]*series)
			}
			current = &series{unit: unit}
			r.series[key] = current
		}
		r.mu.Unlock()
	}
	ticket := current.count.Add(1)
	slot := &current.values[(ticket-1)%sampleCapacity]
	slot.mu.Lock()
	if ticket > slot.ticket {
		slot.ticket = ticket
		slot.value = value
	}
	slot.mu.Unlock()
	if failed {
		current.failures.Add(1)
	}
	for previous := current.max.Load(); value > previous; previous = current.max.Load() {
		if current.max.CompareAndSwap(previous, value) {
			break
		}
	}

}

// Duration 统一以整数微秒记录耗时，不将跨机器绝对时间直接相减。
func Duration(key string, elapsed time.Duration, failed bool) {
	Default.Record(key, "us", elapsed.Microseconds(), failed)
}

// Snapshots 复制当前数据后排序计算分位，排序期间不持有热路径锁，也不暴露内部缓冲。
func (r *Registry) Snapshots() []Snapshot {
	r.mu.RLock()
	entries := make(map[string]*series, len(r.series))
	for key, value := range r.series {
		entries[key] = value
	}
	r.mu.RUnlock()
	result := make([]Snapshot, 0, len(entries))
	for key, entry := range entries {
		// 快照与持续写入可交错，RecentSamples 明确报告本次实际读到的近期样本数。
		type published struct {
			ticket uint64
			value  int64
		}
		copied := make([]published, 0, sampleCapacity)
		for index := range entry.values {
			slot := &entry.values[index]
			slot.mu.Lock()
			if slot.ticket > 0 {
				copied = append(copied, published{slot.ticket, slot.value})
			}
			slot.mu.Unlock()
		}
		failures := entry.failures.Load()
		count := entry.count.Load()
		maximum := entry.max.Load()
		values := make([]int64, 0, len(copied))
		var latest uint64
		snapshot := Snapshot{Key: key, Unit: entry.unit, Count: count, Failures: failures, Max: maximum}
		for _, sample := range copied {
			if count >= sampleCapacity && sample.ticket <= count-sampleCapacity {
				continue
			}
			values = append(values, sample.value)
			if sample.ticket > latest {
				latest = sample.ticket
				snapshot.Last = sample.value
			}
			if sample.value > snapshot.Max {
				snapshot.Max = sample.value
			}
		}
		snapshot.RecentSamples = len(values)
		if len(values) > 0 {
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			snapshot.P50 = values[(len(values)*50+99)/100-1]
			snapshot.P95 = values[(len(values)*95+99)/100-1]
		}
		result = append(result, snapshot)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}
