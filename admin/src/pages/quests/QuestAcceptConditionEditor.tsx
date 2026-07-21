import { Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Typography } from 'antd';
import type { AdminQuestAcceptCondition } from '../../types/quest';

interface SelectOption {
  label: string;
  value: number;
}

interface QuestAcceptConditionEditorProps {
  questOptions: SelectOption[];
  sceneOptions: SelectOption[];
  itemOptions: SelectOption[];
  petOptions: SelectOption[];
}

const conditionTypeOptions = [
  { label: '指定任务已完成', value: 'quest_completed' },
  { label: '人物等级', value: 'player_level' },
  { label: '人物最终属性', value: 'player_stat' },
  { label: '当前所在地图', value: 'scene' },
  { label: '持有物品数量', value: 'item_count' },
  { label: '宠物等级', value: 'pet_level' },
  { label: '剧情标记', value: 'story_flag' },
  { label: '服务端时间段', value: 'time_window' },
];

const operatorOptions = [
  { label: '大于等于', value: 'gte' },
  { label: '等于', value: 'eq' },
  { label: '小于等于', value: 'lte' },
];

const statOptions = [
  { label: '生命上限', value: 'hp_max' },
  { label: '攻击', value: 'atk' },
  { label: '防御', value: 'def' },
  { label: '速度', value: 'spd' },
  { label: '法力', value: 'mana' },
];

/** 用结构化表单维护服务端权威任务开启条件，列表内全部条件按 AND 关系判断。 */
export function QuestAcceptConditionEditor({ questOptions, sceneOptions, itemOptions, petOptions }: QuestAcceptConditionEditorProps) {
  return (
    <Form.List name="accept_conditions">
      {(fields, { add, remove }) => (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Text type="secondary">以下条件必须全部满足；留空表示任务不增加额外开启限制。</Typography.Text>
          {fields.map((field, index) => (
            <Card
              key={field.key}
              size="small"
              title={`开启条件 ${index + 1}`}
              extra={<Button danger type="link" onClick={() => remove(field.name)}>删除</Button>}
            >
              <Row gutter={12}>
                <Col xs={24} md={8}>
                  <Form.Item {...field} name={[field.name, 'type']} label="条件类型" rules={[{ required: true, message: '请选择条件类型' }]}>
                    <Select options={conditionTypeOptions} placeholder="请选择" />
                  </Form.Item>
                </Col>
                <Form.Item noStyle shouldUpdate>
                  {({ getFieldValue }) => renderConditionFields(
                    getFieldValue(['accept_conditions', field.name, 'type']) as AdminQuestAcceptCondition['type'] | undefined,
                    field.name,
                    { questOptions, sceneOptions, itemOptions, petOptions },
                  )}
                </Form.Item>
              </Row>
            </Card>
          ))}
          <Button type="dashed" onClick={() => add({ type: 'quest_completed', operator: 'gte', value: 1 })}>
            添加任务开启条件
          </Button>
        </Space>
      )}
    </Form.List>
  );
}

/** 按条件类型仅展示有效字段，避免运营同时填写互相无关的配置。 */
function renderConditionFields(
  type: AdminQuestAcceptCondition['type'] | undefined,
  fieldName: number,
  options: Pick<QuestAcceptConditionEditorProps, 'questOptions' | 'sceneOptions' | 'itemOptions' | 'petOptions'>,
) {
  if (type === 'quest_completed') {
    return <ConditionSelect fieldName={fieldName} name="quest_id" label="必须完成的任务" options={options.questOptions} />;
  }
  if (type === 'scene') {
    return <ConditionSelect fieldName={fieldName} name="scene_id" label="必须所在地图" options={options.sceneOptions} />;
  }
  if (type === 'story_flag') {
    return <Col xs={24} md={16}><Form.Item name={[fieldName, 'flag_key']} label="剧情标记 Key" rules={[{ required: true, message: '请输入剧情标记' }]}><Input placeholder="例如：taozi_npc_unlocked" /></Form.Item></Col>;
  }
  if (type === 'time_window') {
    return <>
      <Col xs={24} md={8}><Form.Item name={[fieldName, 'start_at']} label="开始时间" rules={[{ required: true, message: '请选择开始时间' }]}><Input type="datetime-local" /></Form.Item></Col>
      <Col xs={24} md={8}><Form.Item name={[fieldName, 'end_at']} label="结束时间" rules={[{ required: true, message: '请选择结束时间' }]}><Input type="datetime-local" /></Form.Item></Col>
    </>;
  }

  const prefix = type === 'player_stat'
    ? <Col xs={24} md={8}><Form.Item name={[fieldName, 'stat_key']} label="人物属性" rules={[{ required: true, message: '请选择人物属性' }]}><Select options={statOptions} /></Form.Item></Col>
    : type === 'item_count'
      ? <ConditionSelect fieldName={fieldName} name="item_id" label="物品" options={options.itemOptions} />
      : type === 'pet_level'
        ? <ConditionSelect fieldName={fieldName} name="pet_id" label="宠物（留空表示任意宠物）" options={options.petOptions} required={false} allowClear />
        : null;
  if (!['player_level', 'player_stat', 'item_count', 'pet_level'].includes(type ?? '')) {
    return null;
  }
  return <>
    {prefix}
    <Col xs={12} md={4}><Form.Item name={[fieldName, 'operator']} label="比较方式" rules={[{ required: true }]}><Select options={operatorOptions} /></Form.Item></Col>
    <Col xs={12} md={4}><Form.Item name={[fieldName, 'value']} label={type === 'item_count' ? '数量' : '目标值'} rules={[{ required: true, message: '请输入目标值' }]}><InputNumber min={type === 'item_count' || type === 'pet_level' || type === 'player_level' ? 1 : 0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
  </>;
}

/** 关联数据选择器统一开启搜索，并由调用页传入数据库加载的真实选项。 */
function ConditionSelect({ fieldName, name, label, options, required = true, allowClear = false }: { fieldName: number; name: string; label: string; options: SelectOption[]; required?: boolean; allowClear?: boolean }) {
  return <Col xs={24} md={8}><Form.Item name={[fieldName, name]} label={label} rules={required ? [{ required: true, message: `请选择${label}` }] : []}><Select showSearch optionFilterProp="label" options={options} allowClear={allowClear} /></Form.Item></Col>;
}
