package wstransport

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/telemetry"
	"pocket-pet-remake/server/internal/protocol"
)

// 原始客户端样本随心跳上报，数量受限且不影响原心跳响应、鉴权或业务状态。
func TestHeartbeatClientTimingSamplesBounded(t *testing.T) {
	count := func() uint64 {
		for _, row := range telemetry.Default.Snapshots() {
			if row.Key == "client.total.5001" {
				return row.Count
			}
		}
		return 0
	}
	before := count()
	samples := make([]protocol.ClientRequestTiming, 40)
	for index := range samples {
		samples[index] = protocol.ClientRequestTiming{Cmd: protocol.CmdBagListReq, TotalUS: 1010, ResponseUS: 1000, DispatchUS: 10, Succeeded: true}
	}
	service := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	conn := &fakeConn{id: "timing-heartbeat"}
	if _, err := service.Bind(1, conn); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, service)
	packet, err := protocol.NewJSONPacket(protocol.CmdHeartbeatReq, 17, 0, protocol.HeartbeatReq{RequestTimings: samples})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.Handle(conn, encoded); err != nil {
		t.Fatal(err)
	}
	if count() != before+32 || len(conn.packets) != 1 || conn.packets[0].Cmd != protocol.CmdHeartbeatResp || conn.packets[0].Seq != 17 {
		t.Fatal("heartbeat semantics or sample bound changed")
	}
	invalid, _ := protocol.MarshalBody(protocol.HeartbeatReq{RequestTimings: []protocol.ClientRequestTiming{{Cmd: protocol.CmdBagListReq, TotalUS: 30000001}, {Cmd: protocol.CmdBagListReq, TotalUS: 1, ResponseUS: 2}, {Cmd: 0, TotalUS: 1}}})
	recordClientRequestTimings(invalid)
	recordClientRequestTimings([]byte(`{"request_timings":"invalid"}`))
	if count() != before+32 {
		t.Fatal("invalid client timing entered metrics")
	}
}

// Redis 丢失权威状态时必须进入已有断线重连恢复，不能在当前连接上永久忽略移动。
func TestMissingMovementStateClosesForReconnect(t *testing.T) {
	sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	conn := &fakeConn{id: "missing-movement-recovery"}
	if _, err := sessions.Bind(1, conn); err != nil {
		t.Fatal(err)
	}
	states := &movementStateRepoForHandlerTest{}
	service := world.NewService(nil)
	service.SetMovementStateRepository(states)
	handler := NewWorldHandler(sessions, nil, nil, nil, nil, service, nil, nil)
	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 1, 0, protocol.MoveIntentReq{MoveSeq: 1, SceneID: 7, TargetPos: &protocol.Vec2i{X: 2, Y: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.HandleMoveIntent(conn, packet); err != nil {
		t.Fatal(err)
	}
	if !conn.closed || len(conn.packets) != 0 || states.compareCount != 0 {
		t.Fatal("lost movement authority was not isolated for reconnect")
	}
	if _, err := states.Load(context.Background(), 1); err != world.ErrMovementStateNotFound {
		t.Fatal("client position was used to manufacture authority")
	}
}
