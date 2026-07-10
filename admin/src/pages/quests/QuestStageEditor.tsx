import { Button, Card, Col, Empty, Form, Input, InputNumber, Modal, Row, Select, Space, Tag, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { CSSProperties, DragEvent } from 'react';
import { useState } from 'react';
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
  const [draggingIndex, setDraggingIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [stageForm] = Form.useForm<QuestStageFormItem>();

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

  /** 按拖拽结果调整阶段顺序；列表顺序即任务推进顺序。 */
  function moveStage(fromIndex: number, toIndex: number): void {
    if (fromIndex === toIndex || fromIndex < 0 || toIndex < 0 || fromIndex >= stages.length || toIndex >= stages.length) {
      return;
    }
    const nextStages: QuestStageFormItem[] = [...stages];
    const [movedStage] = nextStages.splice(fromIndex, 1);
    nextStages.splice(toIndex, 0, movedStage);
    onChange?.(nextStages);
  }

  /** 开始拖拽时记录当前卡片下标。 */
  function handleDragStart(event: DragEvent<HTMLDivElement>, index: number): void {
    setDraggingIndex(index);
    setDragOverIndex(index);
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', String(index));
  }

  /** 拖拽进入目标卡片时记录悬停位置。 */
  function handleDragOver(event: DragEvent<HTMLDivElement>, index: number): void {
    event.preventDefault();
    if (dragOverIndex !== index) {
      setDragOverIndex(index);
    }
    event.dataTransfer.dropEffect = 'move';
  }

  /** 放下时按目标位置重排。 */
  function handleDrop(event: DragEvent<HTMLDivElement>, index: number): void {
    event.preventDefault();
    const rawIndex: string = event.dataTransfer.getData('text/plain');
    const fromIndex: number = Number(rawIndex);
    if (Number.isFinite(fromIndex)) {
      moveStage(fromIndex, index);
    }
    setDraggingIndex(null);
    setDragOverIndex(null);
  }

  /** 拖拽结束后清理高亮状态。 */
  function handleDragEnd(): void {
    setDraggingIndex(null);
    setDragOverIndex(null);
  }

  /** 弹窗确认后写回阶段列表。 */
  function handleStageModalSubmit(values: QuestStageFormItem): void {
    const nextStages: QuestStageFormItem[] = [...stages];
    if (editingIndex === null) {
      nextStages.push(values);
    } else {
      nextStages[editingIndex] = values;
    }
    onChange?.(nextStages);
    setStageModalOpen(false);
    setEditingIndex(null);
    stageForm.resetFields();
  }

  return (
    <>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Typography.Text type="secondary">
          当前卡片顺序就是任务推进顺序；可直接拖拽卡片调整先后，阶段 ID 仅作为条件绑定标识。
        </Typography.Text>
        {stages.length > 0 ? (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {stages.map((stage, index) => (
              <div
                key={`${stage.objective_id}-${index}`}
                draggable
                onDragStart={(event) => handleDragStart(event, index)}
                onDragOver={(event) => handleDragOver(event, index)}
                onDrop={(event) => handleDrop(event, index)}
                onDragEnd={handleDragEnd}
                style={buildStageCardWrapperStyle(draggingIndex === index, dragOverIndex === index)}
              >
                <Card
                  size="small"
                  title={(
                    <Space size={8} wrap>
                      <Tag color="blue">{`阶段 ${index + 1}`}</Tag>
                      <Tag>{`ID ${stage.objective_id}`}</Tag>
                      <Tag color="purple">{formatDisplayLabel(QUEST_EVENT_TYPE_LABELS, stage.event_type)}</Tag>
                    </Space>
                  )}
                  extra={(
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
                  )}
                >
                  <Space direction="vertical" size={10} style={{ width: '100%' }}>
                    <Typography.Text strong>{renderStageDescription(stage)}</Typography.Text>
                    <Space size={[8, 8]} wrap>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('目标次数', String(stage.target_value || 1))}</Tag>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('目标 NPC', stage.npc_id > 0 ? String(stage.npc_id) : '-')}</Tag>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('目标场景', stage.scene_id > 0 ? String(stage.scene_id) : '-')}</Tag>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('菜单', stage.menu_entry_id > 0 ? String(stage.menu_entry_id) : '-')}</Tag>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('剧情', stage.dialogue_entry_id > 0 ? String(stage.dialogue_entry_id) : '-')}</Tag>
                      <Tag bordered={false} color="default">{formatStageMetaLabel('战斗类型', stage.battle_type.trim() || '-')}</Tag>
                    </Space>
                    {stage.guide_text.trim() ? (
                      <Typography.Text type="secondary">
                        {`引导：${stage.guide_text.trim()}`}
                      </Typography.Text>
                    ) : (
                      <Typography.Text type="secondary">未配置引导文案</Typography.Text>
                    )}
                  </Space>
                </Card>
              </div>
            ))}
          </Space>
        ) : (
          <Empty description="暂无任务阶段，请点击下方按钮添加" />
        )}
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
              <Form.Item label="阶段描述" name="description" rules={[{ required: true, message: '请输入阶段描述' }]} extra="可在下方预览中刷色。">
                <RichTextEditor rows={2} placeholder="例如：与市场理萌交谈" />
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
              <Form.Item label="引导文案" name="guide_text" extra="任务追踪/引导提示，可在下方预览中刷色。">
                <RichTextEditor rows={2} placeholder="例如：去商业区找市场理萌" />
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

function renderStageDescription(stage: QuestStageFormItem): string {
  const description: string = stage.description.trim();
  if (description) {
    return description;
  }
  return '未填写阶段描述';
}

function formatStageMetaLabel(label: string, value: string): string {
  return `${label}：${value}`;
}

function buildStageCardWrapperStyle(isDragging: boolean, isDragOver: boolean): CSSProperties {
  return {
    cursor: 'grab',
    borderRadius: 8,
    opacity: isDragging ? 0.65 : 1,
    boxShadow: isDragOver ? '0 0 0 2px #1677ff inset' : 'none',
  };
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
