package npc

import "context"

type Repository interface {
	ListMenuEntriesByEntityID(ctx context.Context, entityID uint64) ([]MenuEntry, error)
	FindActionResult(ctx context.Context, entityID uint64, entryID string) (*ActionResult, error)
}
