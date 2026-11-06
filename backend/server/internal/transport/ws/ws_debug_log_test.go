package wstransport

import (
	"bytes"
	"log"
	"pocket-pet-remake/server/internal/protocol"
	"strings"
	"testing"
	"time"
)

// 诊断只输出慢查询元数据，正常查询和高频移动不产生逐包日志。
func TestSlowRequestTimingLogging(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cmd     uint16
		elapsed time.Duration
		wantLog bool
	}{
		{"fast query", protocol.CmdPetListReq, 199 * time.Millisecond, false},
		{"slow query", protocol.CmdPetListReq, 200 * time.Millisecond, true},
		{"movement", protocol.CmdMoveIntentReq, time.Second, false},
		{"heartbeat", protocol.CmdHeartbeatReq, time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.New(&output, "", 0)
			logSlowWSRequest(logger, "timing-conn", tc.cmd, 72, 26, tc.elapsed)
			if !tc.wantLog {
				if output.Len() != 0 {
					t.Fatalf("unexpected timing log: %s", output.String())
				}
				return
			}
			for _, field := range []string{"[ws-timing]", "conn_id=timing-conn", "seq=72", "request_bytes=26", "handler_us=200000"} {
				if !strings.Contains(output.String(), field) {
					t.Fatalf("missing %s in %s", field, output.String())
				}
			}
		})
	}
}
