package npcdialogue

import "context"

// Service 封装 NPC 结构化剧情的权威推进逻辑，客户端只上传“继续”或“选择了哪个选项”。
type Service struct {
	repo   Repository
	quests QuestSummaryReader
}

// NewService 创建 NPC 结构化剧情服务；当 repo 为空时，上层可安全地把该能力视为未启用。
func NewService(repo Repository, quests QuestSummaryReader) *Service {
	return &Service{repo: repo, quests: quests}
}

// ListAdminDialogues 返回后台剧情配置列表，供运营按 NPC 菜单项定位剧情聚合数据。
func (s *Service) ListAdminDialogues(ctx context.Context, query AdminDialogueListQuery) (*AdminDialogueList, error) {
	if s == nil || s.repo == nil {
		query = query.Normalize()
		return &AdminDialogueList{Items: []AdminDialogueSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	result, err := s.repo.ListDialoguesForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminDialogueList{Items: []AdminDialogueSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminDialogueDetail 返回后台剧情详情，包含节点与选项聚合结构。
func (s *Service) GetAdminDialogueDetail(ctx context.Context, entityID uint64, entryID string) (*AdminDialogueDetail, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueNotFound
	}
	result, err := s.repo.FindDialogueDetailForAdmin(ctx, entityID, entryID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrDialogueNotFound
	}
	return result, nil
}

// CreateAdminDialogue 创建后台剧情聚合配置，正式数据仍落在数据库表中持久化。
func (s *Service) CreateAdminDialogue(ctx context.Context, input AdminCreateDialogueInput) (*AdminDialogueDetail, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueNotFound
	}
	input = input.Normalize()
	if input.EntityID == 0 || input.EntryID == "" || input.Title == "" || input.StartNodeID == "" {
		return nil, ErrInvalidAdminDialogueInput
	}
	exists, err := s.repo.MenuEntryExists(ctx, input.EntityID, input.EntryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrDialogueMenuEntryNotFound
	}
	return s.repo.CreateDialogueForAdmin(ctx, input)
}

// UpdateAdminDialogue 更新后台剧情聚合配置；节点和选项由服务端整体替换，避免前端自行拼增量补丁。
func (s *Service) UpdateAdminDialogue(ctx context.Context, entityID uint64, entryID string, input AdminUpdateDialogueInput) (*AdminDialogueDetail, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueNotFound
	}
	input = input.Normalize()
	if entityID == 0 || entryID == "" || input.EntityID == 0 || input.Title == "" || input.StartNodeID == "" {
		return nil, ErrInvalidAdminDialogueInput
	}
	exists, err := s.repo.MenuEntryExists(ctx, input.EntityID, entryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrDialogueMenuEntryNotFound
	}
	result, err := s.repo.UpdateDialogueForAdmin(ctx, entityID, entryID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrDialogueNotFound
	}
	return result, nil
}

// DeleteAdminDialogue 删除后台剧情聚合配置以及其全部节点/选项配置。
func (s *Service) DeleteAdminDialogue(ctx context.Context, entityID uint64, entryID string) error {
	if s == nil || s.repo == nil {
		return ErrDialogueNotFound
	}
	return s.repo.DeleteDialogueForAdmin(ctx, entityID, entryID)
}

// StartDialogue 以 NPC 菜单项为入口开启一段新的剧情，并返回首个运行态节点给客户端。
func (s *Service) StartDialogue(ctx context.Context, playerID uint64, entityID uint64, entryID string) (*RuntimeNode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueNotFound
	}
	dialogue, err := s.repo.FindDialogueByEntityEntry(ctx, entityID, entryID)
	if err != nil {
		return nil, err
	}
	if dialogue == nil {
		return nil, ErrDialogueNotFound
	}
	startNode, err := s.repo.FindNode(ctx, dialogue.DialogueID, dialogue.StartNodeID)
	if err != nil {
		return nil, err
	}
	if startNode == nil {
		return nil, ErrDialogueNodeNotFound
	}
	ok, err := MatchNodeConditions(ctx, s.quests, playerID, startNode.ConditionsJSON)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDialogueNotFound
	}
	if err := s.repo.UpsertSession(ctx, DialogueSession{PlayerID: playerID, EntityID: entityID, DialogueID: dialogue.DialogueID, CurrentNodeID: dialogue.StartNodeID, Status: SessionStatusActive}); err != nil {
		return nil, err
	}
	return s.loadRuntimeNode(ctx, playerID, dialogue.DialogueID, entityID, dialogue.StartNodeID)
}

// GetActiveDialogue 在断线重连时恢复玩家当前未结束的剧情节点，不会额外推进会话位置。
func (s *Service) GetActiveDialogue(ctx context.Context, playerID uint64) (uint64, *RuntimeNode, error) {
	if s == nil || s.repo == nil || playerID == 0 {
		return 0, nil, nil
	}
	session, err := s.repo.FindSessionByPlayerID(ctx, playerID)
	if err != nil {
		return 0, nil, err
	}
	if session == nil || session.Status != SessionStatusActive {
		return 0, nil, nil
	}
	node, err := s.assembleRuntimeNode(ctx, playerID, session.DialogueID, session.CurrentNodeID, false)
	if err != nil {
		return 0, nil, err
	}
	return session.EntityID, node, nil
}

// AdvanceDialogue 在客户端点击“继续”或剧情动画播完后，把会话推进到当前节点的下一个节点。
func (s *Service) AdvanceDialogue(ctx context.Context, playerID uint64, entityID uint64, dialogueID int64, nodeID string) (*RuntimeNode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueSessionNotFound
	}
	session, err := s.repo.FindSessionByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrDialogueSessionNotFound
	}
	if session.EntityID != entityID || session.DialogueID != dialogueID || session.CurrentNodeID != nodeID || session.Status != SessionStatusActive {
		return nil, ErrDialogueSessionMismatch
	}
	currentNode, err := s.repo.FindNode(ctx, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	if currentNode == nil {
		return nil, ErrDialogueNodeNotFound
	}
	if currentNode.NextNodeID == "" {
		if err := s.repo.DeleteSession(ctx, playerID); err != nil {
			return nil, err
		}
		return &RuntimeNode{DialogueID: dialogueID, NodeID: nodeID, NodeType: NodeTypeEnd, IsEnd: true}, nil
	}
	return s.loadRuntimeNode(ctx, playerID, dialogueID, entityID, currentNode.NextNodeID)
}

// ChooseOption 在客户端选择分支按钮后，根据服务端配置跳到对应 next_node_id。
func (s *Service) ChooseOption(ctx context.Context, playerID uint64, entityID uint64, dialogueID int64, nodeID string, optionID string) (*RuntimeNode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDialogueSessionNotFound
	}
	session, err := s.repo.FindSessionByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrDialogueSessionNotFound
	}
	if session.EntityID != entityID || session.DialogueID != dialogueID || session.CurrentNodeID != nodeID || session.Status != SessionStatusActive {
		return nil, ErrDialogueSessionMismatch
	}
	currentNode, err := s.repo.FindNode(ctx, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	if currentNode == nil {
		return nil, ErrDialogueNodeNotFound
	}
	if currentNode.NodeType != NodeTypeChoice {
		return nil, ErrDialogueOptionInvalid
	}
	options, err := s.filterAvailableOptions(ctx, playerID, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		if option.OptionID != optionID {
			continue
		}
		if option.NextNodeID == "" {
			if err := s.repo.DeleteSession(ctx, playerID); err != nil {
				return nil, err
			}
			return &RuntimeNode{DialogueID: dialogueID, NodeID: nodeID, NodeType: NodeTypeEnd, IsEnd: true}, nil
		}
		return s.loadRuntimeNode(ctx, playerID, dialogueID, entityID, option.NextNodeID)
	}
	return nil, ErrDialogueOptionInvalid
}

// loadRuntimeNode 统一把配置节点装配成客户端可消费的运行态结构，并同步刷新服务端会话当前位置。
func (s *Service) loadRuntimeNode(ctx context.Context, playerID uint64, dialogueID int64, entityID uint64, nodeID string) (*RuntimeNode, error) {
	runtimeNode, err := s.assembleRuntimeNode(ctx, playerID, dialogueID, nodeID, true)
	if err != nil {
		return nil, err
	}
	if runtimeNode.IsEnd {
		if err := s.repo.DeleteSession(ctx, playerID); err != nil {
			return nil, err
		}
		return runtimeNode, nil
	}
	if err := s.repo.UpsertSession(ctx, DialogueSession{PlayerID: playerID, EntityID: entityID, DialogueID: dialogueID, CurrentNodeID: nodeID, Status: SessionStatusActive}); err != nil {
		return nil, err
	}
	return runtimeNode, nil
}

// assembleRuntimeNode 把配置节点装配成运行态结构；persistSession=false 时只读恢复，不写回会话表。
func (s *Service) assembleRuntimeNode(ctx context.Context, playerID uint64, dialogueID int64, nodeID string, checkNodeConditions bool) (*RuntimeNode, error) {
	node, err := s.repo.FindNode(ctx, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, ErrDialogueNodeNotFound
	}
	if checkNodeConditions {
		ok, err := MatchNodeConditions(ctx, s.quests, playerID, node.ConditionsJSON)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDialogueNodeNotFound
		}
	}
	runtimeNode := &RuntimeNode{
		DialogueID:           node.DialogueID,
		NodeID:               node.NodeID,
		NodeType:             node.NodeType,
		Speaker:              node.Speaker,
		Content:              node.Content,
		ContentFormat:        node.ContentFormat,
		PortraitKey:          node.PortraitKey,
		ClientAnimationKey:   node.ClientAnimationKey,
		ClientAnimationBlock: node.ClientAnimationBlock,
		Options:              []DialogueOption{},
		IsEnd:                node.NodeType == NodeTypeEnd,
	}
	if node.NodeType == NodeTypeChoice {
		options, err := s.filterAvailableOptions(ctx, playerID, dialogueID, nodeID)
		if err != nil {
			return nil, err
		}
		runtimeNode.Options = options
	}
	effects := ParseNodeEffects(node.EffectsJSON)
	runtimeNode.EffectNotice = effects.Notice
	runtimeNode.EffectQuestEvent = effects.QuestEvent
	runtimeNode.EffectAcceptQuestID = effects.AcceptQuestID
	runtimeNode.EffectGrantItems = append([]EffectGrantItem{}, effects.GrantItems...)
	return runtimeNode, nil
}

// filterAvailableOptions 只返回当前玩家满足 conditions_json 的选项，避免客户端展示不可进入的分支。
func (s *Service) filterAvailableOptions(ctx context.Context, playerID uint64, dialogueID int64, nodeID string) ([]DialogueOption, error) {
	options, err := s.repo.ListOptions(ctx, dialogueID, nodeID)
	if err != nil {
		return nil, err
	}
	filtered := make([]DialogueOption, 0, len(options))
	for _, option := range options {
		ok, err := MatchNodeConditions(ctx, s.quests, playerID, option.ConditionsJSON)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, option)
		}
	}
	return filtered, nil
}
