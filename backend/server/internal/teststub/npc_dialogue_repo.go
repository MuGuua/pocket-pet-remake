package teststub

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/npcdialogue"
)

// NewNPCDialogueRepository 返回测试环境使用的内存版 NPC 剧情仓储，默认预置一段市场理萌剧情。
func NewNPCDialogueRepository() *NPCDialogueRepository {
	now := time.Now()
	return &NPCDialogueRepository{
		dialoguesByEntityEntry: map[string]npcdialogue.Dialogue{
			"93001:dialog_market_intro": {DialogueID: 1, EntityID: 93001, EntryID: "dialog_market_intro", DialogueCode: "radiant_market_intro", Title: "市场理萌开场", StartNodeID: "start", Version: 1, Status: 1},
		},
		menuEntriesByEntityKey: map[string]struct{}{
			"93001:dialog_market_intro": {},
			"93001:dialog_branch_test":  {},
		},
		nodesByDialogueNode: map[string]npcdialogue.DialogueNode{
			"1:start":      {DialogueID: 1, NodeID: "start", NodeType: npcdialogue.NodeTypeLine, Speaker: "市场理萌", Content: "你先稍等一下，我把前面的货箱挪开。", ContentFormat: "plain", PortraitKey: "npc_limeng_normal", NextNodeID: "move_aside"},
			"1:move_aside": {DialogueID: 1, NodeID: "move_aside", NodeType: npcdialogue.NodeTypeAction, ClientAnimationKey: "market_limeng_step_aside", ClientAnimationBlock: true, NextNodeID: "after_move"},
			"1:after_move": {DialogueID: 1, NodeID: "after_move", NodeType: npcdialogue.NodeTypeChoice, Speaker: "市场理萌", Content: "好啦，现在你是想先听听新鲜事，还是先逛逛？", ContentFormat: "plain", PortraitKey: "npc_limeng_smile"},
			"1:news":       {DialogueID: 1, NodeID: "news", NodeType: npcdialogue.NodeTypeLine, Speaker: "市场理萌", Content: "最近市场来了不少新货，记得多看看。", ContentFormat: "plain", PortraitKey: "npc_limeng_smile", NextNodeID: "end"},
			"1:end":        {DialogueID: 1, NodeID: "end", NodeType: npcdialogue.NodeTypeEnd},
		},
		optionsByDialogueNode: map[string][]npcdialogue.DialogueOption{
			"1:after_move": {
				{DialogueID: 1, NodeID: "after_move", OptionID: "news", OptionText: "听听新鲜事", OptionFormat: "plain", NextNodeID: "news"},
				{DialogueID: 1, NodeID: "after_move", OptionID: "leave", OptionText: "先逛逛", OptionFormat: "plain", NextNodeID: "end"},
			},
		},
		nodeSortOrderByKey: map[string]uint32{
			"1:start":      1,
			"1:move_aside": 2,
			"1:after_move": 3,
			"1:news":       4,
			"1:end":        5,
		},
		optionSortOrderByKey: map[string]uint32{
			"1:after_move:news":  1,
			"1:after_move:leave": 2,
		},
		sessionsByPlayerID: map[uint64]npcdialogue.DialogueSession{},
		createdAtByEntityEntry: map[string]time.Time{
			"93001:dialog_market_intro": now,
		},
		updatedAtByEntityEntry: map[string]time.Time{
			"93001:dialog_market_intro": now,
		},
		nextDialogueID: 1,
	}
}

// NPCDialogueRepository 使用 map 保存测试剧情配置和会话推进位置。
type NPCDialogueRepository struct {
	mu                     sync.Mutex
	dialoguesByEntityEntry map[string]npcdialogue.Dialogue
	menuEntriesByEntityKey map[string]struct{}
	nodesByDialogueNode    map[string]npcdialogue.DialogueNode
	optionsByDialogueNode  map[string][]npcdialogue.DialogueOption
	nodeSortOrderByKey     map[string]uint32
	optionSortOrderByKey   map[string]uint32
	sessionsByPlayerID     map[uint64]npcdialogue.DialogueSession
	createdAtByEntityEntry map[string]time.Time
	updatedAtByEntityEntry map[string]time.Time
	nextDialogueID         int64
}

func (r *NPCDialogueRepository) MenuEntryExists(_ context.Context, entityID uint64, entryID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.menuEntriesByEntityKey[npcDialogueEntityEntryKey(entityID, entryID)]
	return ok, nil
}

func (r *NPCDialogueRepository) ListDialoguesForAdmin(_ context.Context, query npcdialogue.AdminDialogueListQuery) (*npcdialogue.AdminDialogueList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	normalized := query.Normalize()
	items := make([]npcdialogue.AdminDialogueSummary, 0)
	for key, dialogue := range r.dialoguesByEntityEntry {
		if normalized.EntityID > 0 && dialogue.EntityID != normalized.EntityID {
			continue
		}
		if normalized.EntryID != "" && dialogue.EntryID != normalized.EntryID {
			continue
		}
		if normalized.Status != nil && uint32(dialogue.Status) != *normalized.Status {
			continue
		}
		items = append(items, npcdialogue.AdminDialogueSummary{
			EntityID:     dialogue.EntityID,
			EntryID:      dialogue.EntryID,
			DialogueCode: dialogue.DialogueCode,
			Title:        dialogue.Title,
			StartNodeID:  dialogue.StartNodeID,
			Version:      dialogue.Version,
			Status:       uint32(dialogue.Status),
			CreatedAt:    r.createdAtByEntityEntry[key],
			UpdatedAt:    r.updatedAtByEntityEntry[key],
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		if items[i].EntityID != items[j].EntityID {
			return items[i].EntityID < items[j].EntityID
		}
		return items[i].EntryID < items[j].EntryID
	})

	start := int((normalized.Page - 1) * normalized.PageSize)
	if start > len(items) {
		start = len(items)
	}
	end := start + int(normalized.PageSize)
	if end > len(items) {
		end = len(items)
	}
	return &npcdialogue.AdminDialogueList{
		Items:    items[start:end],
		Total:    uint64(len(items)),
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}, nil
}

func (r *NPCDialogueRepository) FindDialogueDetailForAdmin(_ context.Context, entityID uint64, entryID string) (*npcdialogue.AdminDialogueDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := npcDialogueEntityEntryKey(entityID, entryID)
	dialogue, ok := r.dialoguesByEntityEntry[key]
	if !ok {
		return nil, nil
	}
	nodes := r.buildAdminNodesLocked(dialogue.DialogueID)
	return &npcdialogue.AdminDialogueDetail{
		EntityID:     dialogue.EntityID,
		EntryID:      dialogue.EntryID,
		DialogueCode: dialogue.DialogueCode,
		Title:        dialogue.Title,
		StartNodeID:  dialogue.StartNodeID,
		Version:      dialogue.Version,
		Status:       uint32(dialogue.Status),
		CreatedAt:    r.createdAtByEntityEntry[key],
		UpdatedAt:    r.updatedAtByEntityEntry[key],
		Nodes:        nodes,
	}, nil
}

func (r *NPCDialogueRepository) CreateDialogueForAdmin(_ context.Context, input npcdialogue.AdminCreateDialogueInput) (*npcdialogue.AdminDialogueDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := npcDialogueEntityEntryKey(input.EntityID, input.EntryID)
	if _, exists := r.dialoguesByEntityEntry[key]; exists {
		return nil, npcdialogue.ErrAdminDialogueConflict
	}
	r.nextDialogueID++
	now := time.Now()
	dialogue := npcdialogue.Dialogue{
		DialogueID:   r.nextDialogueID,
		EntityID:     input.EntityID,
		EntryID:      input.EntryID,
		DialogueCode: input.DialogueCode,
		Title:        input.Title,
		StartNodeID:  input.StartNodeID,
		Version:      input.Version,
		Status:       int16(input.Status),
	}
	r.dialoguesByEntityEntry[key] = dialogue
	r.createdAtByEntityEntry[key] = now
	r.updatedAtByEntityEntry[key] = now
	r.replaceDialogueNodesLocked(dialogue.DialogueID, input.Nodes)
	return &npcdialogue.AdminDialogueDetail{
		EntityID:     dialogue.EntityID,
		EntryID:      dialogue.EntryID,
		DialogueCode: dialogue.DialogueCode,
		Title:        dialogue.Title,
		StartNodeID:  dialogue.StartNodeID,
		Version:      dialogue.Version,
		Status:       input.Status,
		CreatedAt:    now,
		UpdatedAt:    now,
		Nodes:        r.buildAdminNodesLocked(dialogue.DialogueID),
	}, nil
}

func (r *NPCDialogueRepository) UpdateDialogueForAdmin(_ context.Context, entityID uint64, entryID string, input npcdialogue.AdminUpdateDialogueInput) (*npcdialogue.AdminDialogueDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldKey := npcDialogueEntityEntryKey(entityID, entryID)
	current, ok := r.dialoguesByEntityEntry[oldKey]
	if !ok {
		return nil, nil
	}
	newKey := npcDialogueEntityEntryKey(input.EntityID, entryID)
	if oldKey != newKey {
		if _, exists := r.dialoguesByEntityEntry[newKey]; exists {
			return nil, npcdialogue.ErrAdminDialogueConflict
		}
	}
	delete(r.dialoguesByEntityEntry, oldKey)
	createdAt := r.createdAtByEntityEntry[oldKey]
	delete(r.createdAtByEntityEntry, oldKey)
	delete(r.updatedAtByEntityEntry, oldKey)

	now := time.Now()
	current.EntityID = input.EntityID
	current.DialogueCode = input.DialogueCode
	current.Title = input.Title
	current.StartNodeID = input.StartNodeID
	current.Version = input.Version
	current.Status = int16(input.Status)
	r.dialoguesByEntityEntry[newKey] = current
	r.createdAtByEntityEntry[newKey] = createdAt
	r.updatedAtByEntityEntry[newKey] = now
	r.replaceDialogueNodesLocked(current.DialogueID, input.Nodes)
	return &npcdialogue.AdminDialogueDetail{
		EntityID:     current.EntityID,
		EntryID:      current.EntryID,
		DialogueCode: current.DialogueCode,
		Title:        current.Title,
		StartNodeID:  current.StartNodeID,
		Version:      current.Version,
		Status:       input.Status,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		Nodes:        r.buildAdminNodesLocked(current.DialogueID),
	}, nil
}

func (r *NPCDialogueRepository) DeleteDialogueForAdmin(_ context.Context, entityID uint64, entryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := npcDialogueEntityEntryKey(entityID, entryID)
	dialogue, ok := r.dialoguesByEntityEntry[key]
	if !ok {
		return npcdialogue.ErrDialogueNotFound
	}
	delete(r.dialoguesByEntityEntry, key)
	delete(r.createdAtByEntityEntry, key)
	delete(r.updatedAtByEntityEntry, key)
	r.deleteDialogueNodesLocked(dialogue.DialogueID)
	return nil
}

func (r *NPCDialogueRepository) FindDialogueByEntityEntry(_ context.Context, entityID uint64, entryID string) (*npcdialogue.Dialogue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.dialoguesByEntityEntry[npcDialogueEntityEntryKey(entityID, entryID)]
	if !ok {
		return nil, nil
	}
	copied := item
	return &copied, nil
}

func (r *NPCDialogueRepository) FindNode(_ context.Context, dialogueID int64, nodeID string) (*npcdialogue.DialogueNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.nodesByDialogueNode[npcDialogueNodeKey(dialogueID, nodeID)]
	if !ok {
		return nil, nil
	}
	copied := item
	return &copied, nil
}

func (r *NPCDialogueRepository) ListOptions(_ context.Context, dialogueID int64, nodeID string) ([]npcdialogue.DialogueOption, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.optionsByDialogueNode[npcDialogueNodeKey(dialogueID, nodeID)]
	result := make([]npcdialogue.DialogueOption, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result, nil
}

func (r *NPCDialogueRepository) UpsertSession(_ context.Context, session npcdialogue.DialogueSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionsByPlayerID[session.PlayerID] = session
	return nil
}

func (r *NPCDialogueRepository) FindSessionByPlayerID(_ context.Context, playerID uint64) (*npcdialogue.DialogueSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.sessionsByPlayerID[playerID]
	if !ok {
		return nil, nil
	}
	copied := item
	return &copied, nil
}

func (r *NPCDialogueRepository) DeleteSession(_ context.Context, playerID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessionsByPlayerID, playerID)
	return nil
}

func (r *NPCDialogueRepository) buildAdminNodesLocked(dialogueID int64) []npcdialogue.AdminDialogueNodeDetail {
	nodes := make([]npcdialogue.AdminDialogueNodeDetail, 0)
	for _, node := range r.nodesByDialogueNode {
		if node.DialogueID != dialogueID {
			continue
		}
		options := make([]npcdialogue.AdminDialogueOptionDetail, 0)
		for _, option := range r.optionsByDialogueNode[npcDialogueNodeKey(dialogueID, node.NodeID)] {
			options = append(options, npcdialogue.AdminDialogueOptionDetail{
				OptionID:     option.OptionID,
				OptionText:   option.OptionText,
				OptionFormat: option.OptionFormat,
				NextNodeID:   option.NextNodeID,
				SortOrder:    r.optionSortOrderByKey[npcDialogueOptionKey(dialogueID, node.NodeID, option.OptionID)],
				Conditions:   npcdialogue.DecodeAdminConditionsJSON(option.ConditionsJSON),
			})
		}
		sort.Slice(options, func(i int, j int) bool {
			if options[i].SortOrder != options[j].SortOrder {
				return options[i].SortOrder < options[j].SortOrder
			}
			return options[i].OptionID < options[j].OptionID
		})
		nodes = append(nodes, npcdialogue.AdminDialogueNodeDetail{
			NodeID:               node.NodeID,
			NodeType:             node.NodeType,
			Speaker:              node.Speaker,
			Content:              node.Content,
			ContentFormat:        node.ContentFormat,
			PortraitKey:          node.PortraitKey,
			NextNodeID:           node.NextNodeID,
			ClientAnimationKey:   node.ClientAnimationKey,
			ClientAnimationBlock: node.ClientAnimationBlock,
			SortOrder:            r.nodeSortOrderByKey[npcDialogueNodeKey(dialogueID, node.NodeID)],
			Conditions:           npcdialogue.DecodeAdminConditionsJSON(node.ConditionsJSON),
			Effects:              npcdialogue.DecodeAdminEffectsJSON(node.EffectsJSON),
			Options:              options,
		})
	}
	sort.Slice(nodes, func(i int, j int) bool {
		if nodes[i].SortOrder != nodes[j].SortOrder {
			return nodes[i].SortOrder < nodes[j].SortOrder
		}
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes
}

func (r *NPCDialogueRepository) replaceDialogueNodesLocked(dialogueID int64, nodes []npcdialogue.AdminDialogueNodeInput) {
	r.deleteDialogueNodesLocked(dialogueID)
	for _, node := range nodes {
		r.nodesByDialogueNode[npcDialogueNodeKey(dialogueID, node.NodeID)] = npcdialogue.DialogueNode{
			DialogueID:           dialogueID,
			NodeID:               node.NodeID,
			NodeType:             node.NodeType,
			Speaker:              node.Speaker,
			Content:              node.Content,
			ContentFormat:        node.ContentFormat,
			PortraitKey:          node.PortraitKey,
			NextNodeID:           node.NextNodeID,
			ClientAnimationKey:   node.ClientAnimationKey,
			ClientAnimationBlock: node.ClientAnimationBlock,
			ConditionsJSON:       npcdialogue.EncodeAdminConditionsJSON(node.Conditions),
			EffectsJSON:          npcdialogue.EncodeAdminEffectsJSON(node.Effects),
		}
		options := make([]npcdialogue.DialogueOption, 0, len(node.Options))
		for _, option := range node.Options {
			options = append(options, npcdialogue.DialogueOption{
				DialogueID:     dialogueID,
				NodeID:         node.NodeID,
				OptionID:       option.OptionID,
				OptionText:     option.OptionText,
				OptionFormat:   option.OptionFormat,
				NextNodeID:     option.NextNodeID,
				ConditionsJSON: npcdialogue.EncodeAdminConditionsJSON(option.Conditions),
			})
			r.optionSortOrderByKey[npcDialogueOptionKey(dialogueID, node.NodeID, option.OptionID)] = option.SortOrder
		}
		r.optionsByDialogueNode[npcDialogueNodeKey(dialogueID, node.NodeID)] = options
		r.nodeSortOrderByKey[npcDialogueNodeKey(dialogueID, node.NodeID)] = node.SortOrder
	}
}

func (r *NPCDialogueRepository) deleteDialogueNodesLocked(dialogueID int64) {
	prefix := fmt.Sprintf("%d:", dialogueID)
	for key := range r.nodesByDialogueNode {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.nodesByDialogueNode, key)
		}
	}
	for key := range r.optionsByDialogueNode {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.optionsByDialogueNode, key)
		}
	}
	for key := range r.nodeSortOrderByKey {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.nodeSortOrderByKey, key)
		}
	}
	for key := range r.optionSortOrderByKey {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.optionSortOrderByKey, key)
		}
	}
}

func npcDialogueEntityEntryKey(entityID uint64, entryID string) string {
	return fmt.Sprintf("%d:%s", entityID, entryID)
}

func npcDialogueNodeKey(dialogueID int64, nodeID string) string {
	return fmt.Sprintf("%d:%s", dialogueID, nodeID)
}

func npcDialogueOptionKey(dialogueID int64, nodeID string, optionID string) string {
	return fmt.Sprintf("%d:%s:%s", dialogueID, nodeID, optionID)
}
