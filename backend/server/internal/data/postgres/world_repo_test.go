package postgres

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/world"
)

// TestTimeHousePortalIsOneWay 验证一次性场景只能离开到东路，东路不能通过同一关系返回。
func TestTimeHousePortalIsOneWay(t *testing.T) {
	repository := NewWorldRepository(nil)

	toEastRoad, err := repository.EvaluateTransfer(context.Background(), 1, 1, 7, world.Vec2i{X: 4, Y: 4}, 2, 7001)
	if err != nil {
		t.Fatalf("EvaluateTransfer(time house -> east road) error = %v", err)
	}
	if !toEastRoad.Accepted || toEastRoad.ToSceneID != 2 || toEastRoad.SpawnPos != (world.Vec2i{X: 9, Y: 5}) {
		t.Fatalf("EvaluateTransfer(time house -> east road) = %#v, want accepted east road east entrance", toEastRoad)
	}

	toTimeHouse, err := repository.EvaluateTransfer(context.Background(), 1, 1, 2, world.Vec2i{X: 8, Y: 1}, 7, 7001)
	if err != nil {
		t.Fatalf("EvaluateTransfer(east road -> time house) error = %v", err)
	}
	if toTimeHouse.Accepted {
		t.Fatalf("EvaluateTransfer(east road -> time house) = %#v, want rejected one-way transfer", toTimeHouse)
	}
}

// TestPortalSpawnPositionsMatchClientMaps 逐门校验服务端权威出生坐标与 Godot 地图当前
// 调好的进门站位一致；服务端是多人同屏出生点唯一事实来源，客户端不再本地覆盖。
func TestPortalSpawnPositionsMatchClientMaps(t *testing.T) {
	tests := []struct {
		name         string
		fromSceneID  uint32
		toSceneID    uint32
		portalID     uint32
		wantSpawnPos world.Vec2i
	}{
		{name: "罗克斯小屋到东路", fromSceneID: 1, toSceneID: 2, portalID: 1001, wantSpawnPos: world.Vec2i{X: 4, Y: 4}},
		{name: "东路到罗克斯小屋", fromSceneID: 2, toSceneID: 1, portalID: 2001, wantSpawnPos: world.Vec2i{X: 5, Y: 13}},
		{name: "东路到市场", fromSceneID: 2, toSceneID: 3, portalID: 2002, wantSpawnPos: world.Vec2i{X: 12, Y: 11}},
		{name: "市场到东路", fromSceneID: 3, toSceneID: 2, portalID: 3001, wantSpawnPos: world.Vec2i{X: 1, Y: 7}},
		{name: "市场到北路", fromSceneID: 3, toSceneID: 4, portalID: 3002, wantSpawnPos: world.Vec2i{X: 4, Y: 10}},
		{name: "市场到学校", fromSceneID: 3, toSceneID: 5, portalID: 3003, wantSpawnPos: world.Vec2i{X: 12, Y: 3}},
		{name: "北路到市场", fromSceneID: 4, toSceneID: 3, portalID: 4001, wantSpawnPos: world.Vec2i{X: 5, Y: 1}},
		{name: "学校到市场", fromSceneID: 5, toSceneID: 3, portalID: 5001, wantSpawnPos: world.Vec2i{X: 4, Y: 12}},
		{name: "学校到打怪区", fromSceneID: 5, toSceneID: 6, portalID: 5002, wantSpawnPos: world.Vec2i{X: 6, Y: 2}},
		{name: "打怪区到学校", fromSceneID: 6, toSceneID: 5, portalID: 6001, wantSpawnPos: world.Vec2i{X: 6, Y: 11}},
		{name: "时光小屋到东路", fromSceneID: 7, toSceneID: 2, portalID: 7001, wantSpawnPos: world.Vec2i{X: 9, Y: 5}},
		{name: "东路到闪光镇传送区", fromSceneID: 2, toSceneID: 8, portalID: 2003, wantSpawnPos: world.Vec2i{X: 2, Y: 12}},
		{name: "闪光镇传送区到东路", fromSceneID: 8, toSceneID: 2, portalID: 8001, wantSpawnPos: world.Vec2i{X: 9, Y: 5}},
		{name: "闪光镇传送区到闪耀广场", fromSceneID: 8, toSceneID: 9, portalID: 8002, wantSpawnPos: world.Vec2i{X: 20, Y: 12}},
		{name: "闪耀广场到闪光镇传送区", fromSceneID: 9, toSceneID: 8, portalID: 9001, wantSpawnPos: world.Vec2i{X: 6, Y: 9}},
	}

	repository := NewWorldRepository(nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := repository.EvaluateTransfer(context.Background(), 1, 1, test.fromSceneID, world.Vec2i{}, test.toSceneID, test.portalID)
			if err != nil {
				t.Fatalf("EvaluateTransfer() error = %v", err)
			}
			if !decision.Accepted {
				t.Fatalf("EvaluateTransfer() rejected: %s", decision.Reason)
			}
			if decision.ToSceneID != test.toSceneID || decision.SpawnPos != test.wantSpawnPos {
				t.Fatalf("EvaluateTransfer() = scene %d spawn %+v, want scene %d spawn %+v", decision.ToSceneID, decision.SpawnPos, test.toSceneID, test.wantSpawnPos)
			}
		})
	}
}

// TestSceneAccessRejectionReason 验证地图状态和最低等级均由服务端按数据库配置执行。
func TestSceneAccessRejectionReason(t *testing.T) {
	tests := []struct {
		name          string
		playerLevel   uint32
		requiredLevel uint32
		status        uint32
		wantReason    string
	}{
		{name: "低于最低等级", playerLevel: 4, requiredLevel: 5, status: 1, wantReason: world.SceneLevelRestrictedReason},
		{name: "达到最低等级", playerLevel: 5, requiredLevel: 5, status: 1, wantReason: ""},
		{name: "高于最低等级", playerLevel: 6, requiredLevel: 5, status: 1, wantReason: ""},
		{name: "地图已停用", playerLevel: 100, requiredLevel: 1, status: 0, wantReason: "target scene unavailable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotReason := sceneAccessRejectionReason(test.playerLevel, test.requiredLevel, test.status)
			if gotReason != test.wantReason {
				t.Fatalf("sceneAccessRejectionReason(%d, %d, %d) = %q, want %q", test.playerLevel, test.requiredLevel, test.status, gotReason, test.wantReason)
			}
		})
	}
}
