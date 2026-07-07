import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Empty, Form, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { fetchAllAdminMonsterDefinitions } from '../services/monsterDefinition';

interface MonsterSpawnListEditorProps {
  value?: number[];
  onChange?: (nextValue: number[]) => void;
  disabled?: boolean;
  description?: string;
}

interface MonsterPickerFormValues {
  monster_id: number;
}

// 遭遇刷怪列表编辑器：用列表顺序明确表示每个刷怪槽位对应的 monster_id。
export function MonsterSpawnListEditor({
  value = [],
  onChange,
  disabled = false,
  description,
}: MonsterSpawnListEditorProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [monsterOptions, setMonsterOptions] = useState<Array<{ value: number; label: string }>>([]);
  const [pickerForm] = Form.useForm<MonsterPickerFormValues>();

  useEffect(() => {
    void loadMonsterOptions();
  }, []);

  async function loadMonsterOptions() {
    setOptionsLoading(true);
    try {
      const items = await fetchAllAdminMonsterDefinitions({ enabled: 'true' });
      setMonsterOptions(
        items.map((item) => ({
          value: item.monster_id,
          label: `${item.monster_name} (#${item.monster_id})`,
        })),
      );
    } catch {
      setMonsterOptions([]);
    } finally {
      setOptionsLoading(false);
    }
  }

  function resolveMonsterLabel(monsterID: number): string {
    const option = monsterOptions.find((item) => item.value === monsterID);
    return option?.label ?? `怪物 #${monsterID}`;
  }

  function emitChange(nextValue: number[]) {
    onChange?.(nextValue);
  }

  function handleRemove(index: number) {
    emitChange(value.filter((_monsterID, rowIndex) => rowIndex !== index));
  }

  function handleMove(index: number, delta: number) {
    const targetIndex = index + delta;
    if (targetIndex < 0 || targetIndex >= value.length) {
      return;
    }
    const nextValue = [...value];
    [nextValue[index], nextValue[targetIndex]] = [nextValue[targetIndex], nextValue[index]];
    emitChange(nextValue);
  }

  function openPicker() {
    pickerForm.resetFields();
    setPickerOpen(true);
  }

  function handlePickerSubmit(formValues: MonsterPickerFormValues) {
    const monsterID = Number(formValues.monster_id);
    if (!monsterID) {
      setPickerOpen(false);
      pickerForm.resetFields();
      return;
    }
    emitChange([...value, monsterID]);
    setPickerOpen(false);
    pickerForm.resetFields();
  }

  const columns = useMemo<ColumnsType<{ monster_id: number; index: number }>>(
    () => [
      {
        title: '槽位',
        key: 'order',
        width: 72,
        render: (_value, _record, index) => <Tag>{index + 1}</Tag>,
      },
      {
        title: '怪物',
        dataIndex: 'monster_id',
        key: 'monster_id',
        render: (monsterID: number) => resolveMonsterLabel(monsterID),
      },
      {
        title: '操作',
        key: 'actions',
        width: 220,
        render: (_value, row) => (
          <Space size={4}>
            <Button
              size="small"
              icon={<ArrowUpOutlined />}
              disabled={disabled || row.index === 0}
              onClick={() => handleMove(row.index, -1)}
            />
            <Button
              size="small"
              icon={<ArrowDownOutlined />}
              disabled={disabled || row.index === value.length - 1}
              onClick={() => handleMove(row.index, 1)}
            />
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={disabled}
              onClick={() => handleRemove(row.index)}
            >
              删除
            </Button>
          </Space>
        ),
      },
    ],
    [disabled, monsterOptions, value],
  );

  const tableRows = value.map((monsterID, index) => ({ monster_id: monsterID, index }));

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }} align="start">
        <Typography.Text type="secondary">
          {description ?? '按列表顺序配置刷怪槽位；同一怪物可重复添加，表示多个相同出场位。'}
        </Typography.Text>
        <Button type="dashed" icon={<PlusOutlined />} disabled={disabled} onClick={openPicker}>
          {`添加怪物 (${value.length})`}
        </Button>
      </Space>
      <Table
        size="small"
        rowKey={(record) => `${record.monster_id}-${record.index}`}
        columns={columns}
        dataSource={tableRows}
        pagination={false}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="尚未配置刷怪列表，点击右上角添加。"
            />
          ),
        }}
      />
      <Modal
        title="添加刷怪"
        open={pickerOpen}
        onCancel={() => {
          setPickerOpen(false);
          pickerForm.resetFields();
        }}
        onOk={() => pickerForm.submit()}
        destroyOnClose
        width={520}
      >
        <Form form={pickerForm} layout="vertical" onFinish={handlePickerSubmit}>
          <Form.Item
            label="选择怪物模板"
            name="monster_id"
            rules={[{ required: true, message: '请选择怪物模板' }]}
          >
            <Select
              showSearch
              placeholder="搜索怪物名称或 ID"
              loading={optionsLoading}
              options={monsterOptions}
              optionFilterProp="label"
              notFoundContent={optionsLoading ? '加载中...' : '没有可添加的怪物'}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
