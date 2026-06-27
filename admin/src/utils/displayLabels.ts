// 后台展示层枚举中文化：配置值仍使用英文常量，列表/详情/下拉只展示中文文案。

/** 任务类型 */
export const QUEST_TYPE_LABELS: Record<string, string> = {
  MAIN: '主线',
  SIDE: '支线',
  DAILY: '日常',
};

/** 任务接取/提交方式 */
export const QUEST_MODE_LABELS: Record<string, string> = {
  AUTO: '自动',
  NPC: 'NPC 交互',
};

/** 玩家任务进度状态 */
export const QUEST_STATE_LABELS: Record<string, string> = {
  LOCKED: '未解锁',
  AVAILABLE: '可接取',
  ACCEPTED: '进行中',
  READY_TO_SUBMIT: '待提交',
  COMPLETED: '已完成',
};

/** 玩家账号状态（服务端 status_text 为英文常量） */
export const PLAYER_STATUS_LABELS: Record<string, string> = {
  NORMAL: '正常',
  BANNED: '封禁',
  DELETED: '已删除',
  UNKNOWN: '未知',
};

/** NPC 菜单入口类型 */
export const NPC_ENTRY_TYPE_LABELS: Record<string, string> = {
  dialog: '对话',
  shop: '商店',
  quest: '任务',
  battle: '挑战',
  warehouse: '仓库',
};

/** NPC 菜单动作结果类型 */
export const NPC_ACTION_RESULT_LABELS: Record<string, string> = {
  notice: '提示',
  dialog: '对话',
  dialogue: '剧情对话',
  shop: '商店',
  battle: '直接开战',
  quest_accept: '接取任务',
  quest_submit: '提交任务',
  panel: '打开面板',
};

/** NPC 菜单项运行时状态 key（写入 state 字段，服务端按此判断菜单是否可展示） */
export const NPC_MENU_STATE_LABELS: Record<string, string> = {
  available: '可用',
  unavailable: '不可用',
  hidden: '隐藏',
  locked: '锁定',
  disabled: '停用',
};

/** 任务目标事件类型 */
export const QUEST_EVENT_TYPE_LABELS: Record<string, string> = {
  TALK_TO_NPC: '与 NPC 对话',
  ENTER_SCENE: '进入场景',
  WIN_BATTLE: '战斗胜利',
};

/** 物品大类 */
export const ITEM_TYPE_LABELS: Record<string, string> = {
  consumable: '消耗品',
  equipment: '装备',
  material: '材料',
  quest: '任务物品',
  currency: '货币',
  misc: '杂项',
  functional: '功能道具',
  box: '礼包',
};

/** 物品子类 */
export const ITEM_SUB_TYPE_LABELS: Record<string, string> = {
  expand: '扩容',
  gift_box: '礼包',
  equipment_enhance: '强化材料',
};

/** 材料类物品可选子分类（item_type=material 时在后台表单展示） */
export const MATERIAL_ITEM_SUB_TYPE_LABELS: Record<string, string> = {
  '': '普通材料',
  equipment_enhance: '强化材料',
};

/** 技能分类 */
export const SKILL_CATEGORY_LABELS: Record<string, string> = {
  common: '通用',
  character: '人物',
  pet: '宠物',
  monster: '怪物',
};

/** 技能类型 */
export const SKILL_TYPE_LABELS: Record<string, string> = {
  attack: '攻击',
  heal: '治疗',
  support: '辅助',
  control: '控制',
};

/** 技能目标类型 */
export const TARGET_TYPE_LABELS: Record<string, string> = {
  enemy_single: '单个敌人',
  enemy_all: '全体敌人',
  ally_single: '单个友方',
  ally_all: '全体友方',
  self: '自身',
  lowest: '生命最低',
};

/** 技能优先目标策略 */
export const PREFERRED_TARGET_LABELS: Record<string, string> = {
  lowest: '生命最低',
  highest: '生命最高',
};

/** 战斗控制状态 ID（与后端 battle.Status* 常量一致） */
export const BATTLE_CONTROL_STATUS_OPTIONS: Array<{ value: number; label: string }> = [
  { value: 2, label: '2 · 封印' },
  { value: 3, label: '3 · 眩晕' },
  { value: 9, label: '9 · 束缚' },
  { value: 10, label: '10 · 睡眠' },
  { value: 11, label: '11 · 麻痹' },
  { value: 12, label: '12 · 混乱' },
];

/** 根据控制状态 ID 返回中文名称 */
export function formatControlStatusLabel(statusID: number): string {
  const matched = BATTLE_CONTROL_STATUS_OPTIONS.find((item) => item.value === statusID);
  return matched?.label ?? String(statusID);
}

/** 背包容器类型 */
export const CONTAINER_TYPE_LABELS: Record<string, string> = {
  bag: '背包',
  warehouse: '仓库',
};

/** 绑定类型 */
export const BIND_TYPE_LABELS: Record<string, string> = {
  none: '不绑定',
  pickup_bind: '获得绑定',
  equip_bind: '装备绑定',
};

/**
 * 将枚举配置值转为中文展示文案；未知值原样返回，便于排查。
 */
export function formatDisplayLabel(labels: Record<string, string>, value: string | null | undefined): string {
  if (!value) {
    return '-';
  }
  return labels[value] ?? value;
}

/**
 * 根据枚举映射生成 Select 选项（value 仍为英文配置值）。
 */
export function buildSelectOptions(labels: Record<string, string>): Array<{ label: string; value: string }> {
  return Object.entries(labels).map(([value, label]) => ({ label, value }));
}

/**
 * 生成带「全部」项的筛选下拉选项。
 */
export function buildFilterSelectOptions(
  labels: Record<string, string>,
  allLabel: string = '全部',
): Array<{ label: string; value: string }> {
  return [{ label: allLabel, value: '' }, ...buildSelectOptions(labels)];
}
