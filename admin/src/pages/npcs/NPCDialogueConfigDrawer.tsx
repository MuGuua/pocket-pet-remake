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
    if (!entityId || !entryId || loading) {
      return;
    }
    if (detail) {
      editorForm.setFieldsValue(mapDialogueDetailToForm(detail, getNPCSpeakerName(npcName, entityId)));
      return;
    }
    if (detailMissing) {
      editorForm.setFieldsValue(defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)));
    }
  }, [entityId, entryId, entryTitle, npcName, detail, detailMissing, loading, editorForm]);

  useEffect(() => {
    if (!embedded || !onEmbeddedEditingChange) {
      return;
    }
    onEmbeddedEditingChange(embeddedEditing);
  }, [embedded, embeddedEditing, onEmbeddedEditingChange]);

  useEffect(() => {
    if (!embedded || !entityId || !entryId || loading) {
      return;
    }
    setEmbeddedEditing(true);
  }, [embedded, entityId, entryId, loading]);

  useEffect(() => {
    if (embedded || open) {
      return;
    }
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
    setEmbeddedEditing(true);
    editorForm.resetFields();
    editorForm.setFieldsValue(defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)));
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
    return (
      <Spin spinning={loading}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Text type="secondary">
              正在编辑 {detail ? detail.nodes.length : (editorForm.getFieldValue('nodes')?.length ?? 0)} 段剧情对白。
            </Typography.Text>
            <Button onClick={() => editorForm.setFieldsValue(detail ? mapDialogueDetailToForm(detail, getNPCSpeakerName(npcName, entityId)) : defaultDialogueFormValues(entityId, entryId, entryTitle, getNPCSpeakerName(npcName, entityId)))}>重置表单</Button>
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
        ) : (
          <Form form={editorForm} layout="vertical" onFinish={(values: DialogueEditorFormValues) => void handleSubmit(values)}>
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Descriptions bordered column={2} size="small">
                <Descriptions.Item label="实体ID">{entityId}</Descriptions.Item>
                <Descriptions.Item label="入口ID">{entryId}</Descriptions.Item>
                <Descriptions.Item label="剧情状态">
                  <Tag color={detail ? 'green' : 'default'}>{detail ? '已创建' : '未创建'}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="节点数">{detail?.nodes.length ?? (editorForm.getFieldValue('nodes')?.length ?? 0)}</Descriptions.Item>
              </Descriptions>
              <DialogueEditorFields hasExistingDetail={Boolean(detail)} npcSpeakerName={getNPCSpeakerName(npcName, entityId)} />
              <Space>
                <Button type="primary" loading={saving} onClick={() => editorForm.submit()}>
                  {detail ? '保存修改' : '创建配置'}
                </Button>
                <Button onClick={() => {
                  const resolvedEntityId: number = entityId ?? 0;
                  editorForm.setFieldsValue(
                    detail
                      ? mapDialogueDetailToForm(detail, getNPCSpeakerName(npcName, resolvedEntityId))
                      : defaultDialogueFormValues(resolvedEntityId, entryId, entryTitle, getNPCSpeakerName(npcName, resolvedEntityId)),
                  );
                }}>
                  重置表单
                </Button>
              </Space>
            </Space>
          </Form>
        )}
      </Drawer>
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
  const [draggingNodeIndex, setDraggingNodeIndex] = useState<number | null>(null);
  const [dragOverNodeIndex, setDragOverNodeIndex] = useState<number | null>(null);
  const [draggingOptionKey, setDraggingOptionKey] = useState<string | null>(null);
  const [dragOverOptionKey, setDragOverOptionKey] = useState<string | null>(null);
  const [expandedNodeKeys, setExpandedNodeKeys] = useState<Record<string, boolean>>({});
  const [expandedOptionKeys, setExpandedOptionKeys] = useState<Record<string, boolean>>({});

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

  // 节点 ID、线性跳转和排序都由当前表单列表顺序统一生成，避免运营手填引用关系。
  function applySequentialNodeLayout(nextNodes: AdminNPCDialogueNode[]): void {
    const normalizedNodes: AdminNPCDialogueNode[] = renumberDialogueNodes(nextNodes);
    editorForm.setFieldsValue({
      ...editorForm.getFieldsValue(),
      start_node_id: normalizedNodes.length > 0 ? normalizedNodes[0].node_id : '',
      nodes: normalizedNodes,
    });
  }

  // 在当前节点下方插入一条新节点，并立即顺延后续节点编号。
  function handleInsertNodeAfter(nodeIndex: number): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    nextNodes.splice(nodeIndex + 1, 0, defaultDialogueNode(currentNodes.length + 1, npcSpeakerName));
    applySequentialNodeLayout(nextNodes);
  }

  // 删除节点后重新压缩编号，保证始终保持“节点1、节点2 ...”连续。
  function handleRemoveNode(nodeIndex: number): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const nextNodes: AdminNPCDialogueNode[] = currentNodes.filter((_node: AdminNPCDialogueNode, index: number) => index !== nodeIndex);
    applySequentialNodeLayout(nextNodes);
  }

  // 复制当前节点，方便快速制作结构相近的剧情片段。
  function handleDuplicateNode(nodeIndex: number): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const sourceNode: AdminNPCDialogueNode | undefined = currentNodes[nodeIndex];
    if (!sourceNode) {
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    nextNodes.splice(nodeIndex + 1, 0, {
      ...sourceNode,
      node_id: '',
      next_node_id: '',
      sort_order: 0,
      options: (sourceNode.options ?? []).map((option: AdminNPCDialogueOption) => ({
        ...option,
        sort_order: 0,
      })),
    });
    applySequentialNodeLayout(nextNodes);
  }

  // 上移/下移节点后，节点编号、起始节点和线性跳转都会自动重排。
  function handleMoveNode(nodeIndex: number, direction: 'up' | 'down'): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const targetIndex: number = direction === 'up' ? nodeIndex - 1 : nodeIndex + 1;
    if (targetIndex < 0 || targetIndex >= currentNodes.length) {
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    [nextNodes[nodeIndex], nextNodes[targetIndex]] = [nextNodes[targetIndex], nextNodes[nodeIndex]];
    applySequentialNodeLayout(nextNodes);
  }

  // 分支选项也按当前列表顺序保存；上移/下移后自动回写顺序值。
  function handleMoveOption(nodeIndex: number, optionIndex: number, direction: 'up' | 'down'): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const targetIndex: number = direction === 'up' ? optionIndex - 1 : optionIndex + 1;
    const currentNode: AdminNPCDialogueNode | undefined = currentNodes[nodeIndex];
    if (!currentNode || targetIndex < 0 || targetIndex >= (currentNode.options ?? []).length) {
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    const nextOptions: AdminNPCDialogueOption[] = [...(nextNodes[nodeIndex].options ?? [])];
    [nextOptions[optionIndex], nextOptions[targetIndex]] = [nextOptions[targetIndex], nextOptions[optionIndex]];
    nextNodes[nodeIndex] = {
      ...nextNodes[nodeIndex],
      options: nextOptions.map((option: AdminNPCDialogueOption, index: number) => ({
        ...option,
        sort_order: index + 1,
      })),
    };
    applySequentialNodeLayout(nextNodes);
  }

  // 在当前选项下方插入，避免每次新增都只能追加到最后。
  function handleInsertOptionAfter(nodeIndex: number, optionIndex: number): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const currentNode: AdminNPCDialogueNode | undefined = currentNodes[nodeIndex];
    if (!currentNode) {
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    const nextOptions: AdminNPCDialogueOption[] = [...(currentNode.options ?? [])];
    nextOptions.splice(optionIndex + 1, 0, defaultDialogueOption(nextOptions.length + 1));
    nextNodes[nodeIndex] = {
      ...currentNode,
      options: nextOptions.map((option: AdminNPCDialogueOption, index: number) => ({
        ...option,
        sort_order: index + 1,
      })),
    };
    applySequentialNodeLayout(nextNodes);
  }

  function handleRemoveOption(nodeIndex: number, optionIndex: number): void {
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const currentNode: AdminNPCDialogueNode | undefined = currentNodes[nodeIndex];
    if (!currentNode) {
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    nextNodes[nodeIndex] = {
      ...currentNode,
      options: (currentNode.options ?? [])
        .filter((_option: AdminNPCDialogueOption, index: number) => index !== optionIndex)
        .map((option: AdminNPCDialogueOption, index: number) => ({
          ...option,
          sort_order: index + 1,
        })),
    };
    applySequentialNodeLayout(nextNodes);
  }

  function handleToggleNodeExpanded(nodeKey: string): void {
    setExpandedNodeKeys((previous: Record<string, boolean>) => ({
      ...previous,
      [nodeKey]: !previous[nodeKey],
    }));
  }

  function handleToggleOptionExpanded(optionKey: string): void {
    setExpandedOptionKeys((previous: Record<string, boolean>) => ({
      ...previous,
      [optionKey]: !previous[optionKey],
    }));
  }

  function buildOptionDragKey(nodeIndex: number, optionIndex: number): string {
    return `${nodeIndex}:${optionIndex}`;
  }

  function handleDropNode(targetIndex: number): void {
    if (draggingNodeIndex === null || draggingNodeIndex === targetIndex) {
      setDraggingNodeIndex(null);
      setDragOverNodeIndex(null);
      return;
    }
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    const [draggingNode] = nextNodes.splice(draggingNodeIndex, 1);
    nextNodes.splice(targetIndex, 0, draggingNode);
    applySequentialNodeLayout(nextNodes);
    setDraggingNodeIndex(null);
    setDragOverNodeIndex(null);
  }

  function handleDropOption(nodeIndex: number, targetOptionIndex: number): void {
    if (!draggingOptionKey) {
      setDragOverOptionKey(null);
      return;
    }
    const [sourceNodeIndexText, sourceOptionIndexText] = draggingOptionKey.split(':');
    const sourceNodeIndex: number = Number(sourceNodeIndexText);
    const sourceOptionIndex: number = Number(sourceOptionIndexText);
    if (Number.isNaN(sourceNodeIndex) || Number.isNaN(sourceOptionIndex)) {
      setDraggingOptionKey(null);
      setDragOverOptionKey(null);
      return;
    }
    if (sourceNodeIndex !== nodeIndex || sourceOptionIndex === targetOptionIndex) {
      setDraggingOptionKey(null);
      setDragOverOptionKey(null);
      return;
    }
    const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
    const currentNode: AdminNPCDialogueNode | undefined = currentNodes[nodeIndex];
    if (!currentNode) {
      setDraggingOptionKey(null);
      setDragOverOptionKey(null);
      return;
    }
    const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
    const nextOptions: AdminNPCDialogueOption[] = [...(currentNode.options ?? [])];
    const [draggingOption] = nextOptions.splice(sourceOptionIndex, 1);
    nextOptions.splice(targetOptionIndex, 0, draggingOption);
    nextNodes[nodeIndex] = {
      ...currentNode,
      options: nextOptions.map((option: AdminNPCDialogueOption, index: number) => ({
        ...option,
        sort_order: index + 1,
      })),
    };
    applySequentialNodeLayout(nextNodes);
    setDraggingOptionKey(null);
    setDragOverOptionKey(null);
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
            <Input disabled />
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
        {(fields) => (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {fields.map((field, index) => (
              (() => {
                const currentNode: AdminNPCDialogueNode = (editorForm.getFieldValue(['nodes', field.name]) ?? {}) as AdminNPCDialogueNode;
                const nodeKey: string = buildNodeCollapseKey(String(currentNode.node_id ?? ''), index);
                const expanded: boolean = expandedNodeKeys[nodeKey] ?? false;
                return (
                  <Card
                    key={field.key}
                    size="small"
                    draggable
                    onDragStart={(event) => {
                      event.dataTransfer.effectAllowed = 'move';
                      setDraggingNodeIndex(index);
                    }}
                    onDragOver={(event) => {
                      event.preventDefault();
                      if (draggingNodeIndex !== index) {
                        setDragOverNodeIndex(index);
                      }
                    }}
                    onDragLeave={() => {
                      if (dragOverNodeIndex === index) {
                        setDragOverNodeIndex(null);
                      }
                    }}
                    onDrop={(event) => {
                      event.preventDefault();
                      handleDropNode(index);
                    }}
                    onDragEnd={() => {
                      setDraggingNodeIndex(null);
                      setDragOverNodeIndex(null);
                    }}
                    style={dragOverNodeIndex === index ? { borderColor: '#fa8c16', background: '#fff7e6' } : undefined}
                    title={(
                      <Space direction="vertical" size={0}>
                        <Typography.Text strong>{`节点 ${index + 1}`}</Typography.Text>
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          {buildNodeCollapsedSummary(
                            currentNode,
                            index,
                            resolveLinearNodePreviewLabel(editorForm.getFieldValue('nodes') as AdminNPCDialogueNode[] | undefined, index, String(currentNode.node_type ?? 'line')),
                          )}
                        </Typography.Text>
                      </Space>
                    )}
                    extra={(
                      <Space size={4} wrap>
                        <Button size="small" type="link" onClick={() => handleToggleNodeExpanded(nodeKey)}>
                          {expanded ? '折叠' : '展开'}
                        </Button>
                        <Button size="small" type="link" onClick={() => handleMoveNode(index, 'up')} disabled={index === 0}>上移</Button>
                        <Button size="small" type="link" onClick={() => handleMoveNode(index, 'down')} disabled={index === fields.length - 1}>下移</Button>
                        <Button size="small" type="link" onClick={() => handleDuplicateNode(index)}>复制节点</Button>
                        <Button size="small" type="link" onClick={() => handleInsertNodeAfter(index)}>下方插入</Button>
                        <Button size="small" type="link" danger onClick={() => handleRemoveNode(index)}>删除节点</Button>
                      </Space>
                    )}
                  >
                    {expanded ? (
                      <>
                <Row gutter={12}>
                  <Col xs={24} md={8}>
                    <Form.Item label="节点ID" name={[field.name, 'node_id']} rules={[{ required: true, message: '请输入节点ID' }]}>
                      <Input disabled />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="节点类型" name={[field.name, 'node_type']} rules={[{ required: true, message: '请选择节点类型' }]}>
                      <Select options={nodeTypeOptions} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item label="当前顺序">
                      <Input value={`第 ${index + 1} 个`} disabled />
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
                            <Form.Item label="下一节点">
                              <Input
                                value={resolveLinearNodePreviewLabel(editorForm.getFieldValue('nodes') as AdminNPCDialogueNode[] | undefined, index, nodeType)}
                                disabled
                              />
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
                              {(optionFields) => (
                                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                                  {optionFields.map((optionField, optionIndex) => {
                                    const optionKey: string = buildOptionDragKey(field.name, optionIndex);
                                    const currentOption: AdminNPCDialogueOption = (editorForm.getFieldValue(['nodes', field.name, 'options', optionField.name]) ?? {}) as AdminNPCDialogueOption;
                                    const optionExpanded: boolean = expandedOptionKeys[optionKey] ?? false;
                                    return (
                                    <Card
                                      key={optionField.key}
                                      size="small"
                                      draggable
                                      onDragStart={(event) => {
                                        event.dataTransfer.effectAllowed = 'move';
                                        setDraggingOptionKey(optionKey);
                                      }}
                                      onDragOver={(event) => {
                                        event.preventDefault();
                                        if (draggingOptionKey !== optionKey) {
                                          setDragOverOptionKey(optionKey);
                                        }
                                      }}
                                      onDragLeave={() => {
                                        if (dragOverOptionKey === optionKey) {
                                          setDragOverOptionKey(null);
                                        }
                                      }}
                                      onDrop={(event) => {
                                        event.preventDefault();
                                        handleDropOption(field.name, optionIndex);
                                      }}
                                      onDragEnd={() => {
                                        setDraggingOptionKey(null);
                                        setDragOverOptionKey(null);
                                      }}
                                      style={dragOverOptionKey === optionKey ? { borderColor: '#fa8c16', background: '#fff7e6' } : undefined}
                                      bodyStyle={{ padding: 12 }}
                                      title={(
                                        <Space direction="vertical" size={0}>
                                          <Typography.Text strong>{`选项 ${optionIndex + 1}`}</Typography.Text>
                                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                                            {buildOptionCollapsedSummary(currentOption, optionIndex)}
                                          </Typography.Text>
                                        </Space>
                                      )}
                                      extra={(
                                        <Space size={4} wrap>
                                          <Button size="small" type="link" onClick={() => handleToggleOptionExpanded(optionKey)}>
                                            {optionExpanded ? '折叠' : '展开'}
                                          </Button>
                                          <Button size="small" type="link" onClick={() => handleMoveOption(field.name, optionIndex, 'up')} disabled={optionIndex === 0}>上移</Button>
                                          <Button size="small" type="link" onClick={() => handleMoveOption(field.name, optionIndex, 'down')} disabled={optionIndex === optionFields.length - 1}>下移</Button>
                                          <Button size="small" type="link" onClick={() => handleInsertOptionAfter(field.name, optionIndex)}>下方插入</Button>
                                          <Button size="small" type="link" danger onClick={() => handleRemoveOption(field.name, optionIndex)}>删除选项</Button>
                                        </Space>
                                      )}
                                    >
                                      {optionExpanded ? (
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
                                          <Form.Item label="当前顺序">
                                            <Input value={`第 ${optionIndex + 1} 个`} disabled />
                                          </Form.Item>
                                        </Col>
                                        <Col xs={24} md={12}>
                                          <Form.Item label="跳转到节点" name={[optionField.name, 'next_node_id']}>
                                            <Select
                                              allowClear
                                              placeholder="留空表示该分支直接结束剧情"
                                              options={buildNodeTargetOptions(editorForm.getFieldValue('nodes') as AdminNPCDialogueNode[] | undefined, field.name)}
                                            />
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
                                      ) : null}
                                    </Card>
                                    );
                                  })}
                                  <Button
                                    type="dashed"
                                    block
                                    onClick={() => {
                                      const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
                                      const currentNode: AdminNPCDialogueNode | undefined = currentNodes[field.name];
                                      if (!currentNode) {
                                        return;
                                      }
                                      const nextNodes: AdminNPCDialogueNode[] = [...currentNodes];
                                      nextNodes[field.name] = {
                                        ...currentNode,
                                        options: [...(currentNode.options ?? []), defaultDialogueOption(optionFields.length + 1)].map((option: AdminNPCDialogueOption, index: number) => ({
                                          ...option,
                                          sort_order: index + 1,
                                        })),
                                      };
                                      applySequentialNodeLayout(nextNodes);
                                    }}
                                  >
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
                      </>
                    ) : null}
                  </Card>
                );
              })()
            ))}
            <Button
              type="dashed"
              block
              onClick={() => {
                const currentNodes: AdminNPCDialogueNode[] = (editorForm.getFieldValue('nodes') ?? []) as AdminNPCDialogueNode[];
                applySequentialNodeLayout([...currentNodes, defaultDialogueNode(currentNodes.length + 1, npcSpeakerName)]);
              }}
            >
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
  return buildSequentialDialogueFormValues({
    entity_id: entityId,
    entry_id: entryId,
    dialogue_code: `${entryId}_code`,
    title: `${entryTitle || entryId}剧情`,
    start_node_id: '',
    version: 1,
    status: 1,
    nodes: [
      {
        node_id: '',
        node_type: 'line',
        speaker: npcSpeakerName,
        content: '这里填写第一句对白。',
        content_format: 'plain',
        portrait_key: '',
        next_node_id: '',
        client_animation_key: '',
        client_animation_block: false,
        sort_order: 0,
        conditions: {},
        effects: {},
        options: [],
      },
      {
        node_id: '',
        node_type: 'end',
        speaker: '',
        content: '',
        content_format: 'plain',
        portrait_key: '',
        next_node_id: '',
        client_animation_key: '',
        client_animation_block: false,
        sort_order: 0,
        conditions: {},
        effects: {},
        options: [],
      },
    ],
  });
}

function defaultDialogueNode(sortOrder: number, npcSpeakerName: string): AdminNPCDialogueNode {
  return {
    node_id: '',
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
    option_id: `选项${sortOrder}`,
    option_text: '',
    option_format: 'plain',
    next_node_id: '',
    sort_order: sortOrder,
    conditions: {},
  };
}

function mapDialogueDetailToForm(detail: AdminNPCDialogueDetail, npcSpeakerName: string): DialogueEditorFormValues {
  return buildSequentialDialogueFormValues({
    entity_id: detail.entity_id,
    entry_id: detail.entry_id,
    dialogue_code: detail.dialogue_code,
    title: detail.title,
    start_node_id: '',
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
  });
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
  return buildSequentialDialogueFormValues({
    entity_id: values.entity_id,
    entry_id: values.entry_id?.trim(),
    dialogue_code: values.dialogue_code.trim(),
    title: values.title.trim(),
    start_node_id: '',
    version: values.version > 0 ? values.version : 1,
    status: values.status,
    nodes: (values.nodes ?? []).map((node: AdminNPCDialogueNode) => ({
      node_id: node.node_id.trim(),
      node_type: node.node_type,
      speaker: node.speaker.trim(),
      content: node.content.trim(),
      content_format: node.content_format || 'plain',
      portrait_key: node.portrait_key.trim(),
      next_node_id: '',
      client_animation_key: node.client_animation_key.trim(),
      client_animation_block: Boolean(node.client_animation_block),
      sort_order: 0,
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
        sort_order: optionIndex + 1,
        conditions: {
          quest_id: option.conditions?.quest_id ?? 0,
          quest_state: option.conditions?.quest_state?.trim() ?? '',
        },
      })),
    })),
  });
}

// 节点 ID 固定按当前列表顺序生成，插入/删除后自动重排，避免手工维护引用关系。
function buildSequentialDialogueFormValues(values: DialogueEditorFormValues): DialogueEditorFormValues {
  const sequentialNodes: AdminNPCDialogueNode[] = renumberDialogueNodes(values.nodes ?? []);
  return {
    entity_id: values.entity_id,
    entry_id: values.entry_id?.trim(),
    dialogue_code: values.dialogue_code,
    title: values.title,
    start_node_id: sequentialNodes.length > 0 ? sequentialNodes[0].node_id : '',
    version: values.version,
    status: values.status,
    nodes: sequentialNodes,
  };
}

// 统一重建节点 ID、线性 next_node_id 和顺序值；分支选项目标会按旧 ID 映射到新 ID。
function renumberDialogueNodes(nodes: AdminNPCDialogueNode[]): AdminNPCDialogueNode[] {
  const idMap: Record<string, string> = {};
  nodes.forEach((node: AdminNPCDialogueNode, index: number) => {
    const previousID: string = node.node_id.trim();
    const nextID: string = buildSequentialNodeID(index);
    if (previousID !== '') {
      idMap[previousID] = nextID;
    }
  });
  return nodes.map((node: AdminNPCDialogueNode, index: number) => {
    const nextID: string = buildSequentialNodeID(index);
    const nextNodeID: string = resolveSequentialNextNodeID(nodes, index, node.node_type);
    return {
      ...node,
      node_id: nextID,
      next_node_id: nextNodeID,
      sort_order: index + 1,
      options: (node.options ?? []).map((option: AdminNPCDialogueOption, optionIndex: number) => ({
        ...option,
        option_id: option.option_id.trim() || `选项${optionIndex + 1}`,
        sort_order: optionIndex + 1,
        next_node_id: mapNodeTargetID(option.next_node_id, idMap),
      })),
    };
  });
}

// 线性节点统一跳到后一个节点；分支节点和结束节点不再单独录入下一节点。
function resolveSequentialNextNodeID(nodes: AdminNPCDialogueNode[], index: number, nodeType: string): string {
  if (nodeType === 'choice' || nodeType === 'end') {
    return '';
  }
  if (index + 1 >= nodes.length) {
    return '';
  }
  return buildSequentialNodeID(index + 1);
}

// 分支选项仍允许跳转到某个节点，但节点重编号后会自动改成新 ID。
function mapNodeTargetID(targetNodeID: string, idMap: Record<string, string>): string {
  const normalizedTarget: string = targetNodeID.trim();
  if (normalizedTarget === '') {
    return '';
  }
  return idMap[normalizedTarget] ?? normalizedTarget;
}

function buildSequentialNodeID(index: number): string {
  return `节点${index + 1}`;
}

function buildNodeCollapseKey(nodeID: string, index: number): string {
  return `${nodeID || 'node'}:${index}`;
}

function buildNodeCollapsedSummary(node: AdminNPCDialogueNode, index: number, nextNodeLabel: string): string {
  const nodeTypeLabel: string = formatNodeType(node.node_type || 'line');
  const speakerLabel: string = node.speaker?.trim() ? node.speaker.trim() : '系统';
  const contentPreview: string = (node.content || node.effects?.notice || node.client_animation_key || '').trim();
  const choiceCount: number = (node.options ?? []).length;
  const optionSummary: string = node.node_type === 'choice' ? ` · 分支${choiceCount}个` : '';
  const nextSummary: string = node.node_type === 'choice'
    ? ''
    : ` · 下一步${nextNodeLabel}`;
  if (contentPreview !== '') {
    return `${speakerLabel} · ${nodeTypeLabel}${optionSummary}${nextSummary} · ${contentPreview}`;
  }
  return `${speakerLabel} · ${nodeTypeLabel}${optionSummary}${nextSummary} · 第 ${index + 1} 个节点`;
}

function buildOptionCollapsedSummary(option: AdminNPCDialogueOption, index: number): string {
  const optionText: string = (option.option_text ?? '').trim();
  const targetLabel: string = (option.next_node_id ?? '').trim();
  const targetSummary: string = targetLabel !== '' ? ` · 跳到${targetLabel}` : ' · 直接结束';
  if (optionText !== '') {
    return `${optionText}${targetSummary}`;
  }
  return `第 ${index + 1} 个选项${targetSummary}`;
}

function resolveLinearNodePreviewLabel(nodes: AdminNPCDialogueNode[] | undefined, index: number, nodeType: string): string {
  if (nodeType === 'choice') {
    return '由分支选项决定';
  }
  if (nodeType === 'end') {
    return '剧情结束';
  }
  if (!nodes || index + 1 >= nodes.length) {
    return '剧情结束';
  }
  return buildSequentialNodeID(index + 1);
}

function buildNodeTargetOptions(nodes: AdminNPCDialogueNode[] | undefined, currentNodeIndex: number): Array<{ label: string; value: string }> {
  return (nodes ?? [])
    .map((node: AdminNPCDialogueNode, index: number) => ({
      label: `${buildSequentialNodeID(index)} · ${formatNodeType(node.node_type)}`,
      value: buildSequentialNodeID(index),
      disabled: index === currentNodeIndex,
    }))
    .filter((option: { label: string; value: string; disabled: boolean }) => !option.disabled);
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
