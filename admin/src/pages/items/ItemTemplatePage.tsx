import { Card, Tabs, Typography } from 'antd';
import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { EquipmentDefinitionPage } from '../equipment/EquipmentDefinitionPage';
import { ItemDefinitionPage } from './ItemDefinitionPage';

const ITEM_TEMPLATE_TABS = [
  {
    key: 'items',
    label: '普通物品',
    children: <ItemDefinitionPage excludeItemType="equipment" />,
  },
  {
    key: 'equipment',
    label: '装备',
    children: <EquipmentDefinitionPage embedded />,
  },
];

const DEFAULT_TAB_KEY = ITEM_TEMPLATE_TABS[0].key;

// 物品模板聚合页把普通物品与人物装备收口到同一菜单，避免运营在两个入口之间来回切换。
export function ItemTemplatePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tabKey = searchParams.get('tab') ?? DEFAULT_TAB_KEY;
  const validTabKeys = useMemo(() => new Set(ITEM_TEMPLATE_TABS.map((item) => item.key)), []);
  const activeKey = validTabKeys.has(tabKey) ? tabKey : DEFAULT_TAB_KEY;

  function handleTabChange(nextKey: string) {
    setSearchParams({ tab: nextKey }, { replace: true });
  }

  return (
    <Card>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        物品模板
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        普通物品与人物装备都从数据库模板下发到客户端；图标字段直接填写客户端资源路径，玩家背包发放会复用这里的正式模板。
      </Typography.Paragraph>
      <Tabs activeKey={activeKey} destroyOnHidden items={ITEM_TEMPLATE_TABS} onChange={handleTabChange} />
    </Card>
  );
}
