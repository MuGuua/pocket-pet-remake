package quest

import (
	"context"
	"sort"
	"strings"
	"time"
)

const (
	StateLocked        = "LOCKED"
	StateAvailable     = "AVAILABLE"
	StateAccepted      = "ACCEPTED"
	StateReadyToSubmit = "READY_TO_SUBMIT"
	StateCompleted     = "COMPLETED"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAdminTemplates 返回后台任务模板列表。
// 这样后台页的分页和筛选都复用服务端统一规则，不依赖前端自行裁剪结果。
func (s *Service) ListAdminTemplates(ctx context.Context, query AdminTemplateListQuery) (*AdminTemplateList, error) {
	result, err := s.repo.ListTemplatesForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminTemplateList{Items: []AdminTemplateSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminTemplateDetail 返回后台编辑任务模板所需的完整快照。
func (s *Service) GetAdminTemplateDetail(ctx context.Context, questID uint64) (*AdminTemplateDetail, error) {
	result, err := s.repo.FindAdminTemplateDetailByQuestID(ctx, questID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminQuestTemplateNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminTemplate(ctx context.Context, input AdminCreateTemplateInput) (*AdminTemplateDetail, error) {
	input = input.Normalize()
	if input.Name == "" || input.Title == "" || !validAcceptConditions(input.AcceptConditions, input.QuestID) {
		return nil, ErrInvalidAdminQuestInput
	}
	return s.repo.CreateTemplateForAdmin(ctx, input)
}

func (s *Service) UpdateAdminTemplate(ctx context.Context, questID uint64, input AdminUpdateTemplateInput) (*AdminTemplateDetail, error) {
	input = input.Normalize()
	if questID == 0 || input.Name == "" || input.Title == "" || !validAcceptConditions(input.AcceptConditions, questID) {
		return nil, ErrInvalidAdminQuestInput
	}
	result, err := s.repo.UpdateTemplateForAdmin(ctx, questID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminQuestTemplateNotFound
	}
	return result, nil
}

// validAcceptConditions 拒绝不完整、未知及自引用条件，避免错误配置把任务永久锁死。
func validAcceptConditions(conditions []AcceptCondition, currentQuestID uint64) bool {
	for _, condition := range conditions {
		switch condition.Type {
		case AcceptConditionQuestCompleted:
			if condition.QuestID == 0 || (currentQuestID > 0 && condition.QuestID == currentQuestID) {
				return false
			}
		case AcceptConditionPlayerLevel:
			if condition.Value == 0 || !validConditionOperator(condition.Operator) {
				return false
			}
		case AcceptConditionPlayerStat:
			if !validPlayerStatKey(condition.StatKey) || !validConditionOperator(condition.Operator) {
				return false
			}
		case AcceptConditionScene:
			if condition.SceneID == 0 {
				return false
			}
		case AcceptConditionItemCount:
			if condition.ItemID == 0 || condition.Value == 0 || !validConditionOperator(condition.Operator) {
				return false
			}
		case AcceptConditionPetLevel:
			if condition.Value == 0 || !validConditionOperator(condition.Operator) {
				return false
			}
		case AcceptConditionStoryFlag:
			if strings.TrimSpace(condition.FlagKey) == "" {
				return false
			}
		case AcceptConditionTimeWindow:
			start, startErr := time.Parse(time.RFC3339, condition.StartAt)
			end, endErr := time.Parse(time.RFC3339, condition.EndAt)
			if startErr != nil || endErr != nil || !start.Before(end) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validConditionOperator(operator string) bool {
	return operator == "gte" || operator == "eq" || operator == "lte"
}

func validPlayerStatKey(statKey string) bool {
	switch statKey {
	case "hp_max", "atk", "def", "spd", "mana":
		return true
	default:
		return false
	}
}

func (s *Service) DeleteAdminTemplate(ctx context.Context, questID uint64) error {
	return s.repo.DeleteTemplateForAdmin(ctx, questID)
}

// ListAdminPlayerQuests 返回后台玩家任务列表。
func (s *Service) ListAdminPlayerQuests(ctx context.Context, query AdminPlayerQuestListQuery) (*AdminPlayerQuestList, error) {
	result, err := s.repo.ListPlayerQuestsForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPlayerQuestList{Items: []AdminPlayerQuestSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminPlayerQuestDetail(ctx context.Context, recordID uint64) (*AdminPlayerQuestDetail, error) {
	result, err := s.repo.FindAdminPlayerQuestDetailByRecordID(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminPlayerQuestNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminPlayerQuest(ctx context.Context, input AdminCreatePlayerQuestInput) (*AdminPlayerQuestDetail, error) {
	input = input.Normalize()
	if input.PlayerID == 0 || input.QuestID == 0 || input.State == "" {
		return nil, ErrInvalidAdminQuestInput
	}
	return s.repo.CreatePlayerQuestForAdmin(ctx, input)
}

func (s *Service) UpdateAdminPlayerQuest(ctx context.Context, recordID uint64, input AdminUpdatePlayerQuestInput) (*AdminPlayerQuestDetail, error) {
	input = input.Normalize()
	if recordID == 0 || input.PlayerID == 0 || input.QuestID == 0 || input.State == "" {
		return nil, ErrInvalidAdminQuestInput
	}
	result, err := s.repo.UpdatePlayerQuestForAdmin(ctx, recordID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrAdminPlayerQuestNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminPlayerQuest(ctx context.Context, recordID uint64) error {
	return s.repo.DeletePlayerQuestForAdmin(ctx, recordID)
}

func (s *Service) List(ctx context.Context, playerID uint64) ([]Summary, uint64, error) {
	templates, playerQuestMap, playerObjectiveMap, err := s.loadState(ctx, playerID)
	if err != nil {
		return nil, 0, err
	}
	facts, err := s.repo.LoadAcceptConditionFacts(ctx, playerID)
	if err != nil {
		return nil, 0, err
	}
	if err := s.materializeAutoAcceptedQuests(ctx, playerID, templates, playerQuestMap, playerObjectiveMap, facts); err != nil {
		return nil, 0, err
	}

	summaries := make([]Summary, 0, len(templates))
	var trackedQuestID uint64
	completed := completedQuestSet(playerQuestMap)
	for _, template := range templates {
		summary := buildSummary(template, playerQuestMap[template.QuestID], playerObjectiveMap[template.QuestID], completed, facts)
		if summary.Tracked {
			trackedQuestID = summary.QuestID
		}
		summaries = append(summaries, summary)
	}
	if trackedQuestID == 0 {
		for _, summary := range summaries {
			if summary.State == StateAccepted || summary.State == StateAvailable || summary.State == StateReadyToSubmit {
				trackedQuestID = summary.QuestID
				break
			}
		}
	}
	return summaries, trackedQuestID, nil
}

func (s *Service) Accept(ctx context.Context, playerID uint64, questID uint64, npcID uint64) (Summary, error) {
	template, playerQuestMap, _, err := s.loadSingle(ctx, playerID, questID)
	if err != nil {
		return Summary{}, err
	}
	if template.AcceptMode == "NPC" && template.StartNPCID != 0 && template.StartNPCID != npcID {
		return Summary{}, ErrQuestAcceptNPCMismatch
	}

	completed := completedQuestSet(playerQuestMap)
	facts, err := s.repo.LoadAcceptConditionFacts(ctx, playerID)
	if err != nil {
		return Summary{}, err
	}
	existing := playerQuestMap[questID]
	if !isUnlocked(template, completed, facts) {
		return Summary{}, ErrQuestLocked
	}
	if existing != nil && existing.State == StateCompleted {
		return buildSummary(*template, existing, nil, completed, facts), nil
	}

	tracked := template.AutoTrack
	if err := s.repo.UpsertPlayerQuest(ctx, PlayerQuest{
		PlayerID: playerID,
		QuestID:  questID,
		State:    StateAccepted,
		Tracked:  tracked,
	}); err != nil {
		return Summary{}, err
	}

	objectives := make([]PlayerObjective, 0, len(template.Objectives))
	for _, objective := range template.Objectives {
		objectives = append(objectives, PlayerObjective{
			PlayerID:     playerID,
			QuestID:      questID,
			ObjectiveID:  objective.ObjectiveID,
			Description:  objective.Description,
			CurrentValue: 0,
			TargetValue:  objective.TargetValue,
			Completed:    false,
		})
	}
	if err := s.repo.ReplacePlayerObjectives(ctx, playerID, questID, objectives); err != nil {
		return Summary{}, err
	}
	if tracked {
		if err := s.repo.SetTrackedQuest(ctx, playerID, questID); err != nil {
			return Summary{}, err
		}
	}
	refreshedObjectives := make([]PlayerObjective, 0, len(objectives))
	refreshedObjectives = append(refreshedObjectives, objectives...)
	playerQuest := &PlayerQuest{PlayerID: playerID, QuestID: questID, State: StateAccepted, Tracked: tracked}
	return buildSummary(*template, playerQuest, refreshedObjectives, completed, facts), nil
}

func (s *Service) Submit(ctx context.Context, playerID uint64, questID uint64, npcID uint64) (SubmitResult, error) {
	template, playerQuestMap, playerObjectiveMap, err := s.loadSingle(ctx, playerID, questID)
	if err != nil {
		return SubmitResult{}, err
	}
	if template.SubmitMode == "NPC" && template.SubmitNPCID != 0 && template.SubmitNPCID != npcID {
		return SubmitResult{}, ErrQuestSubmitNPCMismatch
	}

	existing := playerQuestMap[questID]
	if existing == nil {
		return SubmitResult{}, ErrQuestNotAvailable
	}
	if existing.State == StateCompleted {
		completed := completedQuestSet(playerQuestMap)
		return SubmitResult{Summary: buildSummary(*template, existing, playerObjectiveMap[questID], completed, AcceptConditionFacts{})}, nil
	}
	if existing.State != StateReadyToSubmit {
		return SubmitResult{}, ErrQuestNotReady
	}

	objectives := playerObjectiveMap[questID]
	if len(objectives) == 0 || !allObjectivesCompleted(objectives) {
		return SubmitResult{}, ErrQuestNotReady
	}

	if err := s.repo.UpsertPlayerQuest(ctx, PlayerQuest{
		PlayerID: playerID,
		QuestID:  questID,
		State:    StateCompleted,
		Tracked:  existing.Tracked,
	}); err != nil {
		return SubmitResult{}, err
	}
	if err := s.repo.ReplacePlayerObjectives(ctx, playerID, questID, objectives); err != nil {
		return SubmitResult{}, err
	}

	playerQuest := &PlayerQuest{PlayerID: playerID, QuestID: questID, State: StateCompleted, Tracked: existing.Tracked}
	completed := completedQuestSet(playerQuestMap)
	completed[questID] = true
	return SubmitResult{
		Summary: buildSummary(*template, playerQuest, objectives, completed, AcceptConditionFacts{}),
		Rewards: collectSupportedRewards(template.Rewards),
	}, nil
}

// RevertCompletedQuestToReady 把已标记完成但奖励发放失败的任务退回“可提交”。
// 任务目标仍保持完成状态，玩家下次点击提交即可重试领奖，避免出现任务完成但奖励缺失。
func (s *Service) RevertCompletedQuestToReady(ctx context.Context, playerID uint64, questID uint64) error {
	if playerID == 0 || questID == 0 {
		return ErrQuestNotAvailable
	}
	return s.repo.RevertCompletedQuestToReady(ctx, playerID, questID)
}

func (s *Service) Track(ctx context.Context, playerID uint64, questID uint64) error {
	template, _, _, err := s.loadSingle(ctx, playerID, questID)
	if err != nil {
		return err
	}
	if template == nil {
		return ErrQuestTemplateNotFound
	}
	return s.repo.SetTrackedQuest(ctx, playerID, questID)
}

func (s *Service) HandleEvent(ctx context.Context, event Event) ([]Summary, error) {
	templates, playerQuestMap, playerObjectiveMap, err := s.loadState(ctx, event.PlayerID)
	if err != nil {
		return nil, err
	}

	completed := completedQuestSet(playerQuestMap)
	changedQuestIDs := map[uint64]bool{}
	facts, err := s.repo.LoadAcceptConditionFacts(ctx, event.PlayerID)
	if err != nil {
		return nil, err
	}

	if err := s.materializeAutoAcceptedQuests(ctx, event.PlayerID, templates, playerQuestMap, playerObjectiveMap, facts); err != nil {
		return nil, err
	}

	for _, template := range templates {
		current := playerQuestMap[template.QuestID]
		if current == nil || current.State != StateAccepted {
			continue
		}

		objectives := playerObjectiveMap[template.QuestID]
		if len(objectives) == 0 {
			objectives = buildInitialObjectives(event.PlayerID, template)
		}
		updatedObjectives, changed := applyEventToObjectives(template, objectives, event)
		if !changed {
			continue
		}
		if err := s.repo.ReplacePlayerObjectives(ctx, event.PlayerID, template.QuestID, updatedObjectives); err != nil {
			return nil, err
		}
		playerObjectiveMap[template.QuestID] = updatedObjectives

		nextState := StateAccepted
		if allObjectivesCompleted(updatedObjectives) {
			nextState = StateReadyToSubmit
		}
		current.State = nextState
		if err := s.repo.UpsertPlayerQuest(ctx, *current); err != nil {
			return nil, err
		}
		changedQuestIDs[template.QuestID] = true
	}

	if len(changedQuestIDs) == 0 {
		return []Summary{}, nil
	}

	summaries := make([]Summary, 0, len(changedQuestIDs))
	for _, template := range templates {
		if !changedQuestIDs[template.QuestID] {
			continue
		}
		summaries = append(summaries, buildSummary(template, playerQuestMap[template.QuestID], playerObjectiveMap[template.QuestID], completed, facts))
	}
	return summaries, nil
}

func (s *Service) materializeAutoAcceptedQuests(ctx context.Context, playerID uint64, templates []Template, playerQuestMap map[uint64]*PlayerQuest, playerObjectiveMap map[uint64][]PlayerObjective, facts AcceptConditionFacts) error {
	completed := completedQuestSet(playerQuestMap)
	for _, template := range templates {
		if playerQuestMap[template.QuestID] != nil || template.AcceptMode != "AUTO" || !isUnlocked(&template, completed, facts) {
			continue
		}
		tracked := template.AutoTrack
		current := &PlayerQuest{
			PlayerID: playerID,
			QuestID:  template.QuestID,
			State:    StateAccepted,
			Tracked:  tracked,
		}
		if err := s.repo.UpsertPlayerQuest(ctx, *current); err != nil {
			return err
		}
		objectives := buildInitialObjectives(playerID, template)
		if err := s.repo.ReplacePlayerObjectives(ctx, playerID, template.QuestID, objectives); err != nil {
			return err
		}
		if tracked {
			if err := s.repo.SetTrackedQuest(ctx, playerID, template.QuestID); err != nil {
				return err
			}
		}
		playerQuestMap[template.QuestID] = current
		playerObjectiveMap[template.QuestID] = objectives
	}
	return nil
}

func (s *Service) loadState(ctx context.Context, playerID uint64) ([]Template, map[uint64]*PlayerQuest, map[uint64][]PlayerObjective, error) {
	templates, err := s.repo.ListTemplates(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].QuestID < templates[j].QuestID })

	playerQuests, err := s.repo.ListPlayerQuestsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, nil, nil, err
	}
	playerObjectives, err := s.repo.ListPlayerObjectivesByPlayerID(ctx, playerID)
	if err != nil {
		return nil, nil, nil, err
	}

	playerQuestMap := make(map[uint64]*PlayerQuest, len(playerQuests))
	for index := range playerQuests {
		value := playerQuests[index]
		copyValue := value
		playerQuestMap[value.QuestID] = &copyValue
	}
	playerObjectiveMap := make(map[uint64][]PlayerObjective)
	for _, objective := range playerObjectives {
		playerObjectiveMap[objective.QuestID] = append(playerObjectiveMap[objective.QuestID], objective)
	}
	return templates, playerQuestMap, playerObjectiveMap, nil
}

func (s *Service) loadSingle(ctx context.Context, playerID uint64, questID uint64) (*Template, map[uint64]*PlayerQuest, map[uint64][]PlayerObjective, error) {
	template, err := s.repo.FindTemplateByQuestID(ctx, questID)
	if err != nil {
		return nil, nil, nil, err
	}
	if template == nil {
		return nil, nil, nil, ErrQuestTemplateNotFound
	}
	_, playerQuestMap, playerObjectiveMap, err := s.loadState(ctx, playerID)
	if err != nil {
		return nil, nil, nil, err
	}
	return template, playerQuestMap, playerObjectiveMap, nil
}

func completedQuestSet(playerQuestMap map[uint64]*PlayerQuest) map[uint64]bool {
	result := make(map[uint64]bool, len(playerQuestMap))
	for questID, value := range playerQuestMap {
		if value != nil && value.State == StateCompleted {
			result[questID] = true
		}
	}
	return result
}

func isUnlocked(template *Template, completed map[uint64]bool, facts AcceptConditionFacts) bool {
	if template == nil {
		return false
	}
	for _, preQuestID := range template.PreQuestIDs {
		if !completed[preQuestID] {
			return false
		}
	}
	if template.MinPlayerLevel > 1 && facts.Level < uint64(template.MinPlayerLevel) {
		return false
	}
	for _, condition := range template.AcceptConditions {
		if !acceptConditionMatches(condition, completed, facts) {
			return false
		}
	}
	return true
}

// acceptConditionMatches 按条件类型读取对应权威数据；未知或配置不完整的条件一律视为不满足。
func acceptConditionMatches(condition AcceptCondition, completed map[uint64]bool, facts AcceptConditionFacts) bool {
	switch condition.Type {
	case AcceptConditionQuestCompleted:
		return condition.QuestID > 0 && completed[condition.QuestID]
	case AcceptConditionPlayerLevel:
		return compareConditionValue(facts.Level, condition.Operator, condition.Value)
	case AcceptConditionPlayerStat:
		actual, ok := facts.Stats[condition.StatKey]
		return ok && compareConditionValue(actual, condition.Operator, condition.Value)
	case AcceptConditionScene:
		return condition.SceneID > 0 && uint64(facts.SceneID) == uint64(condition.SceneID)
	case AcceptConditionItemCount:
		return condition.ItemID > 0 && compareConditionValue(facts.ItemCounts[condition.ItemID], condition.Operator, condition.Value)
	case AcceptConditionPetLevel:
		actual := facts.MaxPetLevel
		if condition.PetID > 0 {
			actual = facts.PetLevels[condition.PetID]
		}
		return compareConditionValue(actual, condition.Operator, condition.Value)
	case AcceptConditionStoryFlag:
		return condition.FlagKey != "" && facts.StoryFlags[condition.FlagKey]
	case AcceptConditionTimeWindow:
		return timeWindowMatches(condition.StartAt, condition.EndAt, facts.Now)
	default:
		return false
	}
}

// compareConditionValue 统一数值条件比较规则；空操作符按大于等于处理，兼容后台默认值。
func compareConditionValue(actual uint64, operator string, expected uint64) bool {
	switch operator {
	case "eq":
		return actual == expected
	case "lte":
		return actual <= expected
	case "gte", "":
		return actual >= expected
	default:
		return false
	}
}

// timeWindowMatches 使用 RFC3339 时间边界判断当前服务端时间是否处于可接取窗口。
func timeWindowMatches(startAt string, endAt string, now time.Time) bool {
	if startAt == "" && endAt == "" {
		return false
	}
	if startAt != "" {
		start, err := time.Parse(time.RFC3339, startAt)
		if err != nil || now.Before(start) {
			return false
		}
	}
	if endAt != "" {
		end, err := time.Parse(time.RFC3339, endAt)
		if err != nil || now.After(end) {
			return false
		}
	}
	return true
}

func buildSummary(template Template, playerQuest *PlayerQuest, objectives []PlayerObjective, completed map[uint64]bool, facts AcceptConditionFacts) Summary {
	state := StateLocked
	tracked := false
	if playerQuest != nil {
		state = playerQuest.State
		tracked = playerQuest.Tracked
	} else if isUnlocked(&template, completed, facts) {
		state = StateAvailable
	}

	objectiveSummaries := make([]ObjectiveSummary, 0, len(template.Objectives))
	objectiveMap := make(map[uint64]PlayerObjective, len(objectives))
	for _, objective := range objectives {
		objectiveMap[objective.ObjectiveID] = objective
	}
	for _, objective := range template.Objectives {
		existing, ok := objectiveMap[objective.ObjectiveID]
		if ok {
			objectiveSummaries = append(objectiveSummaries, ObjectiveSummary{
				ObjectiveID:    existing.ObjectiveID,
				EventType:      objective.EventType,
				Description:    existing.Description,
				Current:        existing.CurrentValue,
				Target:         existing.TargetValue,
				Completed:      existing.Completed,
				TargetSelector: objective.TargetSelector,
				Guide:          objective.Guide,
			})
			continue
		}
		objectiveSummaries = append(objectiveSummaries, ObjectiveSummary{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			Current:        0,
			Target:         objective.TargetValue,
			Completed:      false,
			TargetSelector: objective.TargetSelector,
			Guide:          objective.Guide,
		})
	}

	return Summary{
		QuestID:              template.QuestID,
		QuestType:            template.QuestType,
		ClientIconID:         template.ClientIconID,
		State:                state,
		Tracked:              tracked,
		StartNPCID:           template.StartNPCID,
		SubmitNPCID:          template.SubmitNPCID,
		AcceptAnimationKey:   template.AcceptAnimationKey,
		SubmitAnimationKey:   template.SubmitAnimationKey,
		Title:                template.Title,
		Description:          template.Description,
		CompletionPromptText: template.CompletionPromptText,
		Objectives:           objectiveSummaries,
		Rewards:              collectSupportedRewards(template.Rewards),
	}
}

func buildInitialObjectives(playerID uint64, template Template) []PlayerObjective {
	result := make([]PlayerObjective, 0, len(template.Objectives))
	for _, objective := range template.Objectives {
		result = append(result, PlayerObjective{
			PlayerID:     playerID,
			QuestID:      template.QuestID,
			ObjectiveID:  objective.ObjectiveID,
			Description:  objective.Description,
			CurrentValue: 0,
			TargetValue:  objective.TargetValue,
			Completed:    false,
		})
	}
	return result
}

func applyEventToObjectives(template Template, objectives []PlayerObjective, event Event) ([]PlayerObjective, bool) {
	changed := false
	templateObjectiveMap := make(map[uint64]ObjectiveTemplate, len(template.Objectives))
	for _, objective := range template.Objectives {
		templateObjectiveMap[objective.ObjectiveID] = objective
	}
	// 多阶段任务按 objective_id 从小到大顺序推进，避免玩家跳过前置阶段直接完成后续目标。
	activeStageID, hasActiveStage := firstIncompleteObjectiveID(objectives)

	nextObjectives := make([]PlayerObjective, 0, len(objectives))
	for _, current := range objectives {
		templateObjective, ok := templateObjectiveMap[current.ObjectiveID]
		if !ok || current.Completed {
			nextObjectives = append(nextObjectives, current)
			continue
		}
		if hasActiveStage && current.ObjectiveID != activeStageID {
			nextObjectives = append(nextObjectives, current)
			continue
		}
		if !eventMatchesObjective(event, templateObjective) {
			nextObjectives = append(nextObjectives, current)
			continue
		}
		current.CurrentValue += maxUint32(1, event.Count)
		if current.CurrentValue >= current.TargetValue {
			current.CurrentValue = current.TargetValue
			current.Completed = true
		}
		changed = true
		nextObjectives = append(nextObjectives, current)
	}
	return nextObjectives, changed
}

// firstIncompleteObjectiveID 返回当前尚未完成的最早阶段 ID。
// 若全部完成则返回 false，表示没有可推进的阶段。
func firstIncompleteObjectiveID(objectives []PlayerObjective) (uint64, bool) {
	var stageID uint64
	found := false
	for _, objective := range objectives {
		if objective.Completed {
			continue
		}
		if !found || objective.ObjectiveID < stageID {
			stageID = objective.ObjectiveID
			found = true
		}
	}
	return stageID, found
}

func eventMatchesObjective(event Event, objective ObjectiveTemplate) bool {
	if objective.EventType != event.EventType {
		return false
	}
	if sceneID, ok := selectorUint32(objective.TargetSelector, "scene_id"); ok && sceneID != event.SceneID {
		return false
	}
	if npcID, ok := selectorUint64(objective.TargetSelector, "npc_id"); ok && npcID != event.NPCID {
		return false
	}
	if battleType, ok := objective.TargetSelector["battle_type"].(string); ok {
		value, _ := event.Meta["battle_type"].(string)
		if battleType != value {
			return false
		}
	}
	return true
}

func allObjectivesCompleted(values []PlayerObjective) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !value.Completed {
			return false
		}
	}
	return true
}

func selectorUint32(selector map[string]any, key string) (uint32, bool) {
	if selector == nil {
		return 0, false
	}
	value, ok := selector[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return uint32(typed), true
	case int:
		return uint32(typed), true
	case int32:
		return uint32(typed), true
	case int64:
		return uint32(typed), true
	case uint32:
		return typed, true
	case uint64:
		return uint32(typed), true
	}
	return 0, false
}

func selectorUint64(selector map[string]any, key string) (uint64, bool) {
	if selector == nil {
		return 0, false
	}
	value, ok := selector[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return uint64(typed), true
	case int:
		return uint64(typed), true
	case int32:
		return uint64(typed), true
	case int64:
		return uint64(typed), true
	case uint32:
		return uint64(typed), true
	case uint64:
		return typed, true
	}
	return 0, false
}

func maxUint32(a uint32, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

// collectSupportedRewards 过滤出当前运行时已经有正式发放能力的奖励。
// 先把金币和角色经验接入服务端权威链路，其他奖励类型后续在背包/宠物/功能系统补齐后再落正式发放。
func collectSupportedRewards(configured []Reward) []Reward {
	result := make([]Reward, 0, len(configured))
	for _, reward := range configured {
		switch strings.ToLower(strings.TrimSpace(reward.Type)) {
		case "gold", "exp", "item", "pet", "feature_unlock":
			result = append(result, reward)
		}
	}
	return result
}
