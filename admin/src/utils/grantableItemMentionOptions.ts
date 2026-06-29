import { fetchAdminEquipmentDefinitions } from '../services/equipmentDefinition';
import { fetchAdminItems } from '../services/item';
import type { GrantableItemCategory, GrantableItemOption } from './grantableItemOptions';
import {
  deduplicateGrantableOptions,
  mapEquipmentSummaryToGrantableOption,
  mapItemSummaryToGrantableOption,
} from './grantableItemOptions';

/** 后台富文本编辑器插入占位符时使用的物品候选项（含 icon）。 */
export interface GrantableItemMentionOption extends GrantableItemOption {
  icon: string;
  desc: string;
}

/** 按关键词搜索可插入 mention 的物品/装备候选项。 */
export async function searchGrantableItemMentionOptions(
  keyword: string,
  category: GrantableItemCategory,
  preferredItems: GrantableItemMentionOption[] = [],
): Promise<GrantableItemMentionOption[]> {
  const trimmedKeyword = keyword.trim() || undefined;
  let nextItems: GrantableItemMentionOption[] = [];
  if (category === 'equipment') {
    const result = await fetchAdminEquipmentDefinitions({
      filters: { keyword: trimmedKeyword, is_enabled: 'true' },
      page: 1,
      pageSize: 48,
    });
    nextItems = result.items.map((item) => ({
      ...mapEquipmentSummaryToGrantableOption(item),
      icon: '',
      desc: '',
    }));
  } else if (category === 'other') {
    const result = await fetchAdminItems({
      filters: { keyword: trimmedKeyword, enabled: 'true', exclude_item_type: 'equipment' },
      page: 1,
      pageSize: 48,
    });
    nextItems = result.items.map((item) => ({
      ...mapItemSummaryToGrantableOption(item),
      icon: item.icon ?? '',
      desc: item.desc ?? '',
    }));
  } else {
    const [itemResult, equipmentResult] = await Promise.all([
      fetchAdminItems({
        filters: { keyword: trimmedKeyword, enabled: 'true', exclude_item_type: 'equipment' },
        page: 1,
        pageSize: 24,
      }),
      fetchAdminEquipmentDefinitions({
        filters: { keyword: trimmedKeyword, is_enabled: 'true' },
        page: 1,
        pageSize: 24,
      }),
    ]);
    nextItems = [
      ...itemResult.items.map((item) => ({
        ...mapItemSummaryToGrantableOption(item),
        icon: item.icon ?? '',
        desc: item.desc ?? '',
      })),
      ...equipmentResult.items.map((item) => ({
        ...mapEquipmentSummaryToGrantableOption(item),
        icon: '',
        desc: '',
      })),
    ];
  }
  const merged = deduplicateGrantableOptions([...preferredItems, ...nextItems]) as GrantableItemMentionOption[];
  const preferredMap = new Map(preferredItems.map((item) => [item.item_id, item]));
  return merged.map((item) => preferredMap.get(item.item_id) ?? item);
}
