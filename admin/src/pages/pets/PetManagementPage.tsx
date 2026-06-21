import { Card, Tabs, Typography } from 'antd';
import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { PetProgressionPage } from '../progression/PetProgressionPage';
import { PetCombatStatCapPage } from './PetCombatStatCapPage';
import { PetDefinitionPage } from './PetDefinitionPage';
import { PetSkillSlotUnlockPage } from './PetSkillSlotUnlockPage';

const PET_MANAGEMENT_TABS = [
  {
    key: 'definitions',
    label: '系统宠物',
    children: <PetDefinitionPage />,
  },
  {
    key: 'progression',
    label: '成长配置',
    children: <PetProgressionPage />,
  },
  {
    key: 'skill-slots',
    label: '神符槽解锁',
    children: <PetSkillSlotUnlockPage />,
  },
  {
    key: 'combat-caps',
    label: '属性封顶',
    children: <PetCombatStatCapPage />,
  },
];

const DEFAULT_TAB_KEY = PET_MANAGEMENT_TABS[0].key;

// 宠物管理聚合页：只保留系统级宠物配置，玩家宠物实例统一在玩家详情中维护。
export function PetManagementPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tabKey = searchParams.get('tab') ?? DEFAULT_TAB_KEY;
  const validTabKeys = useMemo(() => new Set(PET_MANAGEMENT_TABS.map((item) => item.key)), []);
  const activeKey = validTabKeys.has(tabKey) ? tabKey : DEFAULT_TAB_KEY;

  function handleTabChange(nextKey: string) {
    setSearchParams({ tab: nextKey }, { replace: true });
  }

  return (
    <Card>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        宠物管理
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        统一管理系统宠物、成长曲线、神符槽解锁和战斗属性封顶配置；玩家宠物实例统一在玩家详情中维护。
      </Typography.Paragraph>
      <Tabs
        activeKey={activeKey}
        destroyOnHidden
        items={PET_MANAGEMENT_TABS}
        onChange={handleTabChange}
      />
    </Card>
  );
}
