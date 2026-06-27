import { Col, Form, InputNumber, Row, Select, Switch, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo } from 'react';
import type { AdminEquipmentEnhanceGoldCost, EnhanceGoldIncrementMode } from '../types/equipmentDefinition';
import { ENHANCE_GOLD_INCREMENT_MODE_OPTIONS } from '../types/equipmentDefinition';
import { buildEnhanceGoldCostPreview, describeEnhanceGoldCostFormula } from '../utils/enhanceGoldCost';

interface EnhanceGoldCostPreviewRow {
  target_level: number;
  cost_copper: number;
}

/** 装备编辑弹窗内的强化铜币公式配置区。 */
export function EquipmentEnhanceGoldCostEditor() {
  const form = Form.useFormInstance();
  const canEnhance: boolean = Form.useWatch('can_enhance', form) ?? false;
  const incrementMode: EnhanceGoldIncrementMode =
    Form.useWatch(['enhance_gold_cost', 'increment_mode'], form) ?? 'fixed';
  const watchedConfig: AdminEquipmentEnhanceGoldCost | undefined = Form.useWatch('enhance_gold_cost', form);

  const previewRows = useMemo<EnhanceGoldCostPreviewRow[]>(() => {
    if (!watchedConfig) {
      return [];
    }
    return buildEnhanceGoldCostPreview({
      is_enabled: Boolean(watchedConfig.is_enabled),
      base_copper: Number(watchedConfig.base_copper ?? 0),
      increment_mode: watchedConfig.increment_mode === 'percent' ? 'percent' : 'fixed',
      increment_fixed: Number(watchedConfig.increment_fixed ?? 0),
      increment_percent: Number(watchedConfig.increment_percent ?? 0),
    });
  }, [watchedConfig]);

  const previewColumns = useMemo<ColumnsType<EnhanceGoldCostPreviewRow>>(
    () => [
      {
        title: '目标等级',
        dataIndex: 'target_level',
        width: 100,
        render: (value: number) => `+${value}`,
      },
      {
        title: '铜币消耗',
        dataIndex: 'cost_copper',
        render: (value: number) => value.toLocaleString('zh-CN'),
      },
    ],
    [],
  );

  if (!canEnhance) {
    return null;
  }

  return (
    <>
      <Typography.Text type="secondary">
        强化至 +N 时按下方公式计算单次铜币消耗；材料消耗仍按全局强化材料表配置。
      </Typography.Text>
      <Row gutter={16} style={{ marginTop: 12 }}>
        <Col xs={24} md={6}>
          <Form.Item label="启用铜币消耗" name={['enhance_gold_cost', 'is_enabled']} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Col>
        <Col xs={24} md={6}>
          <Form.Item
            label="基础消耗（铜）"
            name={['enhance_gold_cost', 'base_copper']}
            rules={[{ required: true, message: '请输入基础消耗' }]}
            extra="强化至 +1 的铜币消耗"
          >
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={6}>
          <Form.Item
            label="递增方式"
            name={['enhance_gold_cost', 'increment_mode']}
            rules={[{ required: true, message: '请选择递增方式' }]}
          >
            <Select options={ENHANCE_GOLD_INCREMENT_MODE_OPTIONS} />
          </Form.Item>
        </Col>
        {incrementMode === 'fixed' ? (
          <Col xs={24} md={6}>
            <Form.Item
              label="每级固定增加（铜）"
              name={['enhance_gold_cost', 'increment_fixed']}
              rules={[{ required: true, message: '请输入固定增加值' }]}
              extra="例如 200：+1=100，+2=300，+3=500"
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        ) : (
          <Col xs={24} md={6}>
            <Form.Item
              label="每级递增百分比"
              name={['enhance_gold_cost', 'increment_percent']}
              rules={[{ required: true, message: '请输入递增百分比' }]}
              extra="例如 10：+1=100，+2=110，+3=121（复合）"
            >
              <InputNumber min={0} max={1000} precision={0} addonAfter="%" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        )}
      </Row>
      <Typography.Text type="secondary">{describeEnhanceGoldCostFormula(incrementMode)}</Typography.Text>
      <Table<EnhanceGoldCostPreviewRow>
        style={{ marginTop: 12 }}
        size="small"
        rowKey="target_level"
        columns={previewColumns}
        dataSource={previewRows}
        pagination={false}
        scroll={{ y: 180 }}
      />
    </>
  );
}
