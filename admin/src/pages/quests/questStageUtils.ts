import type { AdminQuestObjectiveInput } from '../../types/quest';

/** 后台任务阶段表单单行结构，便于 Form.List 维护多阶段配置。 */
export interface QuestStageFormItem {
  objective_id: number;
  event_type: string;
  description: string;
  target_value: number;
  npc_id: number;
  scene_id: number;
  battle_type: string;
  guide_text: string;
  guide_scene_id: number;
  menu_entry_id: number;
  dialogue_entry_id: number;
}

const DEFAULT_STAGE: QuestStageFormItem = {
  objective_id: 1,
  event_type: 'TALK_TO_NPC',
  description: '',
  target_value: 1,
  npc_id: 0,
  scene_id: 0,
  battle_type: '',
  guide_text: '',
  guide_scene_id: 0,
  menu_entry_id: 0,
  dialogue_entry_id: 0,
};

/** 创建单个默认阶段，供弹窗新增时使用。 */
export function createDefaultStage(objectiveID: number = 1): QuestStageFormItem {
  return { ...DEFAULT_STAGE, objective_id: objectiveID };
}

/** 创建默认阶段列表，新建任务模板时使用。 */
export function createDefaultQuestStages(): QuestStageFormItem[] {
  return [createDefaultStage(1)];
}

/** 将后台 API 目标定义转换为阶段表单结构。 */
export function apiObjectivesToStages(objectives: AdminQuestObjectiveInput[] | null | undefined): QuestStageFormItem[] {
  if (!objectives || objectives.length === 0) {
    return createDefaultQuestStages();
  }
  return objectives.map((item) => {
    const selector = item.target_selector ?? {};
    const guide = item.guide ?? {};
    const legacyTarget = (item as AdminQuestObjectiveInput & { target?: number }).target;
    return {
      objective_id: item.objective_id,
      event_type: item.event_type,
      description: item.description,
      target_value: item.target_value > 0 ? item.target_value : legacyTarget ?? 1,
      npc_id: readNumber(selector.npc_id),
      scene_id: readNumber(selector.scene_id),
      battle_type: typeof selector.battle_type === 'string' ? selector.battle_type : '',
      guide_text: guide.text ?? '',
      guide_scene_id: guide.scene_id ?? 0,
      menu_entry_id: guide.menu_entry_id ?? 0,
      dialogue_entry_id: guide.dialogue_entry_id ?? 0,
    };
  });
}

/** 将阶段表单结构转换为后台 API 目标定义。 */
export function stagesToApiObjectives(stages: QuestStageFormItem[]): AdminQuestObjectiveInput[] {
  return stages.map((stage) => {
    const targetSelector: Record<string, unknown> = {};
    if (stage.npc_id > 0) {
      targetSelector.npc_id = stage.npc_id;
    }
    if (stage.scene_id > 0) {
      targetSelector.scene_id = stage.scene_id;
    }
    if (stage.battle_type.trim()) {
      targetSelector.battle_type = stage.battle_type.trim();
    }
    const hasGuide =
      stage.guide_text.trim() ||
      stage.guide_scene_id > 0 ||
      stage.npc_id > 0 ||
      stage.menu_entry_id > 0 ||
      stage.dialogue_entry_id > 0;
    const guide = hasGuide
      ? {
          scene_id: stage.guide_scene_id > 0 ? stage.guide_scene_id : undefined,
          npc_id: stage.npc_id > 0 ? stage.npc_id : undefined,
          text: stage.guide_text.trim() || undefined,
          menu_entry_id: stage.menu_entry_id > 0 ? stage.menu_entry_id : undefined,
          dialogue_entry_id: stage.dialogue_entry_id > 0 ? stage.dialogue_entry_id : undefined,
        }
      : undefined;
    return {
      objective_id: stage.objective_id,
      event_type: stage.event_type,
      description: stage.description,
      target_value: stage.target_value,
      target_selector: targetSelector,
      guide,
    };
  });
}

/** 生成 NPC 菜单/剧情可见条件配置提示，供运营复制到 NPC 菜单 Tab。 */
export function buildStageConditionHint(questID: number, stage: QuestStageFormItem): string {
  return JSON.stringify(
    {
      quest_id: questID,
      quest_state: 'ACCEPTED',
      objective_id: stage.objective_id,
      objective_completed: false,
    },
    null,
    2,
  );
}

function readNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  return 0;
}
