import { ADMIN_PET_COMBAT_STAT_FIELDS } from './petCombatStats';

/** 消耗品使用效果的大类，对应后台表单第一步选择。 */
export type ConsumableEffectCategory = 'player' | 'pet' | 'equipment' | 'system' | 'other';

/** 数值类效果支持增加、减少或设值；开关类效果仅支持设值。 */
export type ConsumableEffectOperation = 'add' | 'subtract' | 'set';

export type ConsumableEffectValueType = 'number' | 'boolean';

/** 单条消耗品使用效果配置，序列化后写入 effect_params_json.use_effects。 */
export interface ConsumableEffectEntry {
  category: ConsumableEffectCategory;
  field_key: string;
  operation: ConsumableEffectOperation;
  value: number | boolean;
}

export interface ConsumableEffectFieldDefinition {
  key: string;
  label: string;
  group: string;
  valueType: ConsumableEffectValueType;
  operations?: ConsumableEffectOperation[];
}

export const CONSUMABLE_EFFECT_CATEGORY_OPTIONS: Array<{ value: ConsumableEffectCategory; label: string }> = [
  { value: 'player', label: '人物' },
  { value: 'pet', label: '宠物' },
  { value: 'equipment', label: '装备' },
  { value: 'system', label: '系统' },
  { value: 'other', label: '其他' },
];

export const CONSUMABLE_EFFECT_OPERATION_OPTIONS: Array<{ value: ConsumableEffectOperation; label: string }> = [
  { value: 'add', label: '增加' },
  { value: 'subtract', label: '减少' },
  { value: 'set', label: '设为' },
];

const PLAYER_BASIC_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'level', label: '等级', group: '基础属性', valueType: 'number' },
  { key: 'exp', label: '经验', group: '基础属性', valueType: 'number' },
  { key: 'free_attr_points', label: '自由属性点', group: '基础属性', valueType: 'number' },
  { key: 'strength', label: '力量', group: '基础属性', valueType: 'number' },
  { key: 'vitality', label: '体质', group: '基础属性', valueType: 'number' },
  { key: 'agility', label: '敏捷', group: '基础属性', valueType: 'number' },
  { key: 'mind', label: '灵力', group: '基础属性', valueType: 'number' },
];

const PLAYER_WALLET_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'gold', label: '金币', group: '钱币', valueType: 'number' },
  { key: 'total_copper', label: '钱包总铜币', group: '钱币', valueType: 'number' },
];

const PLAYER_RESOURCE_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'hp', label: '生命', group: '生命与精力', valueType: 'number' },
  { key: 'hp_max', label: '生命上限', group: '生命与精力', valueType: 'number' },
  { key: 'vigor', label: '活力', group: '生命与精力', valueType: 'number' },
  { key: 'vigor_max', label: '活力上限', group: '生命与精力', valueType: 'number' },
  { key: 'spirit', label: '精力', group: '生命与精力', valueType: 'number' },
  { key: 'spirit_max', label: '精力上限', group: '生命与精力', valueType: 'number' },
];

const PLAYER_COMBAT_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'atk', label: '攻击', group: '战斗属性', valueType: 'number' },
  { key: 'def', label: '防御', group: '战斗属性', valueType: 'number' },
  { key: 'spd', label: '速度', group: '战斗属性', valueType: 'number' },
  { key: 'mana', label: '法力', group: '战斗属性', valueType: 'number' },
  { key: 'hit_pct', label: '命中', group: '战斗百分比', valueType: 'number' },
  { key: 'dodge_pct', label: '闪避', group: '战斗百分比', valueType: 'number' },
  { key: 'crit_rate_pct', label: '致命', group: '战斗百分比', valueType: 'number' },
  { key: 'crit_dmg_pct', label: '爆伤', group: '战斗百分比', valueType: 'number' },
  { key: 'physical_resist_pct', label: '物抗', group: '战斗百分比', valueType: 'number' },
  { key: 'skill_resist_pct', label: '技抗', group: '战斗百分比', valueType: 'number' },
  { key: 'confusion_resist_pct', label: '混乱抗性', group: '战斗百分比', valueType: 'number' },
  { key: 'sleep_resist_pct', label: '昏睡抗性', group: '战斗百分比', valueType: 'number' },
  { key: 'paralysis_resist_pct', label: '麻痹抗性', group: '战斗百分比', valueType: 'number' },
  { key: 'seal_resist_pct', label: '封印抗性', group: '战斗百分比', valueType: 'number' },
  { key: 'curse_resist_pct', label: '诅咒抗性', group: '战斗百分比', valueType: 'number' },
  { key: 'crit_resist_pct', label: '抗致命', group: '战斗百分比', valueType: 'number' },
  { key: 'crit_dmg_resist_pct', label: '抗爆伤', group: '战斗百分比', valueType: 'number' },
  { key: 'character_resist_pct', label: '抗人物', group: '战斗百分比', valueType: 'number' },
  { key: 'pet_resist_pct', label: '抗宠物', group: '战斗百分比', valueType: 'number' },
  { key: 'mercenary_resist_pct', label: '抗佣兵', group: '战斗百分比', valueType: 'number' },
  { key: 'generic_shield_pct', label: '通用护盾', group: '战斗百分比', valueType: 'number' },
];

const PLAYER_SCENE_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'scene_id', label: '场景ID', group: '场景位置', valueType: 'number', operations: ['set'] },
  { key: 'pos_x', label: 'X 坐标', group: '场景位置', valueType: 'number', operations: ['set'] },
  { key: 'pos_y', label: 'Y 坐标', group: '场景位置', valueType: 'number', operations: ['set'] },
  { key: 'status', label: '账号状态', group: '状态', valueType: 'number', operations: ['set'] },
];

const PET_BASIC_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'level', label: '等级', group: '基础属性', valueType: 'number' },
  { key: 'exp', label: '经验', group: '基础属性', valueType: 'number' },
  { key: 'quality', label: '品质', group: '基础属性', valueType: 'number' },
  { key: 'hp', label: '生命', group: '生命与战斗', valueType: 'number' },
  { key: 'hp_max', label: '生命上限', group: '生命与战斗', valueType: 'number' },
  { key: 'atk', label: '攻击', group: '生命与战斗', valueType: 'number' },
  { key: 'def', label: '防御', group: '生命与战斗', valueType: 'number' },
  { key: 'spd', label: '速度', group: '生命与战斗', valueType: 'number' },
  { key: 'mana', label: '法力', group: '生命与战斗', valueType: 'number' },
];

const PET_COMBAT_FIELDS: ConsumableEffectFieldDefinition[] = ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => ({
  key: field.key,
  label: field.label,
  group: '战斗百分比',
  valueType: 'number' as const,
}));

const EQUIPMENT_INSTANCE_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'enhance_level', label: '强化等级', group: '实例属性', valueType: 'number' },
  { key: 'max_enhance_level', label: '最大强化等级', group: '强化配置', valueType: 'number', operations: ['set'] },
];

const SYSTEM_FIELDS: ConsumableEffectFieldDefinition[] = [
  { key: 'bag_capacity_expand', label: '背包扩容格数', group: '容器', valueType: 'number', operations: ['add'] },
  { key: 'warehouse_capacity_expand', label: '仓库扩容格数', group: '容器', valueType: 'number', operations: ['add'] },
  { key: 'pet_talisman_slot_unlock', label: '解锁宠物神符槽', group: '宠物系统', valueType: 'boolean', operations: ['set'] },
];

export const CONSUMABLE_EFFECT_FIELD_CATALOG: Record<ConsumableEffectCategory, ConsumableEffectFieldDefinition[]> = {
  player: [
    ...PLAYER_BASIC_FIELDS,
    ...PLAYER_WALLET_FIELDS,
    ...PLAYER_RESOURCE_FIELDS,
    ...PLAYER_COMBAT_FIELDS,
    ...PLAYER_SCENE_FIELDS,
  ],
  pet: [...PET_BASIC_FIELDS, ...PET_COMBAT_FIELDS],
  equipment: EQUIPMENT_INSTANCE_FIELDS,
  system: SYSTEM_FIELDS,
  other: [],
};

/** 根据大类与字段 key 查找字段定义；其他类允许自定义 key。 */
export function resolveConsumableEffectField(
  category: ConsumableEffectCategory,
  fieldKey: string,
): ConsumableEffectFieldDefinition | null {
  if (category === 'other') {
    const normalizedKey = fieldKey.trim();
    if (normalizedKey === '') {
      return null;
    }
    return {
      key: normalizedKey,
      label: normalizedKey,
      group: '自定义',
      valueType: 'number',
    };
  }
  return CONSUMABLE_EFFECT_FIELD_CATALOG[category].find((field) => field.key === fieldKey) ?? null;
}

/** 将字段列表转为 Select 分组选项。 */
export function buildConsumableEffectFieldOptions(category: ConsumableEffectCategory) {
  const fields = CONSUMABLE_EFFECT_FIELD_CATALOG[category];
  const groupMap = new Map<string, Array<{ value: string; label: string }>>();
  fields.forEach((field) => {
    const groupItems = groupMap.get(field.group) ?? [];
    groupItems.push({ value: field.key, label: field.label });
    groupMap.set(field.group, groupItems);
  });
  return Array.from(groupMap.entries()).map(([groupLabel, options]) => ({
    label: groupLabel,
    options,
  }));
}

/** 从数据库 effect 字段反推效果列表，兼容旧版单一 use_behavior 配置。 */
export function parseConsumableEffects(
  effectType: string,
  effectValue: number,
  effectParamsJSON: string | undefined,
): ConsumableEffectEntry[] {
  if (effectParamsJSON && effectParamsJSON.trim() !== '') {
    try {
      const payload = JSON.parse(effectParamsJSON) as { use_effects?: unknown };
      if (Array.isArray(payload.use_effects)) {
        return payload.use_effects
          .map((entry) => normalizeConsumableEffectEntry(entry))
          .filter((entry): entry is ConsumableEffectEntry => entry !== null);
      }
    } catch {
      // 继续尝试旧版 effect_type 反推
    }
  }
  return legacyEffectToEntries(effectType, effectValue, effectParamsJSON);
}

/** 将效果列表序列化为服务端存储 JSON，并在可能时保留旧版 effect_type 兼容字段。 */
export function serializeConsumableEffects(effects: ConsumableEffectEntry[]): {
  effectType: string;
  effectValue: number;
  effectParamsJSON: string;
} {
  const normalizedEffects = effects
    .map((entry) => normalizeConsumableEffectEntry(entry))
    .filter((entry): entry is ConsumableEffectEntry => entry !== null);

  if (normalizedEffects.length === 0) {
    return {
      effectType: '',
      effectValue: 0,
      effectParamsJSON: '{}',
    };
  }

  const legacyMapping = mapSingleLegacyEffect(normalizedEffects);
  const payload: Record<string, unknown> = {
    version: 1,
    use_effects: normalizedEffects.map((entry) => ({
      category: entry.category,
      field_key: entry.field_key,
      operation: entry.operation,
      value: entry.value,
    })),
  };

  if (legacyMapping) {
    Object.assign(payload, legacyMapping.legacyParams);
    return {
      effectType: legacyMapping.effectType,
      effectValue: legacyMapping.effectValue,
      effectParamsJSON: JSON.stringify(payload),
    };
  }

  return {
    effectType: 'use_effects',
    effectValue: 0,
    effectParamsJSON: JSON.stringify(payload),
  };
}

/** 根据效果列表推断使用范围与目标类型。 */
export function inferConsumableUseTarget(effects: ConsumableEffectEntry[]): {
  useScope: string;
  targetType: string;
  usable: boolean;
} {
  if (effects.length === 0) {
    return { useScope: '', targetType: '', usable: false };
  }
  const requiresPetTarget = effects.some((entry) => (
    entry.category === 'pet'
    || entry.field_key === 'pet_talisman_slot_unlock'
  ));
  const requiresEquipmentTarget = effects.some((entry) => entry.category === 'equipment');
  if (requiresPetTarget) {
    return { useScope: 'world', targetType: 'pet_single', usable: true };
  }
  if (requiresEquipmentTarget) {
    return { useScope: 'world', targetType: 'equipment_single', usable: true };
  }
  return { useScope: 'world', targetType: 'self', usable: true };
}

export function formatConsumableEffectCategoryLabel(category: ConsumableEffectCategory): string {
  return CONSUMABLE_EFFECT_CATEGORY_OPTIONS.find((option) => option.value === category)?.label ?? category;
}

export function formatConsumableEffectOperationLabel(operation: ConsumableEffectOperation): string {
  return CONSUMABLE_EFFECT_OPERATION_OPTIONS.find((option) => option.value === operation)?.label ?? operation;
}

export function formatConsumableEffectEntryLabel(entry: ConsumableEffectEntry): string {
  const categoryLabel = formatConsumableEffectCategoryLabel(entry.category);
  const field = resolveConsumableEffectField(entry.category, entry.field_key);
  const fieldLabel = field?.label ?? entry.field_key;
  const operationLabel = formatConsumableEffectOperationLabel(entry.operation);
  if (field?.valueType === 'boolean') {
    return `${categoryLabel} · ${fieldLabel} · ${entry.value ? '启用' : '关闭'}`;
  }
  return `${categoryLabel} · ${fieldLabel} · ${operationLabel} ${entry.value}`;
}

export function formatConsumableEffectSummary(effects: ConsumableEffectEntry[]): string {
  if (effects.length === 0) {
    return '未配置使用效果';
  }
  return effects.map((entry) => formatConsumableEffectEntryLabel(entry)).join('；');
}

function normalizeConsumableEffectEntry(entry: unknown): ConsumableEffectEntry | null {
  if (!entry || typeof entry !== 'object') {
    return null;
  }
  const record = entry as Record<string, unknown>;
  const category = String(record.category ?? '').trim() as ConsumableEffectCategory;
  if (!CONSUMABLE_EFFECT_CATEGORY_OPTIONS.some((option) => option.value === category)) {
    return null;
  }
  const fieldKey = String(record.field_key ?? '').trim();
  if (fieldKey === '') {
    return null;
  }
  const field = resolveConsumableEffectField(category, fieldKey);
  if (!field) {
    return null;
  }
  const operation = String(record.operation ?? 'add').trim() as ConsumableEffectOperation;
  if (operation !== 'add' && operation !== 'subtract' && operation !== 'set') {
    return null;
  }
  if (field.valueType === 'boolean') {
    return {
      category,
      field_key: fieldKey,
      operation: 'set',
      value: Boolean(record.value),
    };
  }
  const numericValue = Number(record.value ?? 0);
  if (!Number.isFinite(numericValue)) {
    return null;
  }
  const allowedOperations = field.operations ?? ['add', 'subtract', 'set'];
  const normalizedOperation = allowedOperations.includes(operation) ? operation : allowedOperations[0];
  if ((normalizedOperation === 'add' || normalizedOperation === 'subtract') && numericValue === 0) {
    return null;
  }
  return {
    category,
    field_key: fieldKey,
    operation: normalizedOperation,
    value: Math.trunc(numericValue),
  };
}

function legacyEffectToEntries(
  effectType: string,
  effectValue: number,
  effectParamsJSON: string | undefined,
): ConsumableEffectEntry[] {
  const normalizedEffectType = (effectType ?? '').trim();
  switch (normalizedEffectType) {
    case 'pet_hp_restore': {
      const restoreValue = effectValue > 0 ? effectValue : resolveLegacyExpandSlots(effectParamsJSON);
      if (restoreValue <= 0) {
        return [];
      }
      return [{
        category: 'pet',
        field_key: 'hp',
        operation: 'add',
        value: restoreValue,
      }];
    }
    case 'bag_expand':
    case 'expand': {
      const expandSlots = effectValue > 0 ? effectValue : resolveLegacyExpandSlots(effectParamsJSON);
      if (expandSlots <= 0) {
        return [];
      }
      return [{
        category: 'system',
        field_key: 'bag_capacity_expand',
        operation: 'add',
        value: expandSlots,
      }];
    }
    case 'warehouse_expand': {
      const expandSlots = effectValue > 0 ? effectValue : resolveLegacyExpandSlots(effectParamsJSON);
      if (expandSlots <= 0) {
        return [];
      }
      return [{
        category: 'system',
        field_key: 'warehouse_capacity_expand',
        operation: 'add',
        value: expandSlots,
      }];
    }
    case 'pet_talisman_slot_unlock':
      return [{
        category: 'system',
        field_key: 'pet_talisman_slot_unlock',
        operation: 'set',
        value: true,
      }];
    case 'use_effects':
      return [];
    default:
      return [];
  }
}

function resolveLegacyExpandSlots(effectParamsJSON: string | undefined): number {
  if (!effectParamsJSON || effectParamsJSON.trim() === '') {
    return 0;
  }
  try {
    const payload = JSON.parse(effectParamsJSON) as { expand_slots?: number };
    return Number(payload.expand_slots ?? 0);
  } catch {
    return 0;
  }
}

interface LegacyEffectMapping {
  effectType: string;
  effectValue: number;
  legacyParams: Record<string, unknown>;
}

function mapSingleLegacyEffect(effects: ConsumableEffectEntry[]): LegacyEffectMapping | null {
  if (effects.length !== 1) {
    return null;
  }
  const entry = effects[0];
  if (entry.category === 'pet' && entry.field_key === 'hp' && entry.operation === 'add' && typeof entry.value === 'number') {
    return {
      effectType: 'pet_hp_restore',
      effectValue: entry.value,
      legacyParams: { restore_type: 'flat', target: 'pet_single' },
    };
  }
  if (entry.category === 'system' && entry.field_key === 'bag_capacity_expand' && entry.operation === 'add' && typeof entry.value === 'number') {
    return {
      effectType: 'bag_expand',
      effectValue: entry.value,
      legacyParams: { expand_target: 'bag', expand_slots: entry.value },
    };
  }
  if (entry.category === 'system' && entry.field_key === 'warehouse_capacity_expand' && entry.operation === 'add' && typeof entry.value === 'number') {
    return {
      effectType: 'warehouse_expand',
      effectValue: entry.value,
      legacyParams: { expand_target: 'warehouse', expand_slots: entry.value },
    };
  }
  if (entry.category === 'system' && entry.field_key === 'pet_talisman_slot_unlock' && entry.value === true) {
    return {
      effectType: 'pet_talisman_slot_unlock',
      effectValue: 0,
      legacyParams: {},
    };
  }
  return null;
}
