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
		{name: "闪耀广场到闪光南路", fromSceneID: 9, toSceneID: 20, portalID: 9002, wantSpawnPos: world.Vec2i{X: 9, Y: 2}},
		{name: "闪光南路到闪耀广场", fromSceneID: 20, toSceneID: 9, portalID: 20001, wantSpawnPos: world.Vec2i{X: 12, Y: 12}},
		{name: "闪耀广场到商业区", fromSceneID: 9, toSceneID: 16, portalID: 9003, wantSpawnPos: world.Vec2i{X: 13, Y: 9}},
		{name: "商业区到闪耀广场", fromSceneID: 16, toSceneID: 9, portalID: 16001, wantSpawnPos: world.Vec2i{X: 2, Y: 8}},
		{name: "闪耀广场到办公区", fromSceneID: 9, toSceneID: 15, portalID: 9004, wantSpawnPos: world.Vec2i{X: 5, Y: 16}},
		{name: "办公区到闪耀广场", fromSceneID: 15, toSceneID: 9, portalID: 15004, wantSpawnPos: world.Vec2i{X: 5, Y: 7}},
		{name: "闪耀广场到宠物学校", fromSceneID: 9, toSceneID: 10, portalID: 9005, wantSpawnPos: world.Vec2i{X: 3, Y: 10}},
		{name: "宠物学校到闪耀广场", fromSceneID: 10, toSceneID: 9, portalID: 10001, wantSpawnPos: world.Vec2i{X: 16, Y: 6}},
		{name: "宠物学校到阿尔房间", fromSceneID: 10, toSceneID: 14, portalID: 10002, wantSpawnPos: world.Vec2i{X: 5, Y: 10}},
		{name: "阿尔房间到宠物学校", fromSceneID: 14, toSceneID: 10, portalID: 14001, wantSpawnPos: world.Vec2i{X: 6, Y: 10}},
		{name: "闪耀广场到报名区", fromSceneID: 9, toSceneID: 17, portalID: 9006, wantSpawnPos: world.Vec2i{X: 1, Y: 7}},
		{name: "报名区到闪耀广场", fromSceneID: 17, toSceneID: 9, portalID: 17001, wantSpawnPos: world.Vec2i{X: 23, Y: 8}},
		{name: "报名区到准备区", fromSceneID: 17, toSceneID: 18, portalID: 17002, wantSpawnPos: world.Vec2i{X: 2, Y: 7}},
		{name: "报名区到比武区入口", fromSceneID: 17, toSceneID: 18, portalID: 17003, wantSpawnPos: world.Vec2i{X: 7, Y: 5}},
		{name: "准备区到报名区", fromSceneID: 18, toSceneID: 17, portalID: 18001, wantSpawnPos: world.Vec2i{X: 9, Y: 6}},
		{name: "办公区到冰雪梦境", fromSceneID: 15, toSceneID: 11, portalID: 15001, wantSpawnPos: world.Vec2i{X: 9, Y: 10}},
		{name: "冰雪梦境到办公区", fromSceneID: 11, toSceneID: 15, portalID: 11001, wantSpawnPos: world.Vec2i{X: 1, Y: 7}},
		{name: "办公区到灰烬梦境", fromSceneID: 15, toSceneID: 12, portalID: 15002, wantSpawnPos: world.Vec2i{X: 4, Y: 10}},
		{name: "灰烬梦境到办公区", fromSceneID: 12, toSceneID: 15, portalID: 12001, wantSpawnPos: world.Vec2i{X: 5, Y: 2}},
		{name: "办公区到翡翠梦境", fromSceneID: 15, toSceneID: 13, portalID: 15003, wantSpawnPos: world.Vec2i{X: 1, Y: 9}},
		{name: "翡翠梦境到办公区", fromSceneID: 13, toSceneID: 15, portalID: 13001, wantSpawnPos: world.Vec2i{X: 9, Y: 7}},
		{name: "闪光南路到家族会馆", fromSceneID: 20, toSceneID: 19, portalID: 20002, wantSpawnPos: world.Vec2i{X: 12, Y: 10}},
		{name: "家族会馆到闪光南路", fromSceneID: 19, toSceneID: 20, portalID: 19001, wantSpawnPos: world.Vec2i{X: 4, Y: 8}},
		{name: "闪光南路到五彩湖", fromSceneID: 20, toSceneID: 21, portalID: 20003, wantSpawnPos: world.Vec2i{X: 1, Y: 5}},
		{name: "五彩湖到闪光南路", fromSceneID: 21, toSceneID: 20, portalID: 21001, wantSpawnPos: world.Vec2i{X: 11, Y: 8}},
		{name: "闪光南路到闪光海岸", fromSceneID: 20, toSceneID: 23, portalID: 20004, wantSpawnPos: world.Vec2i{X: 7, Y: 2}},
		{name: "闪光海岸到闪光南路", fromSceneID: 23, toSceneID: 20, portalID: 23001, wantSpawnPos: world.Vec2i{X: 8, Y: 11}},
		{name: "闪光海岸到沼泽地", fromSceneID: 23, toSceneID: 22, portalID: 23002, wantSpawnPos: world.Vec2i{X: 9, Y: 9}},
		{name: "闪光海岸到海道", fromSceneID: 23, toSceneID: 26, portalID: 23003, wantSpawnPos: world.Vec2i{X: 6, Y: 2}},
		{name: "海道到闪光海岸", fromSceneID: 26, toSceneID: 23, portalID: 26001, wantSpawnPos: world.Vec2i{X: 8, Y: 11}},
		{name: "海道到精灵大厅", fromSceneID: 26, toSceneID: 25, portalID: 26002, wantSpawnPos: world.Vec2i{X: 2, Y: 8}},
		{name: "精灵大厅到海道", fromSceneID: 25, toSceneID: 26, portalID: 25001, wantSpawnPos: world.Vec2i{X: 10, Y: 8}},
		{name: "沼泽地到闪光海岸", fromSceneID: 22, toSceneID: 23, portalID: 22001, wantSpawnPos: world.Vec2i{X: 2, Y: 6}},
		{name: "沼泽地到尘泥之地", fromSceneID: 22, toSceneID: 24, portalID: 22002, wantSpawnPos: world.Vec2i{X: 5, Y: 2}},
		{name: "尘泥之地到沼泽地", fromSceneID: 24, toSceneID: 22, portalID: 24001, wantSpawnPos: world.Vec2i{X: 4, Y: 10}},
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

// TestShiningPlainScenesRegistered 验证闪光平原所有已落地地图均进入服务端权威场景表。
func TestShiningPlainScenesRegistered(t *testing.T) {
	for sceneID := uint32(9); sceneID <= 26; sceneID++ {
		if _, ok := worldScenes[sceneID]; !ok {
			t.Fatalf("worldScenes missing shining plain scene_id=%d", sceneID)
		}
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
