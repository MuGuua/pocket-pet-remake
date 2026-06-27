import { Segmented, Select, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import {
  findGrantableItemOption,
  searchGrantableItemOptions,
  type GrantableItemCategory,
  type GrantableItemOption,
} from '../utils/grantableItemOptions';
import { formatDisplayLabel, ITEM_TYPE_LABELS } from '../utils/displayLabels';

interface GrantableItemSelectProps {
  value?: number;
  onChange?: (itemID: number | undefined) => void;
  onItemChange?: (item: GrantableItemOption | null) => void;
  preferredItemID?: number;
  preferredItemName?: string;
  preferredItemType?: string;
  showCategoryFilter?: boolean;
  defaultCategory?: GrantableItemCategory;
  placeholder?: string;
}

/** 可搜索的物品模板下拉，复用玩家背包发放物品的候选加载逻辑。 */
export function GrantableItemSelect({
  value,
  onChange,
  onItemChange,
  preferredItemID,
  preferredItemName = '',
  preferredItemType = '',
  showCategoryFilter = false,
  defaultCategory = 'all',
  placeholder = '输入物品名称或物品ID搜索',
}: GrantableItemSelectProps) {
  const [category, setCategory] = useState<GrantableItemCategory>(defaultCategory);
  const [loading, setLoading] = useState(false);
  const [options, setOptions] = useState<GrantableItemOption[]>([]);

  const preferredSeed = useMemo<GrantableItemOption | null>(() => {
    if (!preferredItemID || preferredItemID <= 0) {
      return null;
    }
    return {
      item_id: preferredItemID,
      item_name: preferredItemName || `物品${preferredItemID}`,
      item_type: preferredItemType || 'misc',
      quality: 1,
    };
  }, [preferredItemID, preferredItemName, preferredItemType]);

  useEffect(() => {
    if (!preferredSeed) {
      return;
    }
    setOptions((current) => {
      if (current.some((item) => item.item_id === preferredSeed.item_id)) {
        return current;
      }
      return [preferredSeed, ...current];
    });
  }, [preferredSeed]);

  async function loadOptions(keyword: string, nextCategory: GrantableItemCategory = category) {
    setLoading(true);
    try {
      const preferredItems = preferredSeed ? [preferredSeed] : [];
      const nextOptions = await searchGrantableItemOptions(keyword, nextCategory, preferredItems);
      setOptions(nextOptions);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品选项失败');
      setOptions(preferredSeed ? [preferredSeed] : []);
    } finally {
      setLoading(false);
    }
  }

  function handleChange(nextValue: number | undefined) {
    onChange?.(nextValue);
    onItemChange?.(findGrantableItemOption(options, nextValue) ?? null);
  }

  return (
    <>
      {showCategoryFilter ? (
        <Segmented<GrantableItemCategory>
          block
          style={{ marginBottom: 12 }}
          options={[
            { label: '全部', value: 'all' },
            { label: '装备', value: 'equipment' },
            { label: '其他', value: 'other' },
          ]}
          value={category}
          onChange={(nextCategory) => {
            setCategory(nextCategory);
            void loadOptions('', nextCategory);
          }}
        />
      ) : null}
      <Select
        showSearch
        allowClear
        filterOption={false}
        loading={loading}
        placeholder={placeholder}
        value={value && value > 0 ? value : undefined}
        onSearch={(keyword) => void loadOptions(keyword, category)}
        onFocus={() => {
          if (options.length === 0) {
            void loadOptions('', category);
          }
        }}
        onChange={handleChange}
        options={options.map((item) => ({
          label: `${item.item_name} (${item.item_id}) · ${formatDisplayLabel(ITEM_TYPE_LABELS, item.item_type)}`,
          value: item.item_id,
        }))}
      />
    </>
  );
}
