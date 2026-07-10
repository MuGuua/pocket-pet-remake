import {
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { SkillDefinitionEditor, type SkillEditorFormValues } from '../../components/SkillDefinitionEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import {
  createAdminSkillDefinition,
  deleteAdminSkillDefinition,
  fetchAdminSkillDefinitionDetail,
  fetchAdminSkillDefinitions,
  updateAdminSkillDefinition,
} from '../../services/skillDefinition';
import type {
  AdminSkillDetail,
  AdminSkillListFilters,
  AdminSkillSummary,
} from '../../types/skillDefinition';
import { defaultSkillValues, detailToPayload } from '../../types/skillDefinition';
import {
  createDefaultSkillEffectEntries,
  filterSkillEffectEntriesForActivationMode,
  mergePayloadFromSkillEffectEntries,
  skillEffectEntriesFromPayload,
} from '../../types/skillEffectConfig';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  formatControlStatusLabel,
  formatDisplayLabel,
  SKILL_ACTIVATION_MODE_LABELS,
  SKILL_ACTIVATION_MODE_OPTIONS,
  SKILL_CATEGORY_LABELS,
  SKILL_CATEGORY_OPTIONS,
  SKILL_QUALITY_LABELS,
  SKILL_QUALITY_OPTIONS,
  SKILL_TYPE_LABELS,
  SKILL_TYPE_OPTIONS,
  TARGET_TYPE_LABELS,
  WEAPON_DISCIPLINE_LABELS,
  PREFERRED_TARGET_LABELS,
} from '../../utils/displayLabels';

// 系统技能模板页维护战斗结算唯一真源；宠物、玩家与怪物配置只能引用 skill_id。
export function SkillDefinitionPage() {
  const [filterForm] = Form.useForm<AdminSkillListFilters>();
  const [editorForm] = Form.useForm<SkillEditorFormValues>();
  const [filters, setFilters] = useState<AdminSkillListFilters>({});
  const [rows, setRows] = useState<AdminSkillSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminSkillDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminSkillDetail | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadSkills(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadSkills(nextFilters: AdminSkillListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminSkillDefinitions({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统技能模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function loadNextSkillID(): Promise<number> {
    try {
      const result = await fetchAdminSkillDefinitions({
        filters: { order_by: 'skill_id_desc' },
        page: 1,
        pageSize: 1,
      });
      const currentMaxSkillID = result.items[0]?.skill_id ?? 1000;
      return currentMaxSkillID + 1;
    } catch (error) {
      message.error(error instanceof Error ? error.message : '获取下一个技能ID失败');
      return 1001;
    }
  }

  async function handleViewDetail(skillID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminSkillDefinitionDetail(skillID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载技能详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', skillID?: number) {
    setEditorOpen(true);
    editorForm.resetFields();
    if (mode === 'create') {
      setEditingRecord(null);
      const nextSkillID = await loadNextSkillID();
      editorForm.setFieldsValue({
        ...defaultSkillValues(nextSkillID),
        effect_entries: createDefaultSkillEffectEntries(),
      });
      return;
    }
    if (!skillID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminSkillDefinitionDetail(skillID);
      setEditingRecord(result);
      const payload = detailToPayload(result);
      const effectEntries = filterSkillEffectEntriesForActivationMode(
        skillEffectEntriesFromPayload(payload),
        payload.activation_mode,
      );
      editorForm.setFieldsValue({
        ...payload,
        effect_entries: effectEntries,
      });
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载技能编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  function handleCloseEditor() {
    setEditorOpen(false);
    setEditingRecord(null);
    editorForm.resetFields();
  }

  async function handleSubmit(values: SkillEditorFormValues) {
    setSaving(true);
    const fullValues = editorForm.getFieldsValue(true) as SkillEditorFormValues;
    const payload = mergePayloadFromSkillEffectEntries(
      { ...fullValues, ...values },
      fullValues.effect_entries ?? values.effect_entries,
    );
    try {
      if (editingRecord) {
        await updateAdminSkillDefinition(editingRecord.skill_id, payload);
        message.success(`系统技能模板已更新：#${editingRecord.skill_id} ${payload.skill_name}`);
      } else {
        const created = await createAdminSkillDefinition(payload);
        message.success(`系统技能模板已创建：#${created.skill_id} ${created.skill_name}`);
      }
      handleCloseEditor();
      await loadSkills(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存系统技能模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(skillID: number) {
    try {
      await deleteAdminSkillDefinition(skillID);
      message.success('系统技能模板已删除');
      if (detail?.skill_id === skillID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadSkills(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除系统技能模板失败');
    }
  }

  const columns = useMemo<ColumnsType<AdminSkillSummary>>(
    () => [
      { title: '技能ID', dataIndex: 'skill_id', key: 'skill_id', width: 90, fixed: 'left' },
      { title: '编码', dataIndex: 'skill_code', key: 'skill_code', width: 130, ellipsis: true },
      { title: '名称', dataIndex: 'skill_name', key: 'skill_name', width: 140 },
      {
        title: '分类',
        dataIndex: 'skill_category',
        key: 'skill_category',
        width: 100,
        render: (value: string) => <Tag>{formatDisplayLabel(SKILL_CATEGORY_LABELS, value)}</Tag>,
      },
      {
        title: '类型',
        dataIndex: 'skill_type',
        key: 'skill_type',
        width: 80,
        render: (value: string) => formatDisplayLabel(SKILL_TYPE_LABELS, value),
      },
      {
        title: '释放',
        dataIndex: 'activation_mode',
        key: 'activation_mode',
        width: 80,
        render: (value: string) => formatDisplayLabel(SKILL_ACTIVATION_MODE_LABELS, value),
      },
      {
        title: '品质',
        dataIndex: 'skill_quality',
        key: 'skill_quality',
        width: 90,
        render: (value: string) => <Tag>{formatDisplayLabel(SKILL_QUALITY_LABELS, value)}</Tag>,
      },
      {
        title: '目标',
        dataIndex: 'target_type',
        key: 'target_type',
        width: 110,
        render: (value: string) => formatDisplayLabel(TARGET_TYPE_LABELS, value),
      },
      { title: '精力', dataIndex: 'energy_cost', key: 'energy_cost', width: 70 },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        key: 'is_enabled',
        width: 80,
        render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>,
      },
      {
        title: '操作',
        key: 'actions',
        width: 90,
        fixed: 'right',
        render: (_value, record) => (
          <span onClick={(event) => event.stopPropagation()}>
            <TableActionDropdown
              actions={[
                { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.skill_id) },
                { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.skill_id) },
                {
                  key: 'delete',
                  label: '删除',
                  danger: true,
                  disabled: record.is_basic_attack,
                  confirm: { title: '确认删除这个系统技能模板吗？', okText: '确认删除', cancelText: '取消' },
                  onClick: () => void handleDelete(record.skill_id),
                },
              ]}
            />
          </span>
        ),
      },
    ],
    [detail],
  );

  return (
    <Card
      title="系统技能模板"
      extra={(
        <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
          战斗公式唯一真源；宠物、怪物与装备仅引用 skill_id。
        </Typography.Paragraph>
      )}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Form
          form={filterForm}
          layout="inline"
          onFinish={(values) => {
            setPage(1);
            setFilters(values);
          }}
        >
          <Form.Item name="skill_id" label="技能ID">
            <Input allowClear placeholder="技能ID" style={{ width: 100 }} />
          </Form.Item>
          <Form.Item name="name" label="关键字">
            <Input allowClear placeholder="名称或编码" style={{ width: 140 }} />
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Select allowClear placeholder="分类" style={{ width: 120 }} options={SKILL_CATEGORY_OPTIONS} />
          </Form.Item>
          <Form.Item name="skill_type" label="类型">
            <Select allowClear placeholder="类型" style={{ width: 100 }} options={SKILL_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="activation_mode" label="释放">
            <Select allowClear placeholder="释放" style={{ width: 100 }} options={SKILL_ACTIVATION_MODE_OPTIONS} />
          </Form.Item>
          <Form.Item name="skill_quality" label="品质">
            <Select allowClear placeholder="品质" style={{ width: 110 }} options={SKILL_QUALITY_OPTIONS} />
          </Form.Item>
          <Form.Item name="enabled" label="启用">
            <Select
              allowClear
              placeholder="状态"
              style={{ width: 90 }}
              options={[
                { label: '启用', value: 'true' },
                { label: '停用', value: 'false' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                查询
              </Button>
              <Button
                onClick={() => {
                  filterForm.resetFields();
                  setPage(1);
                  setFilters({});
                }}
              >
                重置
              </Button>
              <Button type="primary" onClick={() => void handleOpenEditor('create')}>
                新增技能
              </Button>
            </Space>
          </Form.Item>
        </Form>

        <Table
          columns={columns}
          dataSource={rows}
          rowKey="skill_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有技能模板" /> }}
          scroll={{ x: 1100 }}
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
      </Space>

      <Drawer
        title={detail ? `技能详情 · ${detail.skill_name}` : '技能详情'}
        width={760}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
      >
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载技能详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small" title="基础信息">
              <Descriptions.Item label="技能ID">{detail.skill_id}</Descriptions.Item>
              <Descriptions.Item label="编码">{detail.skill_code || '-'}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.skill_name}</Descriptions.Item>
              <Descriptions.Item label="分类 / 类型">
                {formatDisplayLabel(SKILL_CATEGORY_LABELS, detail.skill_category)} / {formatDisplayLabel(SKILL_TYPE_LABELS, detail.skill_type)}
              </Descriptions.Item>
              <Descriptions.Item label="释放方式">{formatDisplayLabel(SKILL_ACTIVATION_MODE_LABELS, detail.activation_mode)}</Descriptions.Item>
              <Descriptions.Item label="技能品质">{formatDisplayLabel(SKILL_QUALITY_LABELS, detail.skill_quality)}</Descriptions.Item>
              <Descriptions.Item label="普攻模板">{detail.is_basic_attack ? '是' : '否'}</Descriptions.Item>
              {detail.skill_category === 'weapon' ? (
                <>
                  <Descriptions.Item label="武器流派">{formatDisplayLabel(WEAPON_DISCIPLINE_LABELS, detail.weapon_discipline ?? '')}</Descriptions.Item>
                  <Descriptions.Item label="学会所需经验">{detail.learn_exp_required ?? 0}</Descriptions.Item>
                  <Descriptions.Item label="每次使用经验">{detail.learn_exp_per_use ?? 1}</Descriptions.Item>
                </>
              ) : null}
              <Descriptions.Item label="获取方式">{detail.acquire_method || '-'}</Descriptions.Item>
              <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>
                <RichTextDisplay value={detail.description} />
              </Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="目标规则">
              <Descriptions.Item label="目标类型">{formatDisplayLabel(TARGET_TYPE_LABELS, detail.target_rule.target_type)}</Descriptions.Item>
              <Descriptions.Item label="目标数量">{detail.target_rule.target_count}</Descriptions.Item>
              <Descriptions.Item label="优先目标">{formatDisplayLabel(PREFERRED_TARGET_LABELS, detail.target_rule.preferred_target_hp)}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="伤害 / 治疗公式">
              <Descriptions.Item label="技能倍数">{detail.formula.skill_mult || '（回退 attack_pct/100）'}</Descriptions.Item>
              <Descriptions.Item label="攻击系数">{detail.formula.attack_pct}</Descriptions.Item>
              <Descriptions.Item label="法力系数">{detail.formula.mana_pct}</Descriptions.Item>
              <Descriptions.Item label="速度系数">{detail.formula.speed_pct}</Descriptions.Item>
              <Descriptions.Item label="技能命中加成">{detail.formula.skill_hit_bonus}</Descriptions.Item>
              <Descriptions.Item label="精力消耗">{detail.formula.energy_cost}</Descriptions.Item>
              <Descriptions.Item label="治疗系数">{detail.formula.heal_pct}</Descriptions.Item>
              <Descriptions.Item label="固定伤害">{detail.formula.fixed_damage}</Descriptions.Item>
              <Descriptions.Item label="允许暴击">{detail.formula.allow_crit ? '是' : '否'}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="控制效果">
              <Descriptions.Item label="封印">{detail.status_effects.seal_chance_pct}% / 威力 {detail.status_effects.seal_power}</Descriptions.Item>
              <Descriptions.Item label="通用控制">{detail.status_effects.control_chance_pct}% / 威力 {detail.status_effects.control_power}</Descriptions.Item>
              <Descriptions.Item label="控制状态">{formatControlStatusLabel(detail.status_effects.control_status_id)}</Descriptions.Item>
              <Descriptions.Item label="控制回合">{detail.status_effects.control_rounds}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="战斗表现">
              <Descriptions.Item label="动画键">{detail.presentation.animation_key}</Descriptions.Item>
              <Descriptions.Item label="投射物">{detail.presentation.projectile ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="施法色">{detail.presentation.cast_color}</Descriptions.Item>
              <Descriptions.Item label="命中色">{detail.presentation.impact_color}</Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑系统技能 · ${editingRecord.skill_name}` : '新增系统技能模板'}
        open={editorOpen}
        onCancel={handleCloseEditor}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={820}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <SkillDefinitionEditor form={editorForm} editingRecord={editingRecord} />
        </Form>
      </Modal>
    </Card>
  );
}
