import { Card, Tabs, Typography } from 'antd';
import { useState } from 'react';
import { NpcFixedEncounterPanel } from './MonsterEncounterPage';
import { SceneWildEncounterPanel } from './SceneWildEncounterPage';

type EncounterTabKey = 'wild' | 'npc';

// 怪物遭遇统一入口：地图暗雷与 NPC 固定战分 Tab 管理，底层仍走两套独立 API。
export function EncounterConfigPage() {
  const [activeTab, setActiveTab] = useState<EncounterTabKey>('wild');

  return (
    <Card title="怪物遭遇配置">
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        地图暗雷按 scene_id 配置移动概率与刷怪池；NPC 固定战按 entity_id 配置交互开战刷怪。保存后玩家需重新进图或切图才会拿到最新暗雷配置。
      </Typography.Paragraph>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as EncounterTabKey)}
        items={[
          {
            key: 'wild',
            label: '地图暗雷',
            children: <SceneWildEncounterPanel />,
          },
          {
            key: 'npc',
            label: 'NPC 固定战',
            children: <NpcFixedEncounterPanel />,
          },
        ]}
      />
    </Card>
  );
}
