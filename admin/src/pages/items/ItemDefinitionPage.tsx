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
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { ConsumableEffectEditor } from '../../components/ConsumableEffectEditor';
import { EnhanceMaterialConfigEditor } from '../../components/EnhanceMaterialConfigEditor';
import { GiftBoxRewardEditor, formatGiftBoxRewardRowSummary } from '../../components/GiftBoxRewardEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import { RichTextEditor } from '../../components/RichTextEditor';
import { createAdminItem, deleteAdminItem, fetchAdminItemDetail, fetchAdminItems, updateAdminItem } from '../../services/item';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import type { AdminItemDetail, AdminItemListFilters, AdminItemSummary } from '../../types/item';
import type { ConsumableEffectEntry } from '../../types/consumableEffect';
import {
  formatConsumableEffectCategoryLabel,
  formatConsumableEffectEntryLabel,
  parseConsumableEffects,
} from '../../types/consumableEffect';
import { parseGiftBoxRewards, type GiftBoxRewardEntry } from '../../types/giftBoxReward';
import { defaultEnhanceMaterialConfig, formatEnhanceMaterialConfigSummary } from '../../types/enhanceMaterialConfig';
import { buildFilterSelectOptions, buildSelectOptions, formatDisplayLabel, ITEM_SUB_TYPE_LABELS, ITEM_TYPE_LABELS, MATERIAL_ITEM_SUB_TYPE_LABELS } from '../../utils/displayLabels';
import {
  formatConsumableEffectSummary,
  formatGiftRewardSummary,
  mapItemDetailToFormValues,
  mapItemFormToPayload,
  type ItemEditorFormValues,
} from '../../utils/itemFormMapping';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface ItemDefinitionPageProps {
  excludeItemType?: string;
}

// 物品模板页直接面向正式数据库模板，所有字段修改都会影响后续背包、掉落、商店和奖励链路。
export function ItemDefinitionPage({ excludeItemType }: ItemDefinitionPageProps) {
  const [filterForm] = Form.useForm<AdminItemListFilters>();
  const [editorForm] = Form.useForm<ItemEditorFormValues>();
  const watchedItemType = Form.useWatch('item_type', editorForm);
  const watchedItemSubType = Form.useWatch('item_sub_type', editorForm);
  const [filters, setFilters] = useState<AdminItemListFilters>({});
  const [rows, setRows] = useState<AdminItemSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminItemDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminItemDetail | null>(null);
  const [saving, setSaving] = useState(false);

  const isGiftBoxForm = watchedItemType === 'box';
  const isMaterialForm = watchedItemType === 'material';
  const isEnhanceMaterialForm = isMaterialForm && watchedItemSubType === 'equipment_enhance';

  useEffect(() => {
    void loadItems(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadItems(nextFilters: AdminItemListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminItems({
        filters: {
          ...nextFilters,
          exclude_item_type: excludeItemType,
        },
        page: nextPage,
        pageSize: nextPageSize,
      });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(itemID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminItemDetail(itemID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', itemID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      const nextItemID = await loadNextItemID();
      editorForm.setFieldsValue(defaultItemValues(nextItemID));
      return;
    }
    if (!itemID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminItemDetail(itemID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapItemDetailToFormValues(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: ItemEditorFormValues) {
    setSaving(true);
    const fullValues = editorForm.getFieldsValue(true) as ItemEditorFormValues;
    const payload = mapItemFormToPayload({ ...fullValues, ...values });
    if (payload.item_type === 'box' && parseGiftBoxRewards(payload.effect_params_json).length === 0) {
      message.error('礼包至少需要配置一条奖励内容');
      setSaving(false);
      return;
    }
    try {
      if (editingRecord) {
        await updateAdminItem(editingRecord.item_id, payload);
        message.success(`物品模板已更新：ID ${editingRecord.item_id} / ${payload.item_code}`);
      } else {
        const created = await createAdminItem(payload);
        message.success(`物品模板已创建：ID ${created.item_id} / ${created.item_code}`);
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadItems(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存物品模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(itemID: number) {
    try {
      await deleteAdminItem(itemID);
      message.success('物品模板已删除');
      if (detail?.item_id === itemID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadItems(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除物品模板失败');
    }
  }

  async function loadNextItemID(): Promise<number> {
    try {
      const result = await fetchAdminItems({ filters: {}, page: 1, pageSize: 1 });
      const currentMaxItemID = result.items[0]?.item_id ?? 10000;
      return currentMaxItemID + 1;
    } catch (error) {
      message.error(error instanceof Error ? error.message : '获取下一个物品ID失败');
      return 10001;
    }
  }

  const detailGiftRewards = useMemo(
    () => (detail ? parseGiftBoxRewards(detail.effect_params_json) : []),
    [detail],
  );

  const detailUseEffects = useMemo(
    () => (detail && detail.item_type !== 'box'
      ? parseConsumableEffects(detail.effect_type, detail.effect_value, detail.effect_params_json)
      : []),
    [detail],
  );

  const detailUseEffectColumns = useMemo<ColumnsType<ConsumableEffectEntry>>(
    () => [
      {
        title: '大类',
        dataIndex: 'category',
        key: 'category',
        width: 90,
        render: (category: ConsumableEffectEntry['category']) => formatConsumableEffectCategoryLabel(category),
      },
      {
        title: '效果',
        key: 'effect',
        render: (_value, record) => formatConsumableEffectEntryLabel(record),
      },
    ],
    [],
  );

  const detailGiftRewardColumns = useMemo<ColumnsType<GiftBoxRewardEntry>>(
    () => [
      {
        title: '类型',
        dataIndex: 'type',
        key: 'type',
        width: 100,
        render: (value: GiftBoxRewardEntry['type']) => {
          if (value === 'gold') {
            return '金币';
          }
          if (value === 'pet') {
            return '宠物';
          }
          return '物品';
        },
      },
      {
        title: '内容',
        key: 'content',
        render: (_value, record) => formatGiftBoxRewardRowSummary(record),
      },
    ],
    [],
  );

  const columns = useMemo<ColumnsType<AdminItemSummary>>(
    () => [
      { title: '物品ID', dataIndex: 'item_id', key: 'item_id', width: 110, fixed: 'left' },
      { title: '编码', dataIndex: 'item_code', key: 'item_code', width: 150 },
      { title: '名称', dataIndex: 'item_name', key: 'item_name', width: 160 },
      { title: '分类', dataIndex: 'item_type', key: 'item_type', width: 120, render: (value: string) => <Tag color="blue">{formatDisplayLabel(ITEM_TYPE_LABELS, value)}</Tag> },
      { title: '堆叠上限', dataIndex: 'max_stack', key: 'max_stack', width: 110 },
      { title: '买价(铜)', dataIndex: 'buy_price_copper', key: 'buy_price_copper', width: 120 },
      { title: '卖价(铜)', dataIndex: 'sell_price_copper', key: 'sell_price_copper', width: 120 },
      { title: '启用', dataIndex: 'is_enabled', key: 'is_enabled', width: 90, render: (value: boolean) => value ? <Tag color="green">启用</Tag> : <Tag>停用</Tag> },
      {
        title: '操作', key: 'actions', width: 100, fixed: 'right', render: (_value, record) => (
          <TableActionDropdown
            actions={[
              { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.item_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.item_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个物品模板吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.item_id),
              },
            ]}
          />
        ),
      },
    ],
    [detail],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="模板列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="item_id" label="物品ID">
              <Input allowClear placeholder="物品ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="item_type" label="分类">
              <Select allowClear placeholder="分类" style={{ width: 120 }} options={buildFilterSelectOptions(ITEM_TYPE_LABELS)} />
            </Form.Item>
            <Form.Item name="keyword" label="关键字">
              <Input allowClear placeholder="编码或名称" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用">
              <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增物品</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="item_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有模板数据" /> }}
          scroll={{ x: 1300 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条模板`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
        {excludeItemType ? (
          <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
            当前页默认排除 `{excludeItemType}`，装备请到“装备”页签维护，避免遗漏扩展字段。
          </Typography.Text>
        ) : null}
      </Card>
      <Drawer title={detail ? `物品详情 · ${detail.item_name}` : '物品详情'} width={720} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? <Typography.Text type="secondary">正在加载物品详情...</Typography.Text> : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="物品ID">{detail.item_id}</Descriptions.Item>
              <Descriptions.Item label="编码">{detail.item_code}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.item_name}</Descriptions.Item>
              <Descriptions.Item label="分类">{formatDisplayLabel(ITEM_TYPE_LABELS, detail.item_type)}</Descriptions.Item>
              {detail.item_sub_type ? (
                <Descriptions.Item label="子分类">{formatDisplayLabel(ITEM_SUB_TYPE_LABELS, detail.item_sub_type)}</Descriptions.Item>
              ) : null}
              <Descriptions.Item label="堆叠上限">{detail.max_stack}</Descriptions.Item>
              <Descriptions.Item label="占格">{detail.occupy_slots}</Descriptions.Item>
              <Descriptions.Item label="买价(铜)">{detail.buy_price_copper}</Descriptions.Item>
              <Descriptions.Item label="卖价(铜)">{detail.sell_price_copper}</Descriptions.Item>
              <Descriptions.Item label="使用效果" span={2}>
                {detail.item_type === 'box'
                  ? '打开获得礼包内容'
                  : detail.item_sub_type === 'equipment_enhance'
                    ? '强化材料（无使用效果）'
                    : formatConsumableEffectSummary(detailUseEffects)}
              </Descriptions.Item>
              {detail.item_sub_type === 'equipment_enhance' ? (
                <Descriptions.Item label="锻造属性" span={2}>
                  {formatEnhanceMaterialConfigSummary(detail.enhance_material_config ?? {
                    success_rate_mode: 'base',
                    success_rate_bonus_pct: 0,
                    success_rate_override_pct: 0,
                    guaranteed_success: false,
                    failure_penalty: 'damage',
                    failure_level_delta: 1,
                    description: '',
                  })}
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="说明" span={2}>
                <RichTextDisplay value={detail.desc} />
              </Descriptions.Item>
            </Descriptions>
            {detail.item_type === 'box' ? (
              <>
                <Divider orientation="left" plain>礼包内容</Divider>
                <Typography.Text type="secondary">{formatGiftRewardSummary(detailGiftRewards)}</Typography.Text>
                <Table
                  size="small"
                  rowKey={(_record, index) => String(index)}
                  columns={detailGiftRewardColumns}
                  dataSource={detailGiftRewards}
                  pagination={false}
                  locale={{ emptyText: '尚未配置礼包内容' }}
                />
              </>
            ) : detail.item_sub_type === 'equipment_enhance' ? (
              <>
                <Divider orientation="left" plain>锻造属性</Divider>
                <Typography.Text type="secondary">
                  {formatEnhanceMaterialConfigSummary(detail.enhance_material_config ?? defaultEnhanceMaterialConfig())}
                </Typography.Text>
                {detail.enhance_material_config?.description ? (
                  <RichTextDisplay value={detail.enhance_material_config.description} />
                ) : null}
              </>
            ) : (
              <>
                <Divider orientation="left" plain>使用效果</Divider>
                <Table
                  size="small"
                  rowKey={(_record, index) => String(index)}
                  columns={detailUseEffectColumns}
                  dataSource={detailUseEffects}
                  pagination={false}
                  locale={{ emptyText: '尚未配置使用效果' }}
                />
              </>
            )}
          </Space>
        )}
      </Drawer>
      <Modal
        title={editingRecord ? `编辑模板 · ${editingRecord.item_name}` : '新增物品模板'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={860}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Form
          form={editorForm}
          layout="vertical"
          onFinish={(values) => void handleSubmit(values)}
          onValuesChange={(changedValues) => {
            if (changedValues.item_type === 'box') {
              editorForm.setFieldsValue({
                usable: true,
                use_effects: [],
                gift_rewards: editorForm.getFieldValue('gift_rewards') ?? [],
                item_sub_type: 'gift_box',
              });
              return;
            }
            if (changedValues.item_type != null && changedValues.item_type !== 'material') {
              editorForm.setFieldsValue({ item_sub_type: '' });
            }
          }}
        >
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item
                label="物品ID"
                name="item_id"
                rules={[{ required: true, message: '系统未生成物品ID，请关闭弹窗后重试' }]}
                extra="新建时自动取当前最大物品ID + 1，创建后不可修改。"
              >
                <InputNumber min={1} disabled controls={false} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                label="物品编码"
                name="item_code"
                rules={[{ required: true, message: '系统未生成物品编码，请关闭弹窗后重试' }]}
                extra="系统按 item_{item_id} 自动生成并锁定。"
              >
                <Input disabled />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="物品名称" name="item_name" rules={[{ required: true, message: '请输入物品名称' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="主分类" name="item_type" rules={[{ required: true, message: '请选择主分类' }]}>
                <Select options={buildSelectOptions(ITEM_TYPE_LABELS)} />
              </Form.Item>
            </Col>
            {isMaterialForm ? (
              <Col xs={24} md={8}>
                <Form.Item
                  label="子分类"
                  name="item_sub_type"
                  extra="选择「强化材料」后，该物品可在客户端装备强化面板作为消耗材料被选用。"
                >
                  <Select options={buildSelectOptions(MATERIAL_ITEM_SUB_TYPE_LABELS)} />
                </Form.Item>
              </Col>
            ) : (
              <Form.Item name="item_sub_type" hidden><Input /></Form.Item>
            )}
            <Form.Item name="rarity" hidden><InputNumber min={1} /></Form.Item>
            <Form.Item name="quality" hidden><InputNumber min={1} /></Form.Item>
            <Form.Item name="icon" hidden><Input /></Form.Item>
            <Form.Item name="use_scope" hidden><Input /></Form.Item>
            <Form.Item name="target_type" hidden><Input /></Form.Item>
            <Col xs={24} md={8}><Form.Item label="堆叠上限" name="max_stack"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="占格" name="occupy_slots"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="买价(铜)" name="buy_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="卖价(铜)" name="sell_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="回收价(铜)" name="recycle_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}>
              <Form.Item
                label="描述"
                name="desc"
                extra="可在下方预览中刷色；{item:物品ID} 占位符会在客户端内联展示物品 icon 与名称。"
              >
                <RichTextEditor rows={4} placeholder="例如：恢复HP +300" />
              </Form.Item>
            </Col>

            {isEnhanceMaterialForm ? (
              <Col span={24}>
                <Divider orientation="left" plain>锻造属性</Divider>
                <Form.Item name="enhance_material_config">
                  <EnhanceMaterialConfigEditor />
                </Form.Item>
                <Typography.Text type="secondary">
                  子分类为「强化材料」时，客户端强化弹窗会按此处配置计算成功率与失败惩罚。
                </Typography.Text>
              </Col>
            ) : isGiftBoxForm ? (
              <Col span={24}>
                <Divider orientation="left" plain>礼包内容</Divider>
                <Form.Item
                  name="gift_rewards"
                  rules={[{
                    validator: async (_rule, value: GiftBoxRewardEntry[] | undefined) => {
                      if ((value ?? []).length > 0) {
                        return;
                      }
                      throw new Error('请至少添加一条礼包奖励');
                    },
                  }]}
                >
                  <GiftBoxRewardEditor />
                </Form.Item>
                <Typography.Text type="secondary">
                  分类为“礼包”时，玩家打开后将按下方列表固定发放奖励；无需再填写效果类型或 JSON。
                </Typography.Text>
              </Col>
            ) : (
              <Col span={24}>
                <Divider orientation="left" plain>使用效果</Divider>
                <Form.Item name="use_effects">
                  <ConsumableEffectEditor />
                </Form.Item>
                <Typography.Text type="secondary">
                  支持配置多条效果；单条旧版行为（如宠物回血、背包扩容）保存时仍兼容现有服务端 effect_type。
                </Typography.Text>
              </Col>
            )}

            <Col xs={12} md={6}><Form.Item label="可使用" name="usable" valuePropName="checked"><Switch disabled={isGiftBoxForm} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可出售" name="can_sell" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可丢弃" name="can_drop" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可存仓" name="can_store" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可交易" name="can_trade" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="自动并堆" name="auto_merge" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultItemValues(itemID: number): ItemEditorFormValues {
  return {
    item_id: itemID,
    item_code: `item_${itemID}`,
    item_name: '新物品模板',
    item_type: 'consumable',
    item_sub_type: '',
    quality: 1,
    rarity: 1,
    icon: '',
    desc: '',
    max_stack: 1,
    occupy_slots: 1,
    auto_merge: true,
    sort_weight: 0,
    usable: false,
    use_scope: '',
    target_type: '',
    required_level: 0,
    required_scene_id: 0,
    bind_type: 'none',
    can_sell: false,
    can_drop: false,
    can_store: true,
    can_trade: false,
    expire_at_rule: '',
    effect_type: '',
    effect_value: 0,
    effect_params_json: '{}',
    buy_price_copper: 0,
    sell_price_copper: 0,
    recycle_price_copper: 0,
    price_type: 'base_coin',
    is_enabled: true,
    use_effects: [],
    gift_rewards: [],
    enhance_material_config: defaultEnhanceMaterialConfig(),
  };
}
