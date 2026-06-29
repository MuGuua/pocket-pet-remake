import { Button, Input, Modal, Segmented, Space, Spin, Tooltip, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { fetchAdminPetDefinitions } from '../services/petDefinition';
import type { AdminPetDefinitionSummary } from '../types/petDefinition';
import type { GrantableItemCategory } from '../utils/grantableItemOptions';
import {
  searchGrantableItemMentionOptions,
  type GrantableItemMentionOption,
} from '../utils/grantableItemMentionOptions';
import { buildItemMentionToken, buildPetMentionToken } from '../utils/itemMentionContent';
import { AdminItemIcon } from './AdminItemIcon';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../utils/modalLayout';
import { formatDisplayLabel, ITEM_TYPE_LABELS } from '../utils/displayLabels';

type MentionPickerTab = GrantableItemCategory | 'pet';

interface ItemMentionPickerModalProps {
  open: boolean;
  onCancel: () => void;
  onSelect: (token: string, preview?: GrantableItemMentionOption | AdminPetDefinitionSummary) => void;
}

/**
 * 系统物品/装备/宠物选择弹窗：与剧情编辑器一致，选中后插入 {item:ID} 或 {pet:ID}。
 */
export function ItemMentionPickerModal({ open, onCancel, onSelect }: ItemMentionPickerModalProps) {
  const [activeTab, setActiveTab] = useState<MentionPickerTab>('all');
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(false);
  const [itemRows, setItemRows] = useState<GrantableItemMentionOption[]>([]);
  const [petRows, setPetRows] = useState<AdminPetDefinitionSummary[]>([]);

  useEffect(() => {
    if (!open) {
      return;
    }
    void loadRows(activeTab, keyword);
  }, [open, activeTab, keyword]);

  async function loadRows(tab: MentionPickerTab, searchKeyword: string) {
    setLoading(true);
    try {
      if (tab === 'pet') {
        const result = await fetchAdminPetDefinitions({
          filters: { name: searchKeyword.trim() || undefined, enabled: 'true' },
          page: 1,
          pageSize: 48,
        });
        setPetRows(result.items);
        setItemRows([]);
        return;
      }
      const rows = await searchGrantableItemMentionOptions(searchKeyword, tab);
      setItemRows(rows);
      setPetRows([]);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统模板失败');
      setItemRows([]);
      setPetRows([]);
    } finally {
      setLoading(false);
    }
  }

  const tabOptions = useMemo(
    () => [
      { label: '全部', value: 'all' as MentionPickerTab },
      { label: '装备/武器', value: 'equipment' as MentionPickerTab },
      { label: '其他物品', value: 'other' as MentionPickerTab },
      { label: '宠物', value: 'pet' as MentionPickerTab },
    ],
    [],
  );

  return (
    <Modal
      title="插入系统模板"
      open={open}
      onCancel={onCancel}
      footer={null}
      width={860}
      style={{ top: FIXED_FORM_MODAL_TOP }}
      styles={FIXED_FORM_MODAL_STYLES}
      destroyOnClose
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Typography.Text type="secondary">
          选中后会插入占位符：物品/装备为 {'{item:ID}'}，宠物为 {'{pet:ID}'}；客户端会展示权威名称与 icon（宠物仅名称）。
        </Typography.Text>
        <Segmented
          block
          options={tabOptions}
          value={activeTab}
          onChange={(value) => setActiveTab(value as MentionPickerTab)}
        />
        <Input.Search
          allowClear
          placeholder={activeTab === 'pet' ? '搜索宠物 ID 或名称' : '搜索物品 ID、编码或名称'}
          enterButton="搜索"
          onSearch={(value) => setKeyword(value)}
        />
        <Spin spinning={loading}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(96px, 1fr))',
              gap: 12,
              maxHeight: 420,
              overflow: 'auto',
              paddingRight: 4,
            }}
          >
            {activeTab === 'pet'
              ? petRows.map((pet) => (
                  <Tooltip
                    key={pet.pet_id}
                    title={(
                      <Space direction="vertical" size={2}>
                        <span>{pet.pet_name}</span>
                        <span>ID：{pet.pet_id}</span>
                        <span>外观：{pet.skin_id || '-'}</span>
                      </Space>
                    )}
                  >
                    <Button
                      onClick={() => {
                        onSelect(buildPetMentionToken(pet.pet_id), pet);
                        onCancel();
                      }}
                      style={{ height: 112, padding: 8, whiteSpace: 'normal' }}
                    >
                      <Space direction="vertical" size={6} align="center" style={{ width: '100%' }}>
                        <AdminItemIcon icon="" size={32} fallbackText="宠" />
                        <Typography.Text ellipsis style={{ width: '100%', textAlign: 'center', fontSize: 12 }}>
                          {pet.pet_name}
                        </Typography.Text>
                      </Space>
                    </Button>
                  </Tooltip>
                ))
              : itemRows.map((item) => (
                  <Tooltip
                    key={item.item_id}
                    title={(
                      <Space direction="vertical" size={2}>
                        <span>{item.item_name}</span>
                        <span>ID：{item.item_id}</span>
                        <span>{formatDisplayLabel(ITEM_TYPE_LABELS, item.item_type)}</span>
                        <span>{item.desc || '暂无介绍'}</span>
                      </Space>
                    )}
                  >
                    <Button
                      onClick={() => {
                        onSelect(buildItemMentionToken(item.item_id), item);
                        onCancel();
                      }}
                      style={{ height: 112, padding: 8, whiteSpace: 'normal' }}
                    >
                      <Space direction="vertical" size={6} align="center" style={{ width: '100%' }}>
                        <AdminItemIcon icon={item.icon} size={32} />
                        <Typography.Text ellipsis style={{ width: '100%', textAlign: 'center', fontSize: 12 }}>
                          {item.item_name}
                        </Typography.Text>
                      </Space>
                    </Button>
                  </Tooltip>
                ))}
          </div>
        </Spin>
      </Space>
    </Modal>
  );
}
