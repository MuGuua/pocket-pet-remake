package memory

import (
	"context"

	"pocket-pet-remake/server/internal/module/world"
)

type WorldRepository struct{}

const (
	sceneMinCoord = 0
	sceneMaxCoord = 20
	moveSpeed     = 180
)

func NewWorldRepository() *WorldRepository {
	return &WorldRepository{}
}

func (r *WorldRepository) GetSceneSnapshot(_ context.Context, _ uint64, sceneID uint32, selfPos world.Vec2i) (*world.SceneSnapshot, error) {
	return &world.SceneSnapshot{
		SceneID:      sceneID,
		SelfPos:      selfPos,
		SceneVersion: 1,
		NearbyEntities: []world.Entity{
			{
				EntityID:   90001,
				EntityType: 2,
				Pos:        world.Vec2i{X: 10, Y: 6},
				Dir:        2,
				Speed:      0,
				Name:       "GuideNPC",
			},
		},
	}, nil
}

func (r *WorldRepository) EvaluateMove(_ context.Context, _ uint64, _ uint32, currentPos world.Vec2i, targetPos world.Vec2i) (*world.MoveDecision, error) {
	decision := &world.MoveDecision{
		SceneVersion: 1,
		FromPos:      currentPos,
		ToPos:        currentPos,
		CorrectedPos: currentPos,
		Speed:        moveSpeed,
	}

	if targetPos.X < sceneMinCoord || targetPos.X > sceneMaxCoord || targetPos.Y < sceneMinCoord || targetPos.Y > sceneMaxCoord {
		decision.Accepted = false
		decision.Reason = "target out of bounds"
		return decision, nil
	}

	decision.Accepted = true
	decision.ToPos = targetPos
	decision.CorrectedPos = targetPos
	return decision, nil
}
