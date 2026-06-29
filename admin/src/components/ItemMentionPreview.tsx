import { Card, Space, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { fetchAdminItemDetail } from '../services/item';
import { fetchAdminPetDefinitionDetail } from '../services/petDefinition';
import {
  extractMentionItemIDs,
  extractMentionPetIDs,
  renderMentionContentFragments,
  type ItemMentionPreviewRecord,
  type PetMentionPreviewRecord,
} from '../utils/itemMentionContent';

interface ItemMentionPreviewProps {
  content: string;
  title?: string;
  /** 为 true 时不渲染 Card 外框，仅输出预览正文。 */
  embedded?: boolean;
  showPlayerName?: boolean;
  itemMap?: Record<number, ItemMentionPreviewRecord>;
  petMap?: Record<number, PetMentionPreviewRecord>;
}

/**
 * 富文本 + 物品/宠物占位符的后台预览区。
 * 未传入 itemMap/petMap 时会按文案中的 ID 懒加载名称。
 */
export function ItemMentionPreview({
  content,
  title = '客户端预览',
  embedded = false,
  showPlayerName = false,
  itemMap: externalItemMap,
  petMap: externalPetMap,
}: ItemMentionPreviewProps) {
  const [loadedItemMap, setLoadedItemMap] = useState<Record<number, ItemMentionPreviewRecord>>({});
  const [loadedPetMap, setLoadedPetMap] = useState<Record<number, PetMentionPreviewRecord>>({});

  const itemIDs = useMemo(() => extractMentionItemIDs(content), [content]);
  const petIDs = useMemo(() => extractMentionPetIDs(content), [content]);
  const itemIDsKey = itemIDs.join(',');
  const petIDsKey = petIDs.join(',');

  useEffect(() => {
    if (externalItemMap || itemIDs.length === 0) {
      return;
    }
    let cancelled = false;
    void (async () => {
      const nextMap: Record<number, ItemMentionPreviewRecord> = {};
      for (const itemID of itemIDs) {
        try {
          const detail = await fetchAdminItemDetail(itemID);
          nextMap[itemID] = {
            item_id: detail.item_id,
            item_name: detail.item_name,
            icon: detail.icon,
          };
        } catch {
          nextMap[itemID] = { item_id: itemID, item_name: `物品${itemID}`, icon: '' };
        }
      }
      if (!cancelled) {
        setLoadedItemMap(nextMap);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [content, externalItemMap, itemIDsKey, itemIDs]);

  useEffect(() => {
    if (externalPetMap || petIDs.length === 0) {
      return;
    }
    let cancelled = false;
    void (async () => {
      const nextMap: Record<number, PetMentionPreviewRecord> = {};
      for (const petID of petIDs) {
        try {
          const detail = await fetchAdminPetDefinitionDetail(petID);
          nextMap[petID] = {
            pet_id: detail.pet_id,
            pet_name: detail.pet_name,
          };
        } catch {
          nextMap[petID] = { pet_id: petID, pet_name: `宠物${petID}` };
        }
      }
      if (!cancelled) {
        setLoadedPetMap(nextMap);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [content, externalPetMap, petIDsKey, petIDs]);

  const itemMap = externalItemMap ?? loadedItemMap;
  const petMap = externalPetMap ?? loadedPetMap;
  const fragments = renderMentionContentFragments(content, itemMap, petMap, { showPlayerName });
  const normalized = content.trim();

  const body = normalized ? (
    <Space wrap size={6} style={{ color: '#c7bbb0', lineHeight: 1.6, wordBreak: 'break-word' }}>
      {fragments.length > 0 ? fragments : null}
    </Space>
  ) : (
    <Typography.Text type="secondary">输入内容后将在此预览客户端展示效果</Typography.Text>
  );

  if (embedded) {
    return body;
  }

  return (
    <Card size="small" title={title} styles={{ body: { padding: 12 } }}>
      {body}
    </Card>
  );
}
