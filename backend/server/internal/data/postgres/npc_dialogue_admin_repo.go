package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/npcdialogue"
)

const adminNPCDialogueListBaseQuery = `
SELECT
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status,
  created_at,
  updated_at
FROM npc_dialogue
`

const adminNPCDialogueDetailQuery = `
SELECT
  dialogue_id,
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status,
  created_at,
  updated_at
FROM npc_dialogue
WHERE entity_id = $1 AND entry_id = $2
LIMIT 1
`

const adminNPCDialogueNodesQuery = `
SELECT
  node_id,
  node_type,
  speaker,
  content,
  content_format,
  portrait_key,
  next_node_id,
  client_animation_key,
  client_animation_block,
  sort_order,
  conditions_json,
  effects_json
FROM npc_dialogue_node
WHERE dialogue_id = $1
ORDER BY sort_order ASC, node_id ASC
`

const adminNPCDialogueOptionsQuery = `
SELECT
  node_id,
  option_id,
  option_text,
  option_format,
  next_node_id,
  sort_order,
  conditions_json
FROM npc_dialogue_option
WHERE dialogue_id = $1
ORDER BY node_id ASC, sort_order ASC, option_id ASC
`

const insertAdminNPCDialogueQuery = `
INSERT INTO npc_dialogue (
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING dialogue_id
`

const updateAdminNPCDialogueQuery = `
UPDATE npc_dialogue
SET entity_id = $3,
    dialogue_code = $4,
    title = $5,
    start_node_id = $6,
    version = $7,
    status = $8,
    updated_at = CURRENT_TIMESTAMP
WHERE entity_id = $1 AND entry_id = $2
RETURNING dialogue_id
`

const deleteAdminNPCDialogueQuery = `
DELETE FROM npc_dialogue
WHERE entity_id = $1 AND entry_id = $2
`

const deleteAdminNPCDialogueNodesQuery = `
DELETE FROM npc_dialogue_node
WHERE dialogue_id = $1
`

const insertAdminNPCDialogueNodeQuery = `
INSERT INTO npc_dialogue_node (
  dialogue_id,
  node_id,
  node_type,
  speaker,
  content,
  content_format,
  portrait_key,
  next_node_id,
  client_animation_key,
  client_animation_block,
  sort_order,
  conditions_json,
  effects_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const insertAdminNPCDialogueOptionQuery = `
INSERT INTO npc_dialogue_option (
  dialogue_id,
  node_id,
  option_id,
  option_text,
  option_format,
  next_node_id,
  sort_order,
  conditions_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

func (r *NPCDialogueRepository) ListDialoguesForAdmin(ctx context.Context, query npcdialogue.AdminDialogueListQuery) (*npcdialogue.AdminDialogueList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.EntityID > 0 {
		conditions = append(conditions, "entity_id = "+nextArg(query.EntityID))
	}
	if query.EntryID != "" {
		conditions = append(conditions, "entry_id ILIKE "+nextArg("%"+query.EntryID+"%"))
	}
	if query.Status != nil {
		conditions = append(conditions, "status = "+nextArg(*query.Status))
	}
	whereClause := joinWhere(conditions)
	countQuery := `SELECT COUNT(1) FROM npc_dialogue ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminNPCDialogueListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY entity_id ASC, entry_id ASC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]npcdialogue.AdminDialogueSummary, 0)
	for rows.Next() {
		var item npcdialogue.AdminDialogueSummary
		if err := rows.Scan(&item.EntityID, &item.EntryID, &item.DialogueCode, &item.Title, &item.StartNodeID, &item.Version, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &npcdialogue.AdminDialogueList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *NPCDialogueRepository) FindDialogueDetailForAdmin(ctx context.Context, entityID uint64, entryID string) (*npcdialogue.AdminDialogueDetail, error) {
	var dialogueID int64
	var item npcdialogue.AdminDialogueDetail
	err := r.db.QueryRowContext(ctx, adminNPCDialogueDetailQuery, entityID, entryID).Scan(
		&dialogueID,
		&item.EntityID,
		&item.EntryID,
		&item.DialogueCode,
		&item.Title,
		&item.StartNodeID,
		&item.Version,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	nodes, err := r.loadAdminDialogueNodes(ctx, dialogueID)
	if err != nil {
		return nil, err
	}
	item.Nodes = nodes
	return &item, nil
}

func (r *NPCDialogueRepository) CreateDialogueForAdmin(ctx context.Context, input npcdialogue.AdminCreateDialogueInput) (*npcdialogue.AdminDialogueDetail, error) {
	var dialogueID int64
	err := r.db.QueryRowContext(ctx, insertAdminNPCDialogueQuery, input.EntityID, input.EntryID, input.DialogueCode, input.Title, input.StartNodeID, input.Version, input.Status).Scan(&dialogueID)
	if err != nil {
		if isNPCDialogueUniqueViolation(err) {
			return nil, npcdialogue.ErrAdminDialogueConflict
		}
		return nil, err
	}
	if err := r.replaceDialogueNodes(ctx, dialogueID, input.Nodes); err != nil {
		return nil, err
	}
	return r.FindDialogueDetailForAdmin(ctx, input.EntityID, input.EntryID)
}

func (r *NPCDialogueRepository) UpdateDialogueForAdmin(ctx context.Context, entityID uint64, entryID string, input npcdialogue.AdminUpdateDialogueInput) (*npcdialogue.AdminDialogueDetail, error) {
	var dialogueID int64
	err := r.db.QueryRowContext(ctx, updateAdminNPCDialogueQuery, entityID, entryID, input.EntityID, input.DialogueCode, input.Title, input.StartNodeID, input.Version, input.Status).Scan(&dialogueID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		if isNPCDialogueUniqueViolation(err) {
			return nil, npcdialogue.ErrAdminDialogueConflict
		}
		return nil, err
	}
	if err := r.replaceDialogueNodes(ctx, dialogueID, input.Nodes); err != nil {
		return nil, err
	}
	return r.FindDialogueDetailForAdmin(ctx, input.EntityID, entryID)
}

func (r *NPCDialogueRepository) DeleteDialogueForAdmin(ctx context.Context, entityID uint64, entryID string) error {
	result, err := r.db.ExecContext(ctx, deleteAdminNPCDialogueQuery, entityID, entryID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return npcdialogue.ErrDialogueNotFound
	}
	return nil
}

func (r *NPCDialogueRepository) loadAdminDialogueNodes(ctx context.Context, dialogueID int64) ([]npcdialogue.AdminDialogueNodeDetail, error) {
	rows, err := r.db.QueryContext(ctx, adminNPCDialogueNodesQuery, dialogueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := make([]npcdialogue.AdminDialogueNodeDetail, 0)
	nodeIndexByID := map[string]int{}
	for rows.Next() {
		var item npcdialogue.AdminDialogueNodeDetail
		var animationBlock int16
		var conditionsJSON []byte
		var effectsJSON []byte
		if err := rows.Scan(&item.NodeID, &item.NodeType, &item.Speaker, &item.Content, &item.ContentFormat, &item.PortraitKey, &item.NextNodeID, &item.ClientAnimationKey, &animationBlock, &item.SortOrder, &conditionsJSON, &effectsJSON); err != nil {
			return nil, err
		}
		item.ClientAnimationBlock = animationBlock == 1
		item.Conditions = npcdialogue.DecodeAdminConditionsJSON(conditionsJSON)
		item.Effects = npcdialogue.DecodeAdminEffectsJSON(effectsJSON)
		item.Options = []npcdialogue.AdminDialogueOptionDetail{}
		nodeIndexByID[item.NodeID] = len(nodes)
		nodes = append(nodes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	optionRows, err := r.db.QueryContext(ctx, adminNPCDialogueOptionsQuery, dialogueID)
	if err != nil {
		return nil, err
	}
	defer optionRows.Close()
	for optionRows.Next() {
		var nodeID string
		var option npcdialogue.AdminDialogueOptionDetail
		var conditionsJSON []byte
		if err := optionRows.Scan(&nodeID, &option.OptionID, &option.OptionText, &option.OptionFormat, &option.NextNodeID, &option.SortOrder, &conditionsJSON); err != nil {
			return nil, err
		}
		option.Conditions = npcdialogue.DecodeAdminConditionsJSON(conditionsJSON)
		index, ok := nodeIndexByID[nodeID]
		if !ok {
			continue
		}
		nodes[index].Options = append(nodes[index].Options, option)
	}
	if err := optionRows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *NPCDialogueRepository) replaceDialogueNodes(ctx context.Context, dialogueID int64, nodes []npcdialogue.AdminDialogueNodeInput) error {
	if _, err := r.db.ExecContext(ctx, deleteAdminNPCDialogueNodesQuery, dialogueID); err != nil {
		return err
	}
	sortedNodes := make([]npcdialogue.AdminDialogueNodeInput, 0, len(nodes))
	sortedNodes = append(sortedNodes, nodes...)
	sort.Slice(sortedNodes, func(i int, j int) bool {
		if sortedNodes[i].SortOrder != sortedNodes[j].SortOrder {
			return sortedNodes[i].SortOrder < sortedNodes[j].SortOrder
		}
		return sortedNodes[i].NodeID < sortedNodes[j].NodeID
	})
	for _, node := range sortedNodes {
		animationBlock := int16(0)
		if node.ClientAnimationBlock {
			animationBlock = 1
		}
		if _, err := r.db.ExecContext(ctx, insertAdminNPCDialogueNodeQuery, dialogueID, node.NodeID, node.NodeType, node.Speaker, node.Content, node.ContentFormat, node.PortraitKey, node.NextNodeID, node.ClientAnimationKey, animationBlock, node.SortOrder, npcdialogue.EncodeAdminConditionsJSON(node.Conditions), npcdialogue.EncodeAdminEffectsJSON(node.Effects)); err != nil {
			return err
		}
		sortedOptions := make([]npcdialogue.AdminDialogueOptionInput, 0, len(node.Options))
		sortedOptions = append(sortedOptions, node.Options...)
		sort.Slice(sortedOptions, func(i int, j int) bool {
			if sortedOptions[i].SortOrder != sortedOptions[j].SortOrder {
				return sortedOptions[i].SortOrder < sortedOptions[j].SortOrder
			}
			return sortedOptions[i].OptionID < sortedOptions[j].OptionID
		})
		for _, option := range sortedOptions {
			if _, err := r.db.ExecContext(ctx, insertAdminNPCDialogueOptionQuery, dialogueID, node.NodeID, option.OptionID, option.OptionText, option.OptionFormat, option.NextNodeID, option.SortOrder, npcdialogue.EncodeAdminConditionsJSON(option.Conditions)); err != nil {
				return err
			}
		}
	}
	return nil
}

func isNPCDialogueUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
