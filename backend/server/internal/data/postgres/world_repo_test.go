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
