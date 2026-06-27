import { fetchAdminEquipmentDefinitions } from '../services/equipmentDefinition';
import { fetchAdminItems } from '../services/item';
import type { AdminEquipmentSummary } from '../types/equipmentDefinition';
import type { AdminItemSummary } from '../types/item';

export type GrantableItemCategory = 'all' | 'equipment' | 'other';

export interface GrantableItemOption {
  item_id: number;
  item_name: string;
  item_type: string;
  quality: number;
}

/** 按关键词搜索可发放物品候选项，逻辑与玩家背包发放一致。 */
export async function searchGrantableItemOptions(
  keyword: string,
  category: GrantableItemCategory,
  preferredItems: GrantableItemOption[] = [],
): Promise<GrantableItemOption[]> {
  const trimmedKeyword = keyword.trim() || undefined;
  let nextItems: GrantableItemOption[] = [];
  if (category === 'equipment') {
    const result = await fetchAdminEquipmentDefinitions({
      filters: { keyword: trimmedKeyword, is_enabled: 'true' },
      page: 1,
      pageSize: 20,
    });
    nextItems = result.items.map(mapEquipmentSummaryToGrantableOption);
  } else if (category === 'other') {
    const result = await fetchAdminItems({
      filters: { keyword: trimmedKeyword, enabled: 'true', exclude_item_type: 'equipment' },
      page: 1,
      pageSize: 20,
    });
    nextItems = result.items.map(mapItemSummaryToGrantableOption);
  } else {
    const [itemResult, equipmentResult] = await Promise.all([
      fetchAdminItems({
        filters: { keyword: trimmedKeyword, enabled: 'true', exclude_item_type: 'equipment' },
        page: 1,
        pageSize: 20,
      }),
      fetchAdminEquipmentDefinitions({
        filters: { keyword: trimmedKeyword, is_enabled: 'true' },
        page: 1,
        pageSize: 20,
      }),
    ]);
    nextItems = [
      ...itemResult.items.map(mapItemSummaryToGrantableOption),
      ...equipmentResult.items.map(mapEquipmentSummaryToGrantableOption),
    ];
  }
  return mergeGrantableItemOptions(nextItems, preferredItems);
}

export function mapItemSummaryToGrantableOption(item: AdminItemSummary): GrantableItemOption {
  return {
    item_id: item.item_id,
    item_name: item.item_name,
    item_type: item.item_type,
    quality: item.quality,
  };
}

export function mapEquipmentSummaryToGrantableOption(item: AdminEquipmentSummary): GrantableItemOption {
  return {
    item_id: item.item_id,
    item_name: item.item_name,
    item_type: 'equipment',
    quality: item.quality,
  };
}

export function deduplicateGrantableOptions(items: GrantableItemOption[]): GrantableItemOption[] {
  const nextMap = new Map<number, GrantableItemOption>();
  items.forEach((item) => {
    if (!nextMap.has(item.item_id)) {
      nextMap.set(item.item_id, item);
    }
  });
  return Array.from(nextMap.values());
}

export function mergeGrantableItemOptions(
  items: GrantableItemOption[],
  preferredItems: GrantableItemOption[] = [],
): GrantableItemOption[] {
  return deduplicateGrantableOptions([...preferredItems, ...items]);
}

export function findGrantableItemOption(
  items: GrantableItemOption[],
  itemID?: number,
): GrantableItemOption | undefined {
  if (!itemID) {
    return undefined;
  }
  return items.find((item) => item.item_id === itemID);
}
