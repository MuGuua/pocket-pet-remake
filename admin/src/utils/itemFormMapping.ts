import type { AdminItemDetail, AdminUpsertItemPayload } from '../types/item';
import type { GiftBoxRewardEntry, ItemUseBehavior } from '../types/giftBoxReward';
import {
  isGiftBoxItemType,
  parseGiftBoxRewards,
  resolveItemUseAmount,
  resolveItemUseBehavior,
  serializeGiftBoxRewards,
} from '../types/giftBoxReward';

export interface ItemEditorFormValues extends AdminUpsertItemPayload {
  use_behavior?: ItemUseBehavior;
  use_amount?: number;
  gift_rewards?: GiftBoxRewardEntry[];
}

export function mapItemDetailToFormValues(detail: AdminItemDetail): ItemEditorFormValues {
  const giftBoxItem = isGiftBoxItemType(detail.item_type, detail.effect_type);
  return {
    ...detail,
    use_behavior: giftBoxItem ? 'none' : resolveItemUseBehavior(detail.effect_type, detail.effect_value),
    use_amount: giftBoxItem
      ? 0
      : resolveItemUseAmount(detail.effect_type, detail.effect_value, detail.effect_params_json),
    gift_rewards: giftBoxItem ? parseGiftBoxRewards(detail.effect_params_json) : [],
  };
}

export function mapItemFormToPayload(values: ItemEditorFormValues): AdminUpsertItemPayload {
  const itemType = values.item_type ?? '';
  const giftBoxItem = itemType === 'box';
  const giftRewards = values.gift_rewards ?? [];
  const useBehavior = values.use_behavior ?? 'none';
  const useAmount = Number(values.use_amount ?? 0);

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
    switch (useBehavior) {
      case 'pet_hp_restore':
        effectType = 'pet_hp_restore';
        effectValue = useAmount;
        effectParamsJSON = JSON.stringify({ restore_type: 'flat', target: 'pet_single' });
        usable = true;
        useScope = useScope || 'world';
        targetType = targetType || 'pet_single';
        break;
      case 'bag_expand':
        effectType = 'bag_expand';
        effectValue = useAmount;
        effectParamsJSON = JSON.stringify({ expand_target: 'bag', expand_slots: useAmount });
        usable = true;
        useScope = useScope || 'world';
        targetType = targetType || 'self';
        break;
      case 'warehouse_expand':
        effectType = 'warehouse_expand';
        effectValue = useAmount;
        effectParamsJSON = JSON.stringify({ expand_target: 'warehouse', expand_slots: useAmount });
        usable = true;
        useScope = useScope || 'world';
        targetType = targetType || 'self';
        break;
      case 'pet_talisman_slot_unlock':
        effectType = 'pet_talisman_slot_unlock';
        effectValue = 0;
        effectParamsJSON = '{}';
        usable = true;
        useScope = useScope || 'world';
        targetType = targetType || 'pet_single';
        break;
      default:
        effectType = '';
        effectValue = 0;
        effectParamsJSON = '{}';
        break;
    }
  }

  return {
    item_id: Number(values.item_id ?? 0),
    item_code: values.item_code ?? '',
    item_name: values.item_name ?? '',
    item_type: itemType,
    item_sub_type: giftBoxItem ? 'gift_box' : String(values.item_sub_type ?? '').trim(),
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
  };
}

export function formatUseBehaviorLabel(behavior: ItemUseBehavior): string {
  switch (behavior) {
    case 'pet_hp_restore':
      return '恢复宠物生命';
    case 'bag_expand':
      return '背包扩容';
    case 'warehouse_expand':
      return '仓库扩容';
    case 'pet_talisman_slot_unlock':
      return '解锁宠物神符槽';
    default:
      return '无使用效果';
  }
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
      const name = entry.item_name?.trim() || `物品${entry.item_id ?? 0}`;
      return `${name}×${entry.count ?? 1}`;
    })
    .join('、');
}
