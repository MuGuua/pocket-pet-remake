import axios from 'axios';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { fetchAdminItems } from '../../services/item';
import {
  createAdminNPCDialogue,
  deleteAdminNPCDialogue,
  fetchAdminNPCDialogueDetail,
  updateAdminNPCDialogue,
} from '../../services/npc';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import type {
  AdminCreateNPCDialoguePayload,
  AdminNPCDialogueDetail,
  AdminNPCDialogueNode,
  AdminNPCDialogueOption,
  AdminUpdateNPCDialoguePayload,
} from '../../types/npc';
import type { AdminItemSummary } from '../../types/item';
import type React from 'react';

interface DialogueEditorFormValues {
  entity_id: number;
  entry_id?: string;
  dialogue_code: string;
  title: string;
  start_node_id: string;
  version: number;
  status: number;
  nodes: AdminNPCDialogueNode[];
}

interface NPCDialogueConfigDrawerProps {
  open: boolean;
  entityId: number | null;
  entryId: string;
  entryTitle: string;
  npcName?: string;
  embeddedFormId?: string;
  onEmbeddedEditingChange?: (editing: boolean) => void;
  onClose: () => void;
  embedded?: boolean;
}

const objectiveCompletedOptions: Array<{ label: string; value: string }> = [
  { label: '不限制', value: '' },
  { label: '要求目标已完成', value: 'true' },
  { label: '要求目标未完成', value: 'false' },
];

const editableStatusOptions: Array<{ label: string; value: number }> = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

const nodeTypeOptions: Array<{ label: string; value: string }> = [
  { label: '台词节点', value: 'line' },
  { label: '分支节点', value: 'choice' },
  { label: '动作节点', value: 'action' },
  { label: '结束节点', value: 'end' },
];

const questStateOptions: Array<{ label: string; value: string }> = [
  { label: '无限制', value: '' },
  { label: '可接取 AVAILABLE', value: 'AVAILABLE' },
  { label: '进行中 ACCEPTED', value: 'ACCEPTED' },
  { label: '可提交 READY_TO_SUBMIT', value: 'READY_TO_SUBMIT' },
  { label: '已完成 COMPLETED', value: 'COMPLETED' },
];

const contentFormatOptions: Array<{ label: string; value: string }> = [
  { label: '纯文本', value: 'plain' },
  { label: 'BBCode', value: 'bbcode' },
];

// 单条 NPC 菜单项对应一段剧情聚合配置；这里固定按 entity_id + entry_id 编辑，避免后台拆成多套页面。
export function NPCDialogueConfigDrawer({
  open,
  entityId,
  entryId,
  entryTitle,
  npcName,
  embeddedFormId,
  onEmbeddedEditingChange,
  onClose,
  embedded = false,
}: NPCDialogueConfigDrawerProps) {
  const [editorForm] = Form.useForm<DialogueEditorFormValues>();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<AdminNPCDialogueDetail | null>(null);
  const [detailMissing, setDetailMissing] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [embeddedEditing, setEmbeddedEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!entityId || !entryId) {
      return;
    }
    if (!embedded && !open) {
      return;
    }
    void loadDetail(entityId, entryId);
  }, [open, entityId, entryId, embedded]);

  useEffect(() => {
    if (!embedded || !entityId || !entryId) {
      return;
    }
    if (loading) {
      return;
    }
    if (detail) {
      editorForm.setFieldsValue(mapDialogueDetailToForm(detail, getNPCSpeakerName(npcName, entityId)));
      return;
    }
    if (detailMissing) {
      editorForm.setFieldsValue(defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)));
    }
  }, [embedded, entityId, entryId, entryTitle, npcName, detail, detailMissing, loading, editorForm]);

  useEffect(() => {
    if (!embedded || !onEmbeddedEditingChange) {
      return;
    }
    onEmbeddedEditingChange(embeddedEditing);
  }, [embedded, embeddedEditing, onEmbeddedEditingChange]);

  useEffect(() => {
    if (embedded || open) {
      return;
    }
    setEditorOpen(false);
    editorForm.resetFields();
  }, [open, editorForm, embedded]);

  const drawerTitle = useMemo((): string => {
    if (!entityId || !entryId) {
      return '剧情配置';
    }
    return `剧情配置 · ${entryTitle || entryId}（${entityId}/${entryId}）`;
  }, [entityId, entryId, entryTitle]);

  async function loadDetail(currentEntityId: number, currentEntryId: string): Promise<void> {
    setLoading(true);
    try {
      const result: AdminNPCDialogueDetail = await fetchAdminNPCDialogueDetail(currentEntityId, currentEntryId);
      setDetail(result);
      setDetailMissing(false);
    } catch (error: unknown) {
      if (axios.isAxiosError(error) && error.response?.status === 404) {
        setDetail(null);
        setDetailMissing(true);
      } else {
        message.error(error instanceof Error ? error.message : '加载 NPC 剧情配置失败');
      }
    } finally {
      setLoading(false);
    }
  }

  function handleOpenCreate(): void {
    if (!entityId) {
      return;
    }
    setEditorOpen(true);
    setEmbeddedEditing(true);
    editorForm.resetFields();
    editorForm.setFieldsValue(defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)));
  }

  function handleOpenEdit(): void {
    if (!detail) {
      return;
    }
    setEditorOpen(true);
    setEmbeddedEditing(true);
    editorForm.resetFields();
    editorForm.setFieldsValue(mapDialogueDetailToForm(detail, getNPCSpeakerName(npcName, entityId)));
  }

  async function handleSubmit(values: DialogueEditorFormValues): Promise<void> {
    if (!entityId || !entryId) {
      return;
    }
    setSaving(true);
    try {
      const normalizedValues: DialogueEditorFormValues = normalizeDialogueFormValues(values);
      if (normalizedValues.nodes.length === 0) {
        throw new Error('至少需要配置一个剧情节点');
      }
      if (detail) {
        await updateAdminNPCDialogue(entityId, entryId, mapDialogueFormToUpdatePayload(normalizedValues));
        message.success('NPC 剧情配置更新成功');
      } else {
        await createAdminNPCDialogue(mapDialogueFormToCreatePayload(normalizedValues));
        message.success('NPC 剧情配置创建成功');
      }
      if (!embedded) {
        setEditorOpen(false);
      }
      await loadDetail(entityId, entryId);
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '保存 NPC 剧情配置失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(): Promise<void> {
    if (!entityId || !entryId) {
      return;
    }
    setDeleting(true);
    try {
      await deleteAdminNPCDialogue(entityId, entryId);
      setDetail(null);
      setDetailMissing(true);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)));
      message.success('NPC 剧情配置已删除');
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '删除 NPC 剧情配置失败');
    } finally {
      setDeleting(false);
    }
  }

  if (embedded) {
    if (!entityId || !entryId) {
      return (
        <Empty
          description="请先保存菜单项后再配置剧情"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      );
    }
    if (!embeddedEditing) {
      return (
        <Spin spinning={loading}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Typography.Paragraph type="secondary">
              剧情与当前菜单项绑定（{entityId}/{entryId}）。先查看这段剧情包含的对白节点，再点击某一段进入编辑页。
            </Typography.Paragraph>
            {detail ? (
              <Card
                title={`剧情对话（${detail.nodes.length} 段）`}
                extra={<Button type="primary" onClick={handleOpenEdit}>进入编辑</Button>}
              >
                <Space direction="vertical" size={10} style={{ width: '100%' }}>
                  {detail.nodes.map((node: AdminNPCDialogueNode, index: number) => (
                    <Card
                      key={`${node.node_id}-${index}`}
                      size="small"
                      hoverable
                      onClick={handleOpenEdit}
                      title={`${index + 1}. ${node.speaker || '系统'} · ${formatNodeType(node.node_type)}`}
                      extra={<Tag>{node.node_id}</Tag>}
                    >
                      <Typography.Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 0 }}>
                        {node.content || node.effects?.notice || node.client_animation_key || '空节点'}
                      </Typography.Paragraph>
                    </Card>
                  ))}
                </Space>
              </Card>
            ) : detailMissing ? (
              <Empty description="当前菜单项还没有剧情配置" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                <Button type="primary" onClick={handleOpenCreate}>创建剧情</Button>
              </Empty>
            ) : null}
          </Space>
        </Spin>
      );
    }
    return (
      <Spin spinning={loading}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Text type="secondary">
              正在编辑 {detail ? detail.nodes.length : (editorForm.getFieldValue('nodes')?.length ?? 0)} 段剧情对白。
            </Typography.Text>
            <Button onClick={() => setEmbeddedEditing(false)}>返回列表</Button>
          </Space>
          <Form
            id={embeddedFormId}
            form={editorForm}
            layout="vertical"
            onFinish={(values: DialogueEditorFormValues) => void handleSubmit(values)}
          >
            <DialogueEditorFields hasExistingDetail={Boolean(detail)} npcSpeakerName={getNPCSpeakerName(npcName, entityId)} />
          </Form>
          {detail ? (
            <Button
              danger
              loading={deleting}
              onClick={() => {
                void Modal.confirm({
                  title: '确认删除这段剧情配置吗？',
                  content: '删除后该菜单项将失去服务端剧情数据。',
                  okText: '确认删除',
                  cancelText: '取消',
                  okButtonProps: { danger: true },
                  onOk: async () => {
                    await handleDelete();
                    setEmbeddedEditing(false);
                  },
                });
              }}
            >
              删除剧情
            </Button>
          ) : null}
        </Space>
      </Spin>
    );
  }

  return (
    <>
      <Drawer
        title={drawerTitle}
        width={920}
        open={open}
        onClose={onClose}
        destroyOnClose
        extra={detail ? (
          <Space>
            <Button onClick={handleOpenEdit}>编辑配置</Button>
            <Button
              danger
              loading={deleting}
              onClick={() => {
                void Modal.confirm({
                  title: '确认删除这段剧情配置吗？',
                  content: '删除后该菜单项将失去服务端剧情数据，正式环境请确认没有线上依赖。',
                  okText: '确认删除',
                  cancelText: '取消',
                  okButtonProps: { danger: true },
                  onOk: async () => {
                    await handleDelete();
                  },
                });
              }}
            >
              删除配置
            </Button>
          </Space>
        ) : (
          <Button type="primary" onClick={handleOpenCreate}>新建剧情配置</Button>
        )}
      >
        {loading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载剧情配置..." />
          </div>
        ) : detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item>
              <Descriptions.Item label="入口ID">{detail.entry_id}</Descriptions.Item>
              <Descriptions.Item label="剧情编码">{detail.dialogue_code || '-'}</Descriptions.Item>
              <Descriptions.Item label="剧情标题">{detail.title}</Descriptions.Item>
              <Descriptions.Item label="起始节点">{detail.start_node_id}</Descriptions.Item>
              <Descriptions.Item label="版本">{detail.version}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={detail.status === 1 ? 'green' : 'default'}>{detail.status === 1 ? '启用' : '停用'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="节点数">{detail.nodes.length}</Descriptions.Item>
            </Descriptions>
            <Card title="节点预览">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                {detail.nodes.map((node: AdminNPCDialogueNode, index: number) => (
                  <Card
                    key={`${node.node_id}-${index}`}
                    size="small"
                    title={`${index + 1}. ${node.node_id}`}
                    extra={<Tag>{formatNodeType(node.node_type)}</Tag>}
                  >
                    <Descriptions column={2} size="small">
                      <Descriptions.Item label="说话人">{node.speaker || '-'}</Descriptions.Item>
                      <Descriptions.Item label="排序">{node.sort_order}</Descriptions.Item>
                      <Descriptions.Item label="内容格式">{node.content_format || 'plain'}</Descriptions.Item>
                      <Descriptions.Item label="立绘Key">{node.portrait_key || '-'}</Descriptions.Item>
                      <Descriptions.Item label="下一节点">{node.next_node_id || '-'}</Descriptions.Item>
                      <Descriptions.Item label="动画Key">{node.client_animation_key || '-'}</Descriptions.Item>
                      <Descriptions.Item label="提示文案">{node.effects?.notice || '-'}</Descriptions.Item>
                      <Descriptions.Item label="任务事件">{node.effects?.quest_event || '-'}</Descriptions.Item>
                    </Descriptions>
                    {node.content ? (
                      <Typography.Paragraph style={{ marginTop: 12, marginBottom: 0 }}>
                        {node.content}
                      </Typography.Paragraph>
                    ) : null}
                    {node.options.length > 0 ? (
                      <>
                        <Divider style={{ margin: '12px 0' }}>选项</Divider>
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                          {node.options.map((option: AdminNPCDialogueOption) => (
                            <Card key={option.option_id} size="small" bodyStyle={{ padding: 12 }}>
                              <Descriptions column={2} size="small">
                                <Descriptions.Item label="选项ID">{option.option_id}</Descriptions.Item>
                                <Descriptions.Item label="排序">{option.sort_order}</Descriptions.Item>
                                <Descriptions.Item label="文案">{option.option_text}</Descriptions.Item>
                                <Descriptions.Item label="下一节点">{option.next_node_id || '-'}</Descriptions.Item>
                              </Descriptions>
                            </Card>
                          ))}
                        </Space>
                      </>
                    ) : null}
                  </Card>
                ))}
              </Space>
            </Card>
          </Space>
        ) : detailMissing ? (
          <Empty description="当前菜单项还没有剧情配置" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={handleOpenCreate}>立即创建</Button>
          </Empty>
        ) : null}
      </Drawer>

      <Modal
        title={detail ? `编辑剧情配置 · ${entryId}` : '新建剧情配置'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={1080}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={detail ? '保存修改' : '创建配置'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values: DialogueEditorFormValues) => void handleSubmit(values)}>
          <DialogueEditorFields hasExistingDetail={Boolean(detail)} npcSpeakerName={getNPCSpeakerName(npcName, entityId)} />
        </Form>
      </Modal>
    </>
  );
}

interface DialogueEditorFieldsProps {
  hasExistingDetail: boolean;
  npcSpeakerName: string;
}

// 剧情编辑表单字段，供独立抽屉与菜单合并 Tab 内嵌复用。
function DialogueEditorFields({ hasExistingDetail, npcSpeakerName }: DialogueEditorFieldsProps) {
  const editorForm = Form.useFormInstance<DialogueEditorFormValues>();
  const speakerOptions: Array<{ label: string; value: string }> = buildSpeakerOptions(npcSpeakerName);
  const [itemPickerOpen, setItemPickerOpen] = useState(false);
  const [itemPickerNodeIndex, setItemPickerNodeIndex] = useState<number | null>(null);
  const [itemKeyword, setItemKeyword] = useState('');
  const [itemPage, setItemPage] = useState(1);
  const [itemTotal, setItemTotal] = useState(0);
  const [itemLoading, setItemLoading] = useState(false);
  const [itemRows, setItemRows] = useState<AdminItemSummary[]>([]);
  const [itemPreviewMap, setItemPreviewMap] = useState<Record<number, AdminItemSummary>>({});

  useEffect(() => {
    if (!itemPickerOpen) {
      return;
    }
    void loadItemPickerRows(itemKeyword, itemPage);
  }, [itemPickerOpen, itemKeyword, itemPage]);

  async function loadItemPickerRows(keyword: string, page: number): Promise<void> {
    setItemLoading(true);
    try {
      const result = await fetchAdminItems({
        filters: { keyword, enabled: 'true' },
        page,
        pageSize: 48,
      });
      setItemRows(result.items);
      setItemTotal(result.total);
      setItemPreviewMap((previous) => mergeItemPreviewMap(previous, result.items));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统物品失败');
      setItemRows([]);
      setItemTotal(0);
    } finally {
      setItemLoading(false);
    }
  }

  function handleOpenItemPicker(nodeIndex: number): void {
    setItemPickerNodeIndex(nodeIndex);
    setItemPickerOpen(true);
  }

  function handleInsertPlayerName(nodeIndex: number): void {
    insertNodeContentToken(nodeIndex, '{player_name}');
  }

  function handleSelectItem(item: AdminItemSummary): void {
    if (itemPickerNodeIndex === null) {
      return;
    }
    setItemPreviewMap((previous) => mergeItemPreviewMap(previous, [item]));
    insertNodeContentToken(itemPickerNodeIndex, `{item:${item.item_id}}`);
    setItemPickerOpen(false);
  }

  function insertNodeContentToken(nodeIndex: number, token: string): void {
    const currentContent: string = String(editorForm.getFieldValue(['nodes', nodeIndex, 'content']) ?? '');
    const separator: string = currentContent === '' || currentContent.endsWith(' ') ? '' : ' ';
    editorForm.setFieldValue(['nodes', nodeIndex, 'content'], `${currentContent}${separator}${token}`);
  }

  return (
    <>
      <Row gutter={16}>
        <Col xs={24} md={8}>
          <Form.Item label="实体ID" name="entity_id">
            <InputNumber min={1} style={{ width: '100%' }} disabled />
          </Form.Item>
        </Col>
        {!hasExistingDetail ? (
          <Col xs={24} md={8}>
            <Form.Item label="入口ID" name="entry_id" rules={[{ required: true, message: '请输入入口ID' }]}>
              <Input />
            </Form.Item>
          </Col>
        ) : null}
        <Col xs={24} md={8}>
          <Form.Item label="剧情编码" name="dialogue_code">
            <Input placeholder="例如：market_intro_v2" />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="剧情标题" name="title" rules={[{ required: true, message: '请输入剧情标题' }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="起始节点ID" name="start_node_id" rules={[{ required: true, message: '请输入起始节点ID' }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={4}>
          <Form.Item label="版本号" name="version">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={4}>
          <Form.Item label="状态" name="status">
            <Select options={editableStatusOptions} />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left">节点配置</Divider>
      <Typography.Paragraph type="secondary">
        每个节点对应一条服务端剧情配置；对白支持 {'{player_name}'} 和 {'{item:物品ID}'} 占位符，服务端会用玩家名、物品名与物品 icon 渲染。
      </Typography.Paragraph>

      <Form.List name="nodes">
        {(fields, { add, remove }) => (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {fields.map((field, index) => (
              <Card
                key={field.key}
                size="small"
                title={`节点 ${index + 1}`}
                extra={(
                  <Space>
                    <Button type="link" onClick={() => add(defaultDialogueNode(index + 2, npcSpeakerName), index + 1)}>下方插入</Button>
                    <Button type="link" danger onClick={() => remove(field.name)}>删除节点</Button>
                  </Space>
                )}
              >
                <Row gutter={12}>
                  <Col xs={24} md={8}>
                    <Form.Item label="节点ID" name={[field.name, 'node_id']} rules={[{ required: true, message: '请输入节点ID' }]}>
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="节点类型" name={[field.name, 'node_type']} rules={[{ required: true, message: '请选择节点类型' }]}>
                      <Select options={nodeTypeOptions} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="排序" name={[field.name, 'sort_order']}>
                      <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                </Row>

                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item label="条件任务ID" name={[field.name, 'conditions', 'quest_id']}>
                      <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不限制" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item label="条件任务状态" name={[field.name, 'conditions', 'quest_state']}>
                      <Select options={questStateOptions} allowClear placeholder="留空表示不限制" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item label="条件目标ID" name={[field.name, 'conditions', 'objective_id']} tooltip="分阶段任务时填写 objective_id">
                      <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不限制" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item
                      label="目标完成状态"
                      name={[field.name, 'conditions', 'objective_completed']}
                      getValueProps={(value: boolean | undefined) => ({
                        value: value === true ? 'true' : value === false ? 'false' : '',
                      })}
                      normalize={(value: string) => {
                        if (value === 'true') {
                          return true;
                        }
                        if (value === 'false') {
                          return false;
                        }
                        return undefined;
                      }}
                    >
                      <Select options={objectiveCompletedOptions} placeholder="留空表示不限制" />
                    </Form.Item>
                  </Col>
                </Row>

                <Row gutter={12}>
                  <Col xs={24} md={8}>
                    <Form.Item label="进入节点提示" name={[field.name, 'effects', 'notice']}>
                      <Input placeholder="例如：理萌把货箱挪开了" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="任务事件Key" name={[field.name, 'effects', 'quest_event']}>
                      <Input placeholder="例如：TALK_TO_NPC" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="接取任务ID" name={[field.name, 'effects', 'accept_quest_id']}>
                      <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不接取" />
                    </Form.Item>
                  </Col>
                </Row>

                <Form.List name={[field.name, 'effects', 'grant_items']}>
                  {(grantFields, grantOperations) => (
                    <Space direction="vertical" size={8} style={{ width: '100%', marginBottom: 12 }}>
                      <Typography.Text type="secondary">进入节点发放物品（服务端权威发奖）</Typography.Text>
                      {grantFields.map((grantField, grantIndex) => (
                        <Row gutter={12} key={grantField.key} align="middle">
                          <Col xs={24} md={10}>
                            <Form.Item label={`物品 ${grantIndex + 1} ID`} name={[grantField.name, 'item_id']} style={{ marginBottom: 0 }}>
                              <InputNumber min={1} style={{ width: '100%' }} placeholder="物品模板 ID" />
                            </Form.Item>
                          </Col>
                          <Col xs={24} md={10}>
                            <Form.Item label="数量" name={[grantField.name, 'quantity']} style={{ marginBottom: 0 }}>
                              <InputNumber min={1} style={{ width: '100%' }} />
                            </Form.Item>
                          </Col>
                          <Col xs={24} md={4}>
                            <Button type="link" danger onClick={() => grantOperations.remove(grantField.name)}>删除</Button>
                          </Col>
                        </Row>
                      ))}
                      <Button type="dashed" block onClick={() => grantOperations.add({ item_id: undefined, quantity: 1 })}>
                        新增发放物品
                      </Button>
                    </Space>
                  )}
                </Form.List>

                <Form.Item noStyle shouldUpdate>
                  {() => {
                    const nodeType: string = editorForm.getFieldValue(['nodes', field.name, 'node_type']) ?? 'line';
                    const isChoiceNode: boolean = nodeType === 'choice';
                    const isActionNode: boolean = nodeType === 'action';
                    const isEndNode: boolean = nodeType === 'end';
                    const needsSpeakerFields: boolean = !isActionNode && !isEndNode;
                    const needsNextNode: boolean = nodeType === 'line' || nodeType === 'action';

                    return (
                      <>
                        <Row gutter={12}>
                          <Col xs={24} md={8}>
                            <Form.Item label="说话人" name={[field.name, 'speaker']}>
                              <Select
                                disabled={!needsSpeakerFields}
                                options={speakerOptions}
                                placeholder={needsSpeakerFields ? '选择当前 NPC 或玩家' : '当前节点无需配置'}
                              />
                            </Form.Item>
                          </Col>
                          <Col xs={24} md={8}>
                            <Form.Item label="内容格式" name={[field.name, 'content_format']}>
                              <Select options={contentFormatOptions} disabled={isActionNode || isEndNode} />
                            </Form.Item>
                          </Col>
                          <Col xs={24} md={8}>
                            <Form.Item label="立绘Key" name={[field.name, 'portrait_key']}>
                              <Input disabled={!needsSpeakerFields} placeholder={needsSpeakerFields ? 'NPC: npc_limeng_smile；玩家: player_default' : '当前节点无需配置'} />
                            </Form.Item>
                          </Col>
                        </Row>
                        <Row gutter={12}>
                          <Col xs={24} md={16}>
                            <Form.Item
                              label="对白/说明内容"
                              name={[field.name, 'content']}
                              extra="可写：欢迎 {player_name}，这是 {item:1001}。物品名和 icon 会从 item_definition 读取。"
                            >
                              <Input.TextArea rows={isChoiceNode ? 3 : 4} disabled={isActionNode || isEndNode} placeholder={isChoiceNode ? '给玩家看的分支提示文案' : '填写本节点要显示的台词'} />
                            </Form.Item>
                            {isActionNode || isEndNode ? null : (
                              <Space direction="vertical" size={8} style={{ width: '100%', marginBottom: 16 }}>
                                <Space wrap>
                                  <Button size="small" onClick={() => handleOpenItemPicker(field.name)}>导入物品</Button>
                                  <Button size="small" onClick={() => handleInsertPlayerName(field.name)}>@玩家</Button>
                                </Space>
                                <Form.Item noStyle shouldUpdate>
                                  {() => (
                                    <DialogueContentPreview
                                      content={String(editorForm.getFieldValue(['nodes', field.name, 'content']) ?? '')}
                                      itemMap={itemPreviewMap}
                                    />
                                  )}
                                </Form.Item>
                              </Space>
                            )}
                          </Col>
                          <Col xs={24} md={8}>
                            <Form.Item label="下一节点ID" name={[field.name, 'next_node_id']}>
                              <Input disabled={!needsNextNode} placeholder={needsNextNode ? '例如：next_step' : 'choice/end 节点由分支或结束控制'} />
                            </Form.Item>
                            <Form.Item label="动画Key" name={[field.name, 'client_animation_key']}>
                              <Input disabled={!isActionNode} placeholder={isActionNode ? '例如：market_limeng_step_aside' : '仅动作节点使用'} />
                            </Form.Item>
                            <Form.Item label="是否阻塞等待动画播完" name={[field.name, 'client_animation_block']}>
                              <Select disabled={!isActionNode} options={[{ label: '否', value: false }, { label: '是', value: true }]} />
                            </Form.Item>
                          </Col>
                        </Row>

                        {isChoiceNode ? (
                          <>
                            <Divider style={{ margin: '8px 0 16px 0' }}>分支选项</Divider>
                            <Form.List name={[field.name, 'options']}>
                              {(optionFields, optionOperations) => (
                                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                                  {optionFields.map((optionField, optionIndex) => (
                                    <Card
                                      key={optionField.key}
                                      size="small"
                                      bodyStyle={{ padding: 12 }}
                                      title={`选项 ${optionIndex + 1}`}
                                      extra={<Button type="link" danger onClick={() => optionOperations.remove(optionField.name)}>删除选项</Button>}
                                    >
                                      <Row gutter={12}>
                                        <Col xs={24} md={6}>
                                          <Form.Item label="选项ID" name={[optionField.name, 'option_id']} rules={[{ required: true, message: '请输入选项ID' }]}>
                                            <Input />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={10}>
                                          <Form.Item label="选项文案" name={[optionField.name, 'option_text']} rules={[{ required: true, message: '请输入选项文案' }]}>
                                            <Input />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={4}>
                                          <Form.Item label="格式" name={[optionField.name, 'option_format']}>
                                            <Select options={contentFormatOptions} />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={4}>
                                          <Form.Item label="排序" name={[optionField.name, 'sort_order']}>
                                            <InputNumber min={1} style={{ width: '100%' }} />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={12}>
                                          <Form.Item label="下一节点ID" name={[optionField.name, 'next_node_id']}>
                                            <Input placeholder="留空表示该分支直接结束剧情" />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={6}>
                                          <Form.Item label="条件任务ID" name={[optionField.name, 'conditions', 'quest_id']}>
                                            <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不限制" />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={6}>
                                          <Form.Item label="条件任务状态" name={[optionField.name, 'conditions', 'quest_state']}>
                                            <Select options={questStateOptions} allowClear placeholder="留空表示不限制" />
                                          </Form.Item>
                                        </Col>
                                      </Row>
                                    </Card>
                                  ))}
                                  <Button type="dashed" block onClick={() => optionOperations.add(defaultDialogueOption(optionFields.length + 1))}>
                                    新增选项
                                  </Button>
                                </Space>
                              )}
                            </Form.List>
                          </>
                        ) : null}
                      </>
                    );
                  }}
                </Form.Item>
              </Card>
            ))}
            <Button type="dashed" block onClick={() => add(defaultDialogueNode(fields.length + 1, npcSpeakerName))}>
              新增节点
            </Button>
          </Space>
        )}
      </Form.List>
      <Modal
        title="导入系统物品"
        open={itemPickerOpen}
        onCancel={() => setItemPickerOpen(false)}
        footer={null}
        width={860}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        destroyOnClose
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Input.Search
            allowClear
            placeholder="搜索物品ID、编码或名称"
            enterButton="搜索"
            onSearch={(value) => {
              setItemKeyword(value);
              setItemPage(1);
            }}
          />
          <Spin spinning={itemLoading}>
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
              {itemRows.map((item) => (
                <Tooltip
                  key={item.item_id}
                  title={(
                    <Space direction="vertical" size={2}>
                      <span>{item.item_name}</span>
                      <span>ID：{item.item_id}</span>
                      <span>{item.desc || '暂无介绍'}</span>
                    </Space>
                  )}
                >
                  <Button
                    onClick={() => handleSelectItem(item)}
                    style={{ height: 112, padding: 8, whiteSpace: 'normal' }}
                  >
                    <Space direction="vertical" size={6} align="center" style={{ width: '100%' }}>
                      <ItemIcon icon={item.icon} />
                      <Typography.Text ellipsis style={{ width: '100%', textAlign: 'center', fontSize: 12 }}>
                        {item.item_name}
                      </Typography.Text>
                    </Space>
                  </Button>
                </Tooltip>
              ))}
            </div>
          </Spin>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Text type="secondary">共 {itemTotal} 个系统物品</Typography.Text>
            <Space>
              <Button disabled={itemPage <= 1} onClick={() => setItemPage((value) => Math.max(1, value - 1))}>上一页</Button>
              <Typography.Text>第 {itemPage} 页</Typography.Text>
              <Button disabled={itemPage * 48 >= itemTotal} onClick={() => setItemPage((value) => value + 1)}>下一页</Button>
            </Space>
          </Space>
        </Space>
      </Modal>
    </>
  );
}

// 默认值按服务端聚合结构直接生成，减少第一次录入时的样板字段。
function defaultDialogueFormValues(entityId: number, entryId: string, entryTitle: string, npcSpeakerName: string): DialogueEditorFormValues {
  return {
    entity_id: entityId,
    entry_id: entryId,
    dialogue_code: `${entryId}_code`,
    title: `${entryTitle || entryId}剧情`,
    start_node_id: 'start',
    version: 1,
    status: 1,
    nodes: [
      {
        node_id: 'start',
        node_type: 'line',
        speaker: npcSpeakerName,
        content: '这里填写第一句对白。',
        content_format: 'plain',
        portrait_key: '',
        next_node_id: 'end',
        client_animation_key: '',
        client_animation_block: false,
        sort_order: 1,
        conditions: {},
        effects: {},
        options: [],
      },
      {
        node_id: 'end',
        node_type: 'end',
        speaker: '',
        content: '',
        content_format: 'plain',
        portrait_key: '',
        next_node_id: '',
        client_animation_key: '',
        client_animation_block: false,
        sort_order: 2,
        conditions: {},
        effects: {},
        options: [],
      },
    ],
  };
}

function defaultDialogueNode(sortOrder: number, npcSpeakerName: string): AdminNPCDialogueNode {
  return {
    node_id: `node_${sortOrder}`,
    node_type: 'line',
    speaker: npcSpeakerName,
    content: '',
    content_format: 'plain',
    portrait_key: '',
    next_node_id: '',
    client_animation_key: '',
    client_animation_block: false,
    sort_order: sortOrder,
    conditions: {},
    effects: {},
    options: [],
  };
}

function defaultDialogueOption(sortOrder: number): AdminNPCDialogueOption {
  return {
    option_id: `option_${sortOrder}`,
    option_text: '',
    option_format: 'plain',
    next_node_id: '',
    sort_order: sortOrder,
    conditions: {},
  };
}

function mapDialogueDetailToForm(detail: AdminNPCDialogueDetail, npcSpeakerName: string): DialogueEditorFormValues {
  return {
    entity_id: detail.entity_id,
    entry_id: detail.entry_id,
    dialogue_code: detail.dialogue_code,
    title: detail.title,
    start_node_id: detail.start_node_id,
    version: detail.version,
    status: detail.status,
    nodes: detail.nodes.map((node: AdminNPCDialogueNode) => ({
      ...node,
      speaker: normalizeSpeakerForForm(node.speaker, npcSpeakerName),
      content_format: node.content_format || 'plain',
      conditions: node.conditions ?? {},
      effects: node.effects ?? {},
      options: node.options.map((option: AdminNPCDialogueOption) => ({
        ...option,
        option_format: option.option_format || 'plain',
        conditions: option.conditions ?? {},
      })),
    })),
  };
}

// 说话人固定为当前 NPC 和玩家，避免运营录入出无法统一展示的自由文本。
function buildSpeakerOptions(npcSpeakerName: string): Array<{ label: string; value: string }> {
  return [
    { label: npcSpeakerName, value: npcSpeakerName },
    { label: '玩家', value: '玩家' },
  ];
}

// 独立剧情列表页可能拿不到 NPC 名称，此时用实体 ID 生成可识别的兜底显示值。
function getNPCSpeakerName(npcName: string | undefined, entityId: number | null): string {
  const trimmedName: string = (npcName ?? '').trim();
  if (trimmedName !== '') {
    return trimmedName;
  }
  if (entityId && entityId > 0) {
    return `NPC ${entityId}`;
  }
  return '当前NPC';
}

// 兼容历史数据中存过的通用 NPC 占位符，新保存时会改成当前 NPC 名。
function normalizeSpeakerForForm(speaker: string, npcSpeakerName: string): string {
  const trimmedSpeaker: string = speaker.trim();
  if (trimmedSpeaker === 'NPC' || trimmedSpeaker === '') {
    return npcSpeakerName;
  }
  return trimmedSpeaker;
}

function mergeItemPreviewMap(previous: Record<number, AdminItemSummary>, items: AdminItemSummary[]): Record<number, AdminItemSummary> {
  const nextMap: Record<number, AdminItemSummary> = { ...previous };
  items.forEach((item) => {
    nextMap[item.item_id] = item;
  });
  return nextMap;
}

function DialogueContentPreview({ content, itemMap }: { content: string; itemMap: Record<number, AdminItemSummary> }) {
  const fragments = renderDialogueContentFragments(content, itemMap);
  return (
    <Card size="small" title="对话预览（客户端最终效果）" bodyStyle={{ padding: 12 }}>
      <Space wrap size={6}>
        {fragments.length > 0 ? fragments : <Typography.Text type="secondary">暂无对白内容</Typography.Text>}
      </Space>
    </Card>
  );
}

function renderDialogueContentFragments(content: string, itemMap: Record<number, AdminItemSummary>): React.ReactNode[] {
  const fragments: React.ReactNode[] = [];
  const tokenPattern = /(\{player_name\}|\{item:(\d+)\})/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null = tokenPattern.exec(content);
  while (match) {
    if (match.index > lastIndex) {
      fragments.push(<span key={`text-${lastIndex}`}>{content.slice(lastIndex, match.index)}</span>);
    }
    if (match[1] === '{player_name}') {
      fragments.push(<Typography.Text key={`player-${match.index}`} strong>玩家</Typography.Text>);
    } else {
      const itemID = Number(match[2]);
      const item = itemMap[itemID];
      fragments.push(<ItemMentionChip key={`item-${match.index}-${itemID}`} itemID={itemID} item={item} />);
    }
    lastIndex = match.index + match[0].length;
    match = tokenPattern.exec(content);
  }
  if (lastIndex < content.length) {
    fragments.push(<span key={`text-${lastIndex}`}>{content.slice(lastIndex)}</span>);
  }
  return fragments;
}

function ItemMentionChip({ itemID, item }: { itemID: number; item?: AdminItemSummary }) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, verticalAlign: 'middle' }}>
      <ItemIcon icon={item?.icon ?? ''} size={20} />
      <Typography.Text strong>{item?.item_name ?? `物品${itemID}`}</Typography.Text>
    </span>
  );
}

function ItemIcon({ icon, size = 32 }: { icon: string; size?: number }) {
  const imageSrc = toAdminImageSrc(icon);
  if (!imageSrc) {
    return (
      <span
        style={{
          width: size,
          height: size,
          display: 'inline-grid',
          placeItems: 'center',
          borderRadius: 6,
          background: '#f5f5f5',
          border: '1px solid #d9d9d9',
          fontSize: Math.max(12, Math.floor(size * 0.5)),
        }}
      >
        物
      </span>
    );
  }
  return <img src={imageSrc} alt="" style={{ width: size, height: size, objectFit: 'contain' }} />;
}

function toAdminImageSrc(icon: string): string {
  const trimmedIcon = icon.trim();
  if (trimmedIcon === '' || trimmedIcon.startsWith('res://')) {
    return '';
  }
  return trimmedIcon;
}

// 这里统一清洗表单结构，确保空白输入不会直接以脏数据写回数据库。
function normalizeDialogueFormValues(values: DialogueEditorFormValues): DialogueEditorFormValues {
  return {
    entity_id: values.entity_id,
    entry_id: values.entry_id?.trim(),
    dialogue_code: values.dialogue_code.trim(),
    title: values.title.trim(),
    start_node_id: values.start_node_id.trim(),
    version: values.version > 0 ? values.version : 1,
    status: values.status,
    nodes: (values.nodes ?? []).map((node: AdminNPCDialogueNode, nodeIndex: number) => ({
      node_id: node.node_id.trim(),
      node_type: node.node_type,
      speaker: node.speaker.trim(),
      content: node.content.trim(),
      content_format: node.content_format || 'plain',
      portrait_key: node.portrait_key.trim(),
      next_node_id: node.next_node_id.trim(),
      client_animation_key: node.client_animation_key.trim(),
      client_animation_block: Boolean(node.client_animation_block),
      sort_order: node.sort_order > 0 ? node.sort_order : nodeIndex + 1,
      conditions: {
        quest_id: node.conditions?.quest_id ?? 0,
        quest_state: node.conditions?.quest_state?.trim() ?? '',
        objective_id: node.conditions?.objective_id ?? 0,
        objective_completed: node.conditions?.objective_completed,
      },
      effects: {
        notice: node.effects?.notice?.trim() ?? '',
        quest_event: node.effects?.quest_event?.trim() ?? '',
        accept_quest_id: node.effects?.accept_quest_id ?? 0,
        grant_items: (node.effects?.grant_items ?? [])
          .filter((item) => (item.item_id ?? 0) > 0)
          .map((item) => ({
            item_id: item.item_id ?? 0,
            quantity: item.quantity && item.quantity > 0 ? item.quantity : 1,
          })),
      },
      options: (node.options ?? []).map((option: AdminNPCDialogueOption, optionIndex: number) => ({
        option_id: option.option_id.trim(),
        option_text: option.option_text.trim(),
        option_format: option.option_format || 'plain',
        next_node_id: option.next_node_id.trim(),
        sort_order: option.sort_order > 0 ? option.sort_order : optionIndex + 1,
        conditions: {
          quest_id: option.conditions?.quest_id ?? 0,
          quest_state: option.conditions?.quest_state?.trim() ?? '',
        },
      })),
    })),
  };
}

function mapDialogueFormToCreatePayload(values: DialogueEditorFormValues): AdminCreateNPCDialoguePayload {
  return {
    entity_id: values.entity_id,
    entry_id: values.entry_id ?? '',
    dialogue_code: values.dialogue_code,
    title: values.title,
    start_node_id: values.start_node_id,
    version: values.version,
    status: values.status,
    nodes: values.nodes,
  };
}

function mapDialogueFormToUpdatePayload(values: DialogueEditorFormValues): AdminUpdateNPCDialoguePayload {
  return {
    entity_id: values.entity_id,
    dialogue_code: values.dialogue_code,
    title: values.title,
    start_node_id: values.start_node_id,
    version: values.version,
    status: values.status,
    nodes: values.nodes,
  };
}

function formatNodeType(nodeType: string): string {
  if (nodeType === 'line') {
    return '台词';
  }
  if (nodeType === 'choice') {
    return '分支';
  }
  if (nodeType === 'action') {
    return '动作';
  }
  if (nodeType === 'end') {
    return '结束';
  }
  return nodeType;
}
