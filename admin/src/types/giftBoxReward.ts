/** 礼包开启后写入 effect_params_json.rewards 的单条奖励配置。 */
export interface GiftBoxRewardEntry {
  type: 'item' | 'gold' | 'pet';
  item_id?: number;
  item_name?: string;
  count?: number;
  value?: number;
  pet_id?: number;
  pet_name?: string;
}

/** 非礼包类物品在后台表单里展示的使用行为。 */
export type ItemUseBehavior =
  | 'none'
  | 'pet_hp_restore'
  | 'bag_expand'
  | 'warehouse_expand'
  | 'pet_talisman_slot_unlock';

export const ITEM_USE_BEHAVIOR_OPTIONS: Array<{ value: ItemUseBehavior; label: string }> = [
  { value: 'none', label: '无使用效果' },
  { value: 'pet_hp_restore', label: '恢复宠物生命' },
  { value: 'bag_expand', label: '背包扩容' },
  { value: 'warehouse_expand', label: '仓库扩容' },
  { value: 'pet_talisman_slot_unlock', label: '解锁宠物神符槽' },
];

export const GIFT_BOX_REWARD_TYPE_OPTIONS = [
  { value: 'item', label: '物品' },
  { value: 'gold', label: '金币（铜币）' },
  { value: 'pet', label: '宠物' },
] as const;

export function isGiftBoxItemType(itemType: string, effectType?: string): boolean {
  if (itemType === 'box') {
    return true;
  }
  const normalizedEffectType = (effectType ?? '').trim();
  return normalizedEffectType === 'reward_box'
    || normalizedEffectType === 'gift_box'
    || normalizedEffectType === 'box_open';
}

/** 从 effect_params_json 解析礼包奖励列表。 */
export function parseGiftBoxRewards(effectParamsJSON: string | undefined): GiftBoxRewardEntry[] {
  if (!effectParamsJSON || effectParamsJSON.trim() === '') {
    return [];
  }
  try {
    const payload = JSON.parse(effectParamsJSON) as { rewards?: unknown };
    if (!Array.isArray(payload.rewards)) {
      return [];
    }
    return payload.rewards
      .map((entry) => normalizeGiftBoxRewardEntry(entry))
      .filter((entry): entry is GiftBoxRewardEntry => entry !== null);
  } catch {
    return [];
  }
}

/** 将礼包奖励列表序列化为服务端消费的 JSON 字符串。 */
export function serializeGiftBoxRewards(rewards: GiftBoxRewardEntry[]): string {
  const normalizedRewards = rewards
    .map((entry) => normalizeGiftBoxRewardEntry(entry))
    .filter((entry): entry is GiftBoxRewardEntry => entry !== null)
    .map((entry) => {
      if (entry.type === 'gold') {
        return {
          type: 'gold',
          value: Number(entry.value ?? 0),
        };
      }
      if (entry.type === 'pet') {
        return {
          type: 'pet',
          pet_id: Number(entry.pet_id ?? 0),
          pet_name: entry.pet_name ?? '',
        };
      }
      return {
        type: 'item',
        item_id: Number(entry.item_id ?? 0),
        item_name: entry.item_name ?? '',
        count: Number(entry.count ?? 1),
      };
    });
  return JSON.stringify({ rewards: normalizedRewards });
}

/** 根据数据库 effect 字段反推后台使用行为。 */
export function resolveItemUseBehavior(effectType: string, effectValue: number): ItemUseBehavior {
  switch ((effectType ?? '').trim()) {
    case 'pet_hp_restore':
      return 'pet_hp_restore';
    case 'bag_expand':
    case 'expand':
      return 'bag_expand';
    case 'warehouse_expand':
      return 'warehouse_expand';
    case 'pet_talisman_slot_unlock':
      return 'pet_talisman_slot_unlock';
    default:
      return 'none';
  }
}

/** 使用行为对应的数值含义（恢复量 / 扩容格数）。 */
export function resolveItemUseAmount(effectType: string, effectValue: number, effectParamsJSON: string): number {
  if (effectValue > 0) {
    return effectValue;
  }
  try {
    const payload = JSON.parse(effectParamsJSON || '{}') as { expand_slots?: number };
    return Number(payload.expand_slots ?? 0);
  } catch {
    return 0;
  }
}

function normalizeGiftBoxRewardEntry(entry: unknown): GiftBoxRewardEntry | null {
  if (!entry || typeof entry !== 'object') {
    return null;
  }
  const record = entry as Record<string, unknown>;
  const rewardType = String(record.type ?? '').trim();
  if (rewardType === 'gold') {
    const value = Number(record.value ?? 0);
    if (value <= 0) {
      return null;
    }
    return { type: 'gold', value };
  }
  if (rewardType === 'item') {
    const itemID = Number(record.item_id ?? 0);
    const count = Number(record.count ?? 1);
    if (itemID <= 0 || count <= 0) {
      return null;
    }
    return {
      type: 'item',
      item_id: itemID,
      item_name: String(record.item_name ?? ''),
      count,
    };
  }
  if (rewardType === 'pet') {
    const petID = Number(record.pet_id ?? 0);
    if (petID <= 0) {
      return null;
    }
    return {
      type: 'pet',
      pet_id: petID,
      pet_name: String(record.pet_name ?? ''),
    };
  }
  return null;
}
