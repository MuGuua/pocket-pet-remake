package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/npcdialogue"
)

// NPCDialogueRepository 负责把数据库里的 NPC 剧情配置映射成服务端运行态需要的仓储读写能力。
type NPCDialogueRepository struct {
	db DBTX
}

// NewNPCDialogueRepository 创建 PostgreSQL 版 NPC 剧情仓储。
func NewNPCDialogueRepository(db DBTX) *NPCDialogueRepository {
	return &NPCDialogueRepository{db: db}
}

const findNPCDialogueByEntityEntryQuery = `
SELECT
  dialogue_id,
  entity_id,
  entry_id,
  dialogue_code,
  title,
  start_node_id,
  version,
  status
FROM npc_dialogue
WHERE entity_id = $1 AND entry_id = $2 AND status = 1
LIMIT 1
`

const existsNPCMenuEntryQuery = `
SELECT 1
FROM npc_menu_entry
WHERE entity_id = $1 AND entry_id = $2
LIMIT 1
`

const findNPCDialogueNodeQuery = `
SELECT
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
  effects_json,
  conditions_json
FROM npc_dialogue_node
WHERE dialogue_id = $1 AND node_id = $2
LIMIT 1
`

const listNPCDialogueOptionsQuery = `
SELECT
  dialogue_id,
  node_id,
  option_id,
  option_text,
  option_format,
  next_node_id,
  conditions_json
FROM npc_dialogue_option
WHERE dialogue_id = $1 AND node_id = $2
ORDER BY sort_order ASC, option_id ASC
`

const upsertNPCDialogueSessionQuery = `
INSERT INTO player_npc_dialogue_session (
  player_id,
  entity_id,
  dialogue_id,
  current_node_id,
  status
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (player_id) DO UPDATE SET
  entity_id = EXCLUDED.entity_id,
  dialogue_id = EXCLUDED.dialogue_id,
  current_node_id = EXCLUDED.current_node_id,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP
`

const findNPCDialogueSessionByPlayerIDQuery = `
SELECT
  player_id,
  entity_id,
  dialogue_id,
  current_node_id,
  status
FROM player_npc_dialogue_session
WHERE player_id = $1
LIMIT 1
`

const deleteNPCDialogueSessionByPlayerIDQuery = `
DELETE FROM player_npc_dialogue_session
WHERE player_id = $1
`

// FindDialogueByEntityEntry 根据 NPC 实体和菜单项定位一段结构化剧情。
func (r *NPCDialogueRepository) FindDialogueByEntityEntry(ctx context.Context, entityID uint64, entryID string) (*npcdialogue.Dialogue, error) {
	var item npcdialogue.Dialogue
	err := r.db.QueryRowContext(ctx, findNPCDialogueByEntityEntryQuery, entityID, entryID).Scan(
		&item.DialogueID,
		&item.EntityID,
		&item.EntryID,
		&item.DialogueCode,
		&item.Title,
		&item.StartNodeID,
		&item.Version,
		&item.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// MenuEntryExists 用于后台保存剧情前确认对应的 NPC 菜单项已经存在，避免写入孤儿剧情配置。
func (r *NPCDialogueRepository) MenuEntryExists(ctx context.Context, entityID uint64, entryID string) (bool, error) {
	var marker int
	err := r.db.QueryRowContext(ctx, existsNPCMenuEntryQuery, entityID, entryID).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return marker == 1, nil
}

// FindNode 读取剧情中的单个节点，供服务端判断当前该展示台词还是播放动作动画。
func (r *NPCDialogueRepository) FindNode(ctx context.Context, dialogueID int64, nodeID string) (*npcdialogue.DialogueNode, error) {
	var item npcdialogue.DialogueNode
	var clientAnimationBlock int16
	var effectsJSON []byte
	var conditionsJSON []byte
	err := r.db.QueryRowContext(ctx, findNPCDialogueNodeQuery, dialogueID, nodeID).Scan(
		&item.DialogueID,
		&item.NodeID,
		&item.NodeType,
		&item.Speaker,
		&item.Content,
		&item.ContentFormat,
		&item.PortraitKey,
		&item.NextNodeID,
		&item.ClientAnimationKey,
		&clientAnimationBlock,
		&effectsJSON,
		&conditionsJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.ClientAnimationBlock = clientAnimationBlock == 1
	item.EffectsJSON = effectsJSON
	item.ConditionsJSON = conditionsJSON
	return &item, nil
}

// ListOptions 返回 choice 节点下可展示的全部选项，客户端只负责逐条渲染按钮。
func (r *NPCDialogueRepository) ListOptions(ctx context.Context, dialogueID int64, nodeID string) ([]npcdialogue.DialogueOption, error) {
	rows, err := r.db.QueryContext(ctx, listNPCDialogueOptionsQuery, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]npcdialogue.DialogueOption, 0)
	for rows.Next() {
		var item npcdialogue.DialogueOption
		var conditionsJSON []byte
		if err := rows.Scan(&item.DialogueID, &item.NodeID, &item.OptionID, &item.OptionText, &item.OptionFormat, &item.NextNodeID, &conditionsJSON); err != nil {
			return nil, err
		}
		item.ConditionsJSON = conditionsJSON
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// UpsertSession 把玩家当前剧情推进位置持久化，避免客户端伪造跳转路径。
func (r *NPCDialogueRepository) UpsertSession(ctx context.Context, session npcdialogue.DialogueSession) error {
	_, err := r.db.ExecContext(ctx, upsertNPCDialogueSessionQuery, session.PlayerID, session.EntityID, session.DialogueID, session.CurrentNodeID, session.Status)
	return err
}

// FindSessionByPlayerID 返回玩家当前活跃剧情会话，用于继续或选择分支时做权威校验。
func (r *NPCDialogueRepository) FindSessionByPlayerID(ctx context.Context, playerID uint64) (*npcdialogue.DialogueSession, error) {
	var item npcdialogue.DialogueSession
	err := r.db.QueryRowContext(ctx, findNPCDialogueSessionByPlayerIDQuery, playerID).Scan(
		&item.PlayerID,
		&item.EntityID,
		&item.DialogueID,
		&item.CurrentNodeID,
		&item.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteSession 在剧情结束后移除玩家会话，避免旧节点被客户端重复利用。
func (r *NPCDialogueRepository) DeleteSession(ctx context.Context, playerID uint64) error {
	_, err := r.db.ExecContext(ctx, deleteNPCDialogueSessionByPlayerIDQuery, playerID)
	return err
}
