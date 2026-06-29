import { Col, Divider, Form, Input, InputNumber, Row, Select, Switch } from 'antd';
import type { FormInstance } from 'antd/es/form';
import { RichTextEditor } from './RichTextEditor';
import { SkillEffectConfigEditor } from './SkillEffectConfigEditor';
import type { AdminSkillDetail, AdminUpsertSkillPayload } from '../types/skillDefinition';
import type { SkillEffectConfigEntry } from '../types/skillEffectConfig';
import {
  PREFERRED_TARGET_OPTIONS,
  SKILL_CATEGORY_OPTIONS,
  SKILL_TYPE_OPTIONS,
  TARGET_TYPE_OPTIONS,
  WEAPON_DISCIPLINE_OPTIONS,
} from '../utils/displayLabels';

export interface SkillEditorFormValues extends AdminUpsertSkillPayload {
  effect_entries: SkillEffectConfigEntry[];
}

interface SkillDefinitionEditorProps {
  form: FormInstance<SkillEditorFormValues>;
  editingRecord: AdminSkillDetail | null;
}

// 系统技能模板表单：基础/目标字段平铺，公式与状态效果走列表新增编辑器。
export function SkillDefinitionEditor({ form, editingRecord }: SkillDefinitionEditorProps) {
  const skillCategory = Form.useWatch('skill_category', form);
  const isWeaponSkillCategory = skillCategory === 'weapon';

  return (
    <>
      <Divider plain>基础信息</Divider>
      <Row gutter={16}>
        <Col xs={24} md={8}>
          <Form.Item
            label="技能ID"
            name="skill_id"
            rules={[{ required: true, message: '系统未生成技能ID，请关闭弹窗后重试' }]}
            extra="新建时自动分配，创建后不可修改。"
          >
            <InputNumber min={1} disabled style={{ width: '100%' }} controls={false} />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item
            label="技能编码"
            name="skill_code"
            rules={[{ required: true, message: '系统未生成编码，请关闭弹窗后重试' }]}
            extra="系统按 skill_{id} 自动生成。"
          >
            <Input disabled />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="技能名称" name="skill_name" rules={[{ required: true, message: '请输入技能名称' }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="分类" name="skill_category" rules={[{ required: true, message: '请选择分类' }]}>
            <Select
              options={SKILL_CATEGORY_OPTIONS}
              onChange={(value: string) => {
                if (value !== 'weapon') {
                  return;
                }
                const discipline = form.getFieldValue('weapon_discipline');
                const learnRequired = form.getFieldValue('learn_exp_required');
                form.setFieldsValue({
                  weapon_discipline: discipline || 'sword',
                  learn_exp_required: learnRequired && learnRequired > 0 ? learnRequired : 100,
                  learn_exp_per_use: form.getFieldValue('learn_exp_per_use') || 1,
                });
              }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="类型" name="skill_type" rules={[{ required: true, message: '请选择类型' }]}>
            <Select options={SKILL_TYPE_OPTIONS} />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item label="排序权重" name="sort_weight">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        {isWeaponSkillCategory ? (
          <>
            <Col xs={24} md={8}>
              <Form.Item label="武器流派" name="weapon_discipline" rules={[{ required: true, message: '请选择武器流派' }]}>
                <Select options={WEAPON_DISCIPLINE_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="学会所需经验" name="learn_exp_required" rules={[{ required: true, message: '请输入学会所需经验' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="每次使用经验" name="learn_exp_per_use">
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </>
        ) : null}
        <Col span={24}>
          <Form.Item label="描述" name="description" extra="支持 BBCode 富文本，客户端技能详情会原样渲染。">
            <RichTextEditor rows={4} placeholder="例如：对单个敌人造成物理伤害，伤害 [color=green]+120%[/color]" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item label="获取方式" name="acquire_method">
            <Input placeholder="运营配置" />
          </Form.Item>
        </Col>
        <Col xs={12} md={6}>
          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Col>
        {!editingRecord ? (
          <Col xs={12} md={6}>
            <Form.Item label="普攻模板" name="is_basic_attack" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
        ) : null}
      </Row>

      <Divider plain>目标与消耗</Divider>
      <Row gutter={[16, 8]}>
        <Col xs={12} sm={8} md={6} lg={5}>
          <Form.Item label="目标类型" name="target_type" rules={[{ required: true, message: '请选择目标类型' }]}>
            <Select options={TARGET_TYPE_OPTIONS} style={{ width: 152 }} />
          </Form.Item>
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <Form.Item label="目标数量" name="target_count">
            <InputNumber min={0} style={{ width: 96 }} />
          </Form.Item>
        </Col>
        <Col xs={12} sm={8} md={6} lg={5}>
          <Form.Item label="优先目标" name="preferred_target_hp">
            <Select allowClear options={PREFERRED_TARGET_OPTIONS} style={{ width: 152 }} />
          </Form.Item>
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <Form.Item label="精力消耗" name="energy_cost">
            <InputNumber min={0} style={{ width: 96 }} />
          </Form.Item>
        </Col>
      </Row>

      <Divider plain>伤害 / 治疗 / 状态 / 表现</Divider>
      <Form.Item
        name="effect_entries"
        rules={[
          {
            validator: async (_, entries: SkillEffectConfigEntry[] | undefined) => {
              if (!entries || entries.length === 0) {
                throw new Error('至少添加一条技能效果配置');
              }
            },
          },
        ]}
      >
        <SkillEffectConfigEditor />
      </Form.Item>
    </>
  );
}
