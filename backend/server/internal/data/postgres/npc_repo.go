package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/npc"
)

type NPCRepository struct {
	db DBTX
}

func NewNPCRepository(db DBTX) *NPCRepository {
	return &NPCRepository{db: db}
}

const listNPCMenuEntriesByEntityIDQuery = `
SELECT
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  action_result_type,
  action_notice
FROM npc_menu_entry
WHERE entity_id = $1 AND status = 1
ORDER BY priority DESC, sort_order ASC, entry_id ASC
`

const findNPCActionResultQuery = `
SELECT
  entity_id,
  entry_id,
  action_result_type,
  action_notice
FROM npc_menu_entry
WHERE entity_id = $1 AND entry_id = $2 AND status = 1
LIMIT 1
`

func (r *NPCRepository) ListMenuEntriesByEntityID(ctx context.Context, entityID uint64) ([]npc.MenuEntry, error) {
	rows, err := r.db.QueryContext(ctx, listNPCMenuEntriesByEntityIDQuery, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []npc.MenuEntry{}
	for rows.Next() {
		var value npc.MenuEntry
		if err := rows.Scan(
			&value.EntityID,
			&value.EntryID,
			&value.EntryType,
			&value.Title,
			&value.Subtitle,
			&value.State,
			&value.Priority,
			&value.ActionResultType,
			&value.ActionNotice,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *NPCRepository) FindActionResult(ctx context.Context, entityID uint64, entryID string) (*npc.ActionResult, error) {
	var value npc.ActionResult
	err := r.db.QueryRowContext(ctx, findNPCActionResultQuery, entityID, entryID).Scan(
		&value.EntityID,
		&value.EntryID,
		&value.ResultType,
		&value.Notice,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}
