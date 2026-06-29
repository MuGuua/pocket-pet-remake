import { Button, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import type {
  ConsumableEffectCategory,
  ConsumableEffectEntry,
  ConsumableEffectOperation,
} from '../types/consumableEffect';
import {
  buildConsumableEffectFieldOptions,
  CONSUMABLE_EFFECT_CATEGORY_OPTIONS,
  CONSUMABLE_EFFECT_FIELD_CATALOG,
  CONSUMABLE_EFFECT_OPERATION_OPTIONS,
  formatConsumableEffectCategoryLabel,
  formatConsumableEffectEntryLabel,
  formatConsumableEffectOperationLabel,
  resolveConsumableEffectField,
} from '../types/consumableEffect';

interface ConsumableEffectEditorProps {
  value?: ConsumableEffectEntry[];
  onChange?: (nextValue: ConsumableEffectEntry[]) => void;
}

interface EffectEditorFormValues {
  category: ConsumableEffectCategory;
  field_key: string;
  operation: ConsumableEffectOperation;
  number_value?: number;
  boolean_value?: boolean;
}

/** 消耗品使用效果列表编辑器：先选大类，再选可编辑字段与操作数值。 */
export function ConsumableEffectEditor({ value = [], onChange }: ConsumableEffectEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editorForm] = Form.useForm<EffectEditorFormValues>();
  const watchedCategory = Form.useWatch('category', editorForm);
  const watchedFieldKey = Form.useWatch('field_key', editorForm);
  const watchedOperation = Form.useWatch('operation', editorForm);
  const selectedField = resolveConsumableEffectField(watchedCategory ?? 'player', watchedFieldKey ?? '');
  const fieldOptions = useMemo(
    () => buildConsumableEffectFieldOptions(watchedCategory ?? 'player'),
    [watchedCategory],
  );
  const operationOptions = useMemo(() => {
    const allowedOperations = selectedField?.operations ?? ['add', 'subtract', 'set'];
    return CONSUMABLE_EFFECT_OPERATION_OPTIONS.filter((option) => allowedOperations.includes(option.value));
  }, [selectedField]);
  const numberValueMin = watchedOperation === 'set' ? 0 : 1;

  const columns = useMemo<ColumnsType<ConsumableEffectEntry>>(
    () => [
      {
        title: '大类',
        dataIndex: 'category',
        key: 'category',
        width: 90,
        render: (category: ConsumableEffectCategory) => (
          <Tag color="blue">{formatConsumableEffectCategoryLabel(category)}</Tag>
        ),
      },
      {
        title: '效果',
        key: 'effect',
        render: (_value, record) => formatConsumableEffectEntryLabel(record),
      },
      {
        title: '操作',
        key: 'actions',
        width: 140,
        render: (_value, _record, index) => (
          <Space size={8}>
            <Button size="small" onClick={() => openEditor(index)}>编辑</Button>
            <Button size="small" danger onClick={() => removeEntry(index)}>删除</Button>
          </Space>
        ),
      },
    ],
    [value],
  );

  function emitChange(nextValue: ConsumableEffectEntry[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      editorForm.setFieldsValue({
        category: 'player',
        field_key: CONSUMABLE_EFFECT_FIELD_CATALOG.player[0]?.key ?? 'level',
        operation: 'add',
        number_value: 1,
        boolean_value: true,
      });
    } else {
      const current = value[index];
      const field = resolveConsumableEffectField(current.category, current.field_key);
      editorForm.setFieldsValue({
        category: current.category,
        field_key: current.field_key,
        operation: current.operation,
        number_value: typeof current.value === 'number' ? current.value : 1,
        boolean_value: typeof current.value === 'boolean' ? current.value : true,
      });
      if (field?.valueType === 'boolean') {
        editorForm.setFieldValue('operation', 'set');
      }
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleCategoryChange(nextCategory: ConsumableEffectCategory) {
    const nextFields = buildConsumableEffectFieldOptions(nextCategory);
    const firstFieldKey = nextCategory === 'other'
      ? ''
      : nextFields[0]?.options[0]?.value ?? '';
    const nextField = resolveConsumableEffectField(nextCategory, firstFieldKey);
    editorForm.setFieldsValue({
      category: nextCategory,
      field_key: firstFieldKey,
      operation: nextField?.operations?.[0] ?? 'add',
      number_value: 1,
      boolean_value: true,
    });
  }

  function handleFieldChange(nextFieldKey: string) {
    const category = editorForm.getFieldValue('category') as ConsumableEffectCategory;
    const field = resolveConsumableEffectField(category, nextFieldKey);
    if (field?.valueType === 'boolean') {
      editorForm.setFieldsValue({ operation: 'set', boolean_value: true });
      return;
    }
    editorForm.setFieldsValue({
      operation: field?.operations?.[0] ?? 'add',
      number_value: 1,
    });
  }

  function handleSubmitEditor(formValues: EffectEditorFormValues) {
    const field = resolveConsumableEffectField(formValues.category, formValues.field_key);
    if (!field) {
      return;
    }
    const nextEntry: ConsumableEffectEntry = field.valueType === 'boolean'
      ? {
          category: formValues.category,
          field_key: formValues.field_key.trim(),
          operation: 'set',
          value: Boolean(formValues.boolean_value),
        }
      : {
          category: formValues.category,
          field_key: formValues.field_key.trim(),
          operation: formValues.operation,
          value: Math.trunc(Number(formValues.number_value ?? 0)),
        };
    const nextValue = [...value];
    if (editingIndex === null) {
      nextValue.push(nextEntry);
    } else {
      nextValue[editingIndex] = nextEntry;
    }
    emitChange(nextValue);
    setEditorOpen(false);
    setEditingIndex(null);
    editorForm.resetFields();
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Text type="secondary">
          配置消耗品使用后的效果列表；先选大类，再选对应可编辑字段与数值。
        </Typography.Text>
        <Button type="primary" onClick={() => openEditor(null)}>新增效果</Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={value}
        pagination={false}
        locale={{ emptyText: '尚未配置使用效果，请点击右上角“新增效果”。' }}
      />
      <Modal
        title={editingIndex === null ? '新增使用效果' : '编辑使用效果'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingIndex(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        destroyOnClose
        width={560}
      >
        <Form
          form={editorForm}
          layout="vertical"
          onFinish={(formValues) => handleSubmitEditor(formValues)}
        >
          <Form.Item label="效果大类" name="category" rules={[{ required: true, message: '请选择效果大类' }]}>
            <Select
              options={CONSUMABLE_EFFECT_CATEGORY_OPTIONS.map((option) => ({
                value: option.value,
                label: option.label,
              }))}
              onChange={(nextCategory: ConsumableEffectCategory) => handleCategoryChange(nextCategory)}
            />
          </Form.Item>
          {watchedCategory === 'other' ? (
            <Form.Item
              label="自定义字段 Key"
              name="field_key"
              rules={[{ required: true, message: '请输入自定义字段 Key' }]}
              extra="预留扩展字段，需与服务端约定后才会生效。"
            >
              <Input placeholder="例如 custom_buff_id" />
            </Form.Item>
          ) : (
            <Form.Item label="作用字段" name="field_key" rules={[{ required: true, message: '请选择作用字段' }]}>
              <Select
                options={fieldOptions}
                onChange={(nextFieldKey: string) => handleFieldChange(nextFieldKey)}
              />
            </Form.Item>
          )}
          {selectedField?.valueType === 'boolean' ? (
            <Form.Item label="开关" name="boolean_value" valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="关闭" />
            </Form.Item>
          ) : (
            <>
              <Form.Item label="操作方式" name="operation" rules={[{ required: true, message: '请选择操作方式' }]}>
                <Select
                  options={operationOptions.map((option) => ({
                    value: option.value,
                    label: formatConsumableEffectOperationLabel(option.value),
                  }))}
                />
              </Form.Item>
              <Form.Item label="数值" name="number_value" rules={[{ required: true, message: '请输入效果数值' }]}>
                <InputNumber min={numberValueMin} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>
    </Space>
  );
}
