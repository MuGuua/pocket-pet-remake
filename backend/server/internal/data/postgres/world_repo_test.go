package postgres

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/world"
)

// TestTimeHousePortalIsOneWay 验证一次性场景只能离开到东路，东路不能通过同一关系返回。
func TestTimeHousePortalIsOneWay(t *testing.T) {
	repository := NewWorldRepository(nil)

	toEastRoad, err := repository.EvaluateTransfer(context.Background(), 1, 7, world.Vec2i{X: 4, Y: 4}, 2, 7001)
	if err != nil {
		t.Fatalf("EvaluateTransfer(time house -> east road) error = %v", err)
	}
	if !toEastRoad.Accepted || toEastRoad.ToSceneID != 2 || toEastRoad.SpawnPos != (world.Vec2i{X: 9, Y: 5}) {
		t.Fatalf("EvaluateTransfer(time house -> east road) = %#v, want accepted east road east entrance", toEastRoad)
	}

	toTimeHouse, err := repository.EvaluateTransfer(context.Background(), 1, 2, world.Vec2i{X: 9, Y: 5}, 7, 7001)
	if err != nil {
		t.Fatalf("EvaluateTransfer(east road -> time house) error = %v", err)
	}
	if toTimeHouse.Accepted {
		t.Fatalf("EvaluateTransfer(east road -> time house) = %#v, want rejected one-way transfer", toTimeHouse)
	}
}
