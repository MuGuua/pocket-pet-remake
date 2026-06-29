import type { AdminItemDetail, AdminUpsertItemPayload } from '../types/item';
import type { GiftBoxRewardEntry } from '../types/giftBoxReward';
import type { ConsumableEffectEntry } from '../types/consumableEffect';
import type { EnhanceMaterialConfig } from '../types/enhanceMaterialConfig';
import {
  defaultEnhanceMaterialConfig,
  normalizeEnhanceMaterialConfig,
} from '../types/enhanceMaterialConfig';
import {
  formatConsumableEffectSummary,
  inferConsumableUseTarget,
  parseConsumableEffects,
  serializeConsumableEffects,
} from '../types/consumableEffect';
import {
  isGiftBoxItemType,
  parseGiftBoxRewards,
  serializeGiftBoxRewards,
} from '../types/giftBoxReward';

export interface ItemEditorFormValues extends AdminUpsertItemPayload {
  use_effects?: ConsumableEffectEntry[];
  gift_rewards?: GiftBoxRewardEntry[];
  enhance_material_config?: EnhanceMaterialConfig;
}

export function mapItemDetailToFormValues(detail: AdminItemDetail): ItemEditorFormValues {
  const giftBoxItem = isGiftBoxItemType(detail.item_type, detail.effect_type);
  return {
    ...detail,
    use_effects: giftBoxItem
      ? []
      : parseConsumableEffects(detail.effect_type, detail.effect_value, detail.effect_params_json),
    gift_rewards: giftBoxItem ? parseGiftBoxRewards(detail.effect_params_json) : [],
    enhance_material_config: detail.item_sub_type === 'equipment_enhance'
      ? normalizeEnhanceMaterialConfig(detail.enhance_material_config ?? defaultEnhanceMaterialConfig())
      : defaultEnhanceMaterialConfig(),
  };
}

export function mapItemFormToPayload(values: ItemEditorFormValues): AdminUpsertItemPayload {
  const itemType = values.item_type ?? '';
  const giftBoxItem = itemType === 'box';
  const itemSubType = giftBoxItem ? 'gift_box' : String(values.item_sub_type ?? '').trim();
  const giftRewards = values.gift_rewards ?? [];
  const useEffects = values.use_effects ?? [];

  let effectType = '';
  let effectValue = 0;
  let effectParamsJSON = '{}';
  let usable = Boolean(values.usable);
  let useScope = values.use_scope ?? '';
  let targetType = values.target_type ?? '';

  if (giftBoxItem) {
    effectType = 'reward_box';
    effectValue = 0;
    effectParamsJSON = serializeGiftBoxRewards(giftRewards);
    usable = true;
    useScope = 'world';
    targetType = 'self';
  } else {
    const serializedEffects = serializeConsumableEffects(useEffects);
    effectType = serializedEffects.effectType;
    effectValue = serializedEffects.effectValue;
    effectParamsJSON = serializedEffects.effectParamsJSON;
    const inferredTarget = inferConsumableUseTarget(useEffects);
    if (useEffects.length > 0) {
      usable = true;
      useScope = inferredTarget.useScope;
      targetType = inferredTarget.targetType;
    } else {
      usable = false;
      useScope = '';
      targetType = '';
    }
  }

  return {
    item_id: Number(values.item_id ?? 0),
    item_code: values.item_code ?? '',
    item_name: values.item_name ?? '',
    item_type: itemType,
    item_sub_type: itemSubType,
    quality: Number(values.quality ?? 1),
    rarity: Number(values.rarity ?? 1),
    icon: values.icon ?? '',
    desc: values.desc ?? '',
    max_stack: Number(values.max_stack ?? 1),
    occupy_slots: Number(values.occupy_slots ?? 1),
    auto_merge: Boolean(values.auto_merge),
    sort_weight: Number(values.sort_weight ?? 0),
    usable,
    use_scope: useScope,
    target_type: targetType,
    required_level: Number(values.required_level ?? 0),
    required_scene_id: Number(values.required_scene_id ?? 0),
    bind_type: values.bind_type ?? 'none',
    can_sell: Boolean(values.can_sell),
    can_drop: Boolean(values.can_drop),
    can_store: Boolean(values.can_store),
    can_trade: Boolean(values.can_trade),
    expire_at_rule: values.expire_at_rule ?? '',
    effect_type: effectType,
    effect_value: effectValue,
    effect_params_json: effectParamsJSON,
    buy_price_copper: Number(values.buy_price_copper ?? 0),
    sell_price_copper: Number(values.sell_price_copper ?? 0),
    recycle_price_copper: Number(values.recycle_price_copper ?? 0),
    price_type: values.price_type ?? 'base_coin',
    is_enabled: Boolean(values.is_enabled),
    enhance_material_config: itemSubType === 'equipment_enhance'
      ? normalizeEnhanceMaterialConfig(values.enhance_material_config ?? defaultEnhanceMaterialConfig())
      : undefined,
  };
}

export function formatGiftRewardSummary(rewards: GiftBoxRewardEntry[]): string {
  if (rewards.length === 0) {
    return '未配置礼包内容';
  }
  return rewards
    .map((entry) => {
      if (entry.type === 'gold') {
        return `铜币×${entry.value ?? 0}`;
      }
      if (entry.type === 'pet') {
        const petName = entry.pet_name?.trim() || `宠物${entry.pet_id ?? 0}`;
        return petName;
      }
      const name = entry.item_name?.trim() || `物品${entry.item_id ?? 0}`;
      return `${name}×${entry.count ?? 1}`;
    })
    .join('、');
}

export { formatConsumableEffectSummary };
