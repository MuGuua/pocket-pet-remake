import { Alert, Descriptions, Form, InputNumber, Modal, Select, Spin, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { formatPetQualityLabel, getPetQualityTagColor } from '../constants/petQuality';
import { grantAdminPetFromTemplate } from '../services/pet';
import { fetchAllEnabledAdminPetDefinitions } from '../services/petDefinition';
import type { AdminPetDefinitionSummary } from '../types/petDefinition';

interface GrantPetFormValues {
  player_id?: number;
  pet_id: number;
}

interface GrantPetFromTemplateModalProps {
  /** 是否显示弹窗。 */
  open: boolean;
  /** 固定所属玩家 ID；传入后不再展示玩家 ID 输入框。 */
  fixedPlayerId?: number;
  /** 固定玩家名称，用于标题展示。 */
  fixedPlayerName?: string;
  /** 关闭弹窗。 */
  onCancel: () => void;
  /** 发放成功后回调。 */
  onSuccess: () => void;
}

/** 运营发放宠物：从启用的系统宠物模板中选择，服务端按模板生成初始实例。 */
export function GrantPetFromTemplateModal({
  open,
  fixedPlayerId,
  fixedPlayerName,
  onCancel,
  onSuccess,
}: GrantPetFromTemplateModalProps) {
  const [form] = Form.useForm<GrantPetFormValues>();
  const [loadingOptions, setLoadingOptions] = useState(false);
  const [saving, setSaving] = useState(false);
  const [templates, setTemplates] = useState<AdminPetDefinitionSummary[]>([]);
  const selectedPetID = Form.useWatch('pet_id', form);

  useEffect(() => {
    if (!open) {
      return;
    }
    form.setFieldsValue({
      player_id: fixedPlayerId,
      pet_id: undefined as unknown as number,
    });
    void loadTemplates();
  }, [open, fixedPlayerId, form]);

  async function loadTemplates() {
    setLoadingOptions(true);
    try {
      setTemplates(await fetchAllEnabledAdminPetDefinitions());
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统宠物模板失败');
      setTemplates([]);
    } finally {
      setLoadingOptions(false);
    }
  }

  const templateOptions = useMemo(
    () => templates.map((item) => ({
      value: item.pet_id,
      label: `#${item.pet_id} ${item.pet_name} · ${formatPetQualityLabel(item.quality)} · Lv.${item.level}`,
    })),
    [templates],
  );

  const selectedTemplate = useMemo(
    () => templates.find((item) => item.pet_id === selectedPetID) ?? null,
    [selectedPetID, templates],
  );

  async function handleSubmit(values: GrantPetFormValues) {
    const playerID = fixedPlayerId ?? Number(values.player_id ?? 0);
    const petID = Number(values.pet_id ?? 0);
    if (!playerID || !petID) {
      message.error('请选择玩家与系统宠物模板');
      return;
    }
    setSaving(true);
    try {
      await grantAdminPetFromTemplate({ player_id: playerID, pet_id: petID });
      message.success('宠物发放成功');
      onSuccess();
      onCancel();
      form.resetFields();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '发放宠物失败');
    } finally {
      setSaving(false);
    }
  }

  const title = fixedPlayerName
    ? `新增宠物 · ${fixedPlayerName}`
    : '按系统宠物模板发放';

  return (
    <Modal
      title={title}
      open={open}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      onOk={() => form.submit()}
      confirmLoading={saving}
      destroyOnClose
      width={560}
      okText="确认发放"
      cancelText="取消"
    >
      <Form form={form} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
        {fixedPlayerId ? (
          <Form.Item label="所属玩家">
            <Typography.Text>{fixedPlayerName ?? '-'}（ID: {fixedPlayerId}）</Typography.Text>
          </Form.Item>
        ) : (
          <Form.Item
            label="所属玩家ID"
            name="player_id"
            rules={[{ required: true, message: '请输入玩家ID' }]}
          >
            <InputNumber min={1} style={{ width: '100%' }} placeholder="输入要发放的玩家 ID" />
          </Form.Item>
        )}
        <Form.Item
          label="系统宠物模板"
          name="pet_id"
          rules={[{ required: true, message: '请选择系统宠物' }]}
          extra="仅展示已启用的系统宠物；等级、资质、技能由服务端按模板初始值发放。"
        >
          <Select
            showSearch
            allowClear
            placeholder={loadingOptions ? '正在加载模板…' : '搜索宠物名称或 ID'}
            loading={loadingOptions}
            options={templateOptions}
            optionFilterProp="label"
            notFoundContent={loadingOptions ? <Spin size="small" /> : '没有可用的系统宠物模板'}
          />
        </Form.Item>
        {selectedTemplate ? (
          <Descriptions bordered size="small" column={2} title="模板预览">
            <Descriptions.Item label="宠物ID">{selectedTemplate.pet_id}</Descriptions.Item>
            <Descriptions.Item label="名称">{selectedTemplate.pet_name}</Descriptions.Item>
            <Descriptions.Item label="品质">
              <Tag color={getPetQualityTagColor(selectedTemplate.quality)}>
                {formatPetQualityLabel(selectedTemplate.quality)}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="带出等级">{selectedTemplate.level}</Descriptions.Item>
          </Descriptions>
        ) : (
          <Alert
            type="info"
            showIcon
            message="发放说明"
            description="无需手动填写战斗数值；确认后会创建 1 只按模板初始化的宠物实例，后续可在「编辑」中微调。"
          />
        )}
      </Form>
    </Modal>
  );
}
