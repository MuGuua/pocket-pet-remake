import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Empty, Form, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { fetchAllAdminSkillDefinitions } from '../services/skillDefinition';
import type { SkillReferenceMap } from '../utils/skillReference';

interface PetSkillSlotListEditorProps {
  /** 当前已选 skill_id 列表，顺序即战斗/发放顺序。 */
  value?: number[];
  /** 列表变更回调。 */
  onChange?: (nextValue: number[]) => void;
  /** 最多可配置技能数量。 */
  maxCount: number;
  /** 区块说明文案。 */
  description?: string;
  /** 技能分类过滤，默认 pet。 */
  category?: string;
  /** 技能名称映射，用于展示已有 skill_id。 */
  skillReferenceMap: SkillReferenceMap;
  /** 禁用编辑。 */
  disabled?: boolean;
}

interface SkillPickerFormValues {
  skill_id: number;
}

/** 宠物模板技能槽可视化编辑器：表格展示 + 弹窗选择新增。 */
export function PetSkillSlotListEditor({
  value = [],
  onChange,
  maxCount,
  description,
  category = 'pet',
  skillReferenceMap,
  disabled = false,
}: PetSkillSlotListEditorProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const [skillOptions, setSkillOptions] = useState<Array<{ value: number; label: string }>>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [pickerForm] = Form.useForm<SkillPickerFormValues>();

  useEffect(() => {
    void loadSkillOptions();
  }, [category]);

  async function loadSkillOptions() {
    setOptionsLoading(true);
    try {
      const items = await fetchAllAdminSkillDefinitions({
        category,
        enabled: 'true',
      });
      setSkillOptions(
        items.map((item) => ({
          value: item.skill_id,
          label: `${item.skill_name} (#${item.skill_id})`,
        })),
      );
    } catch {
      setSkillOptions([]);
    } finally {
      setOptionsLoading(false);
    }
  }

  function resolveSkillLabel(skillID: number): string {
    const mappedName = skillReferenceMap.idToName[skillID];
    if (mappedName) {
      return `${mappedName} (#${skillID})`;
    }
    const option = skillOptions.find((item) => item.value === skillID);
    return option?.label ?? `技能 #${skillID}`;
  }

  function emitChange(nextValue: number[]) {
    onChange?.(nextValue);
  }

  function handleRemove(index: number) {
    emitChange(value.filter((_skillID, entryIndex) => entryIndex !== index));
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
    pickerForm.setFieldsValue({ skill_id: undefined as unknown as number });
    setPickerOpen(true);
  }

  function handlePickerSubmit(formValues: SkillPickerFormValues) {
    const skillID = Number(formValues.skill_id);
    if (!skillID || value.includes(skillID)) {
      setPickerOpen(false);
      pickerForm.resetFields();
      return;
    }
    emitChange([...value, skillID]);
    setPickerOpen(false);
    pickerForm.resetFields();
  }

  const availableOptions = useMemo(
    () => skillOptions.filter((option) => !value.includes(option.value)),
    [skillOptions, value],
  );

  const columns = useMemo<ColumnsType<{ skill_id: number; index: number }>>(
    () => [
      {
        title: '序号',
        key: 'order',
        width: 72,
        render: (_record, _row, index) => <Tag>{index + 1}</Tag>,
      },
      {
        title: '技能',
        dataIndex: 'skill_id',
        key: 'skill_id',
        render: (skillID: number) => resolveSkillLabel(skillID),
      },
      {
        title: '操作',
        key: 'actions',
        width: 220,
        render: (_record, row) => (
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
    [disabled, skillOptions, skillReferenceMap, value],
  );

  const tableRows = value.map((skillID, index) => ({ skill_id: skillID, index }));

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }} align="start">
        <Typography.Text type="secondary">
          {description ?? `最多配置 ${maxCount} 个技能，顺序会影响兼容技能列表的默认合并顺序。`}
        </Typography.Text>
        <Button
          type="dashed"
          icon={<PlusOutlined />}
          disabled={disabled || value.length >= maxCount}
          onClick={openPicker}
        >
          添加技能 ({value.length}/{maxCount})
        </Button>
      </Space>
      <Table
        size="small"
        rowKey={(record) => `${record.skill_id}-${record.index}`}
        columns={columns}
        dataSource={tableRows}
        pagination={false}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="尚未配置技能，点击右上角添加。"
            />
          ),
        }}
      />
      <Modal
        title="添加技能"
        open={pickerOpen}
        onCancel={() => {
          setPickerOpen(false);
          pickerForm.resetFields();
        }}
        onOk={() => pickerForm.submit()}
        destroyOnClose
        width={520}
      >
        <Form form={pickerForm} layout="vertical" onFinish={(formValues) => handlePickerSubmit(formValues)}>
          <Form.Item
            label="选择技能"
            name="skill_id"
            rules={[{ required: true, message: '请选择技能' }]}
          >
            <Select
              showSearch
              placeholder="搜索技能名称或 ID"
              loading={optionsLoading}
              options={availableOptions}
              optionFilterProp="label"
              notFoundContent={optionsLoading ? '加载中…' : '没有可添加的技能'}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
