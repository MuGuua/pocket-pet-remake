package telemetry

import (
	"fmt"
	"sync"
	"testing"
)

// 样本环与累计值具有不同窗口，返回副本不能影响后续读取。
func TestRegistryBoundedRecentSamples(t *testing.T) {
	registry := &Registry{}
	for n := int64(0); n < 1000; n++ {
		registry.Record("test", "us", n, n%10 == 0)
	}
	rows := registry.Snapshots()
	if len(rows) != 1 {
		t.Fatal("missing metric")
	}
	row := rows[0]
	if row.Count != 1000 || row.Failures != 100 || row.RecentSamples != 256 || row.Last != 999 || row.Max != 999 || row.P50 != 871 || row.P95 != 987 {
		t.Fatalf("unexpected snapshot %+v", row)
	}
	rows[0].Key = "changed"
	if registry.Snapshots()[0].Key != "test" {
		t.Fatal("caller mutated metric storage")
	}
	for n := 0; n < 200; n++ {
		registry.Record(fmt.Sprintf("bounded_%d", n), "events", 1, false)
	}
	if len(registry.Snapshots()) != seriesCapacity {
		t.Fatal("metric key cardinality is unbounded")
	}
}

// 持续快照和多生产者记录互不丢失累计计数，race 检查覆盖热路径锁。
func TestRegistryConcurrentRecordAndSnapshot(t *testing.T) {
	registry := &Registry{}
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 1000; n++ {
				registry.Record("shared", "events", int64(n), false)
				if n%10 == 0 {
					registry.Snapshots()
				}
			}
		}()
	}
	group.Wait()
	rows := registry.Snapshots()
	if len(rows) != 1 || rows[0].Count != 8000 || rows[0].RecentSamples != 256 {
		t.Fatalf("lost metrics: %+v", rows)
	}
}
