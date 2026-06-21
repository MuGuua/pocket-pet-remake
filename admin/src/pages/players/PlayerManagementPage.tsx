import { Card, Tabs, Typography } from 'antd';
import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { PlayerProgressionPage } from '../progression/PlayerProgressionPage';
import { PlayerListPage } from './PlayerListPage';

const PLAYER_MANAGEMENT_TABS = [
  {
    key: 'list',
    label: '玩家列表',
    children: <PlayerListPage />,
  },
  {
    key: 'progression',
    label: '成长配置',
    children: <PlayerProgressionPage />,
  },
];

const DEFAULT_TAB_KEY = PLAYER_MANAGEMENT_TABS[0].key;

// 玩家管理聚合页：把玩家账号实例与玩家成长配置合并到同一入口，减少运营后台左侧菜单数量。
export function PlayerManagementPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tabKey = searchParams.get('tab') ?? DEFAULT_TAB_KEY;
  const validTabKeys = useMemo(() => new Set(PLAYER_MANAGEMENT_TABS.map((item) => item.key)), []);
  const activeKey = validTabKeys.has(tabKey) ? tabKey : DEFAULT_TAB_KEY;

  function handleTabChange(nextKey: string) {
    setSearchParams({ tab: nextKey }, { replace: true });
  }

  return (
    <Card>
      <Typography.Title level={3} style={{ marginTop: 0 }}>
        玩家管理
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        统一管理玩家账号、角色数据，以及服务端升级和加点计算使用的玩家成长配置。
      </Typography.Paragraph>
      <Tabs
        activeKey={activeKey}
        destroyOnHidden
        items={PLAYER_MANAGEMENT_TABS}
        onChange={handleTabChange}
      />
    </Card>
  );
}
