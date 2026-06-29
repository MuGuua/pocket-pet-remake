import { Button, Col, Empty, Form, Input, InputNumber, Modal, Row, Select, Space, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined } from '@ant-design/icons';
import { useMemo, useState } from 'react';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { RichTextEditor } from '../../components/RichTextEditor';
import { buildSelectOptions, formatDisplayLabel, QUEST_EVENT_TYPE_LABELS } from '../../utils/displayLabels';
import type { QuestStageFormItem } from './questStageUtils';
import { createDefaultStage } from './questStageUtils';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

const eventTypeOptions = buildSelectOptions(QUEST_EVENT_TYPE_LABELS);

interface QuestStageEditorProps {
  /** 当前阶段列表，由 Form.Item 注入。 */
  value?: QuestStageFormItem[];
  /** 阶段列表变更回调，由 Form.Item 注入。 */
  onChange?: (stages: QuestStageFormItem[]) => void;
  /** 当前任务 ID，用于弹窗内展示 NPC 菜单可见条件；新建时可传 0。 */
  questID: number;
}

/** 任务模板多阶段编辑器：主表单仅展示阶段列表，新增/编辑在弹窗中完成。 */
export function QuestStageEditor({ value, onChange, questID }: QuestStageEditorProps) {
  const stages: QuestStageFormItem[] = value ?? [];
  const [stageModalOpen, setStageModalOpen] = useState<boolean>(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [stageForm] = Form.useForm<QuestStageFormItem>();

  const columns = useMemo<ColumnsType<QuestStageFormItem>>(
    () => [
      { title: '阶段 ID', dataIndex: 'objective_id', key: 'objective_id', width: 90 },
      {
        title: '事件类型',
        dataIndex: 'event_type',
        key: 'event_type',
        width: 120,
        render: (eventType: string) => formatDisplayLabel(QUEST_EVENT_TYPE_LABELS, eventType),
      },
      { title: '阶段描述', dataIndex: 'description', key: 'description', ellipsis: true },
      { title: '目标 NPC', dataIndex: 'npc_id', key: 'npc_id', width: 100, render: (npcID: number) => (npcID > 0 ? npcID : '-') },
      { title: '菜单 entry', dataIndex: 'menu_entry_id', key: 'menu_entry_id', width: 110, render: (entryID: number) => (entryID > 0 ? entryID : '-') },
      {
        title: '操作',
        key: 'actions',
        width: 80,
        fixed: 'right',
        render: (_value, _record, index) => (
          <TableActionDropdown
            actions={[
              { key: 'edit', label: '编辑', onClick: () => openStageEditor(index) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                disabled: stages.length <= 1,
                confirm: {
                  title: '确认删除该阶段吗？',
                  description: '删除后需重新保存任务模板才会生效。',
                  okText: '确认删除',
                  cancelText: '取消',
                },
                onClick: () => removeStage(index),
              },
            ]}
          />
        ),
      },
    ],
    [stages.length],
  );

  /** 打开新增阶段弹窗。 */
  function openStageCreator(): void {
    const nextObjectiveID: number = stages.reduce((maxID, stage) => Math.max(maxID, stage.objective_id), 0) + 1;
    setEditingIndex(null);
    stageForm.resetFields();
    stageForm.setFieldsValue(createDefaultStage(nextObjectiveID));
    setStageModalOpen(true);
  }

  /** 打开编辑阶段弹窗。 */
  function openStageEditor(index: number): void {
    setEditingIndex(index);
    stageForm.resetFields();
    stageForm.setFieldsValue(stages[index]);
    setStageModalOpen(true);
  }

  /** 删除指定下标的阶段。 */
  function removeStage(index: number): void {
    if (stages.length <= 1) {
      return;
    }
    onChange?.(stages.filter((_stage, stageIndex) => stageIndex !== index));
  }

  /** 弹窗确认后写回阶段列表。 */
  function handleStageModalSubmit(values: QuestStageFormItem): void {
    const nextStages: QuestStageFormItem[] = [...stages];
    if (editingIndex === null) {
      nextStages.push(values);
    } else {
      nextStages[editingIndex] = values;
    }
    nextStages.sort((left, right) => left.objective_id - right.objective_id);
    onChange?.(nextStages);
    setStageModalOpen(false);
    setEditingIndex(null);
    stageForm.resetFields();
  }

  return (
    <>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Table
          columns={columns}
          dataSource={stages}
          rowKey={(record) => `${record.objective_id}-${record.description}`}
          size="small"
          pagination={false}
          scroll={{ x: 720 }}
          locale={{ emptyText: <Empty description="暂无任务阶段，请点击下方按钮添加" /> }}
        />
        <Button type="dashed" block icon={<PlusOutlined />} onClick={openStageCreator}>
          添加阶段
        </Button>
      </Space>
      <Modal
        title={editingIndex === null ? '新增任务阶段' : `编辑任务阶段 · ${stageForm.getFieldValue('objective_id') ?? ''}`}
        open={stageModalOpen}
        onCancel={() => {
          setStageModalOpen(false);
          setEditingIndex(null);
          stageForm.resetFields();
        }}
        onOk={() => stageForm.submit()}
        destroyOnClose
        width={760}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText="保存阶段"
        cancelText="取消"
      >
        <Form form={stageForm} layout="vertical" onFinish={(values) => handleStageModalSubmit(values)}>
          <Row gutter={12}>
            <Col xs={12} md={6}>
              <Form.Item
                label="阶段 ID"
                name="objective_id"
                rules={[{ required: true, message: '请输入阶段 ID' }]}
                tooltip="与 NPC 菜单/剧情条件中的 objective_id 一致，按 1、2、3 顺序推进"
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12} md={10}>
              <Form.Item label="事件类型" name="event_type" rules={[{ required: true, message: '请选择事件类型' }]}>
                <Select options={eventTypeOptions} />
              </Form.Item>
            </Col>
            <Col xs={12} md={8}>
              <Form.Item label="目标次数" name="target_value" rules={[{ required: true, message: '请输入目标次数' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item label="阶段描述" name="description" rules={[{ required: true, message: '请输入阶段描述' }]} extra="支持 BBCode 富文本。">
                <RichTextEditor rows={2} placeholder="例如：与市场理萌交谈" showPreview={false} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="目标 NPC 实体 ID" name="npc_id" tooltip="TALK_TO_NPC 阶段必填">
                <InputNumber min={0} style={{ width: '100%' }} placeholder="93001" />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="目标场景 ID" name="scene_id" tooltip="ENTER_SCENE 阶段填写">
                <InputNumber min={0} style={{ width: '100%' }} placeholder="3" />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="战斗类型" name="battle_type" tooltip="WIN_BATTLE 阶段填写，例如 PVE">
                <Input placeholder="PVE" />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="引导场景 ID" name="guide_scene_id">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item label="引导文案" name="guide_text" extra="任务追踪/引导提示，支持 BBCode 富文本。">
                <RichTextEditor rows={2} placeholder="例如：去商业区找市场理萌" showPreview={false} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="菜单 entry_id" name="menu_entry_id" tooltip="对应 npc_menu_entry.entry_id">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="剧情 entry_id" name="dialogue_entry_id" tooltip="通常与菜单 entry_id 相同">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            {questID > 0 ? (
              <Col span={24}>
                <StageConditionHint questID={questID} stageForm={stageForm} />
              </Col>
            ) : null}
          </Row>
        </Form>
      </Modal>
    </>
  );
}

interface StageConditionHintProps {
  questID: number;
  stageForm: ReturnType<typeof Form.useForm<QuestStageFormItem>>[0];
}

/** 根据弹窗内阶段 ID 生成 NPC 菜单/剧情可见条件提示。 */
function StageConditionHint({ questID, stageForm }: StageConditionHintProps) {
  const objectiveID: number = Form.useWatch('objective_id', stageForm) ?? 1;
  return (
    <Typography.Text type="secondary">
      NPC 菜单/剧情「可见条件」建议：
      <pre style={{ margin: '8px 0 0', fontSize: 12, background: '#fafafa', padding: 8, borderRadius: 4 }}>
        {JSON.stringify(
          {
            quest_id: questID,
            quest_state: 'ACCEPTED',
            objective_id: objectiveID,
            objective_completed: false,
          },
          null,
          2,
        )}
      </pre>
      请在「地图 NPC → 菜单配置」中配置上述条件，并在剧情节点 effects 填写对应 quest_event。
    </Typography.Text>
  );
}
