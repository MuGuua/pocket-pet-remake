import { Button, Card, Form, Input, Space, Spin, Table, Tag, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { fetchAdminNPCDialogues } from '../../services/npc';
import type { AdminNPCDialogueFilters, AdminNPCDialogueSummary } from '../../types/npc';
import { NPCDialogueConfigDrawer } from './NPCDialogueConfigDrawer';

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '启用', value: '1' },
  { label: '停用', value: '0' },
];

// NPC 剧情配置列表页：按 entity_id + entry_id 检索全部结构化剧情聚合数据。
export function NPCDialogueListPage() {
  const [filterForm] = Form.useForm<AdminNPCDialogueFilters>();
  const [filters, setFilters] = useState<AdminNPCDialogueFilters>({ status: '1' });
  const [rows, setRows] = useState<AdminNPCDialogueSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerEntityId, setDrawerEntityId] = useState<number | null>(null);
  const [drawerEntryId, setDrawerEntryId] = useState('');
  const [drawerEntryTitle, setDrawerEntryTitle] = useState('');

  useEffect(() => {
    filterForm.setFieldsValue({ status: '1' });
  }, [filterForm]);

  useEffect(() => {
    void loadRows(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadRows(nextFilters: AdminNPCDialogueFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCDialogues({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 剧情配置失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  const columns: ColumnsType<AdminNPCDialogueSummary> = useMemo(
    () => [
      { title: 'NPC 实体 ID', dataIndex: 'entity_id', width: 120 },
      { title: '菜单项 ID', dataIndex: 'entry_id', width: 180 },
      { title: '剧情编码', dataIndex: 'dialogue_code', width: 180 },
      { title: '标题', dataIndex: 'title' },
      { title: '起始节点', dataIndex: 'start_node_id', width: 140 },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (value: number) => (value === 1 ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
      },
      { title: '更新时间', dataIndex: 'updated_at', width: 180 },
      {
        title: '操作',
        key: 'actions',
        width: 120,
        render: (_, record) => (
          <Button
            type="link"
            onClick={() => {
              setDrawerEntityId(record.entity_id);
              setDrawerEntryId(record.entry_id);
              setDrawerEntryTitle(record.title);
              setDrawerOpen(true);
            }}
          >
            编辑剧情
          </Button>
        ),
      },
    ],
    [],
  );

  return (
    <>
      <Card title="NPC 剧情配置" style={{ marginBottom: 16 }}>
        <Form
          form={filterForm}
          layout="inline"
          onFinish={(values) => {
            setPage(1);
            setFilters(values);
          }}
        >
          <Form.Item name="entity_id" label="NPC 实体 ID">
            <Input placeholder="例如 93001" allowClear />
          </Form.Item>
          <Form.Item name="entry_id" label="菜单项 ID">
            <Input placeholder="例如 dialog_market_intro" allowClear />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Input placeholder="1" allowClear list="npc-dialogue-status-options" />
          </Form.Item>
          <datalist id="npc-dialogue-status-options">
            {statusOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </datalist>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                查询
              </Button>
              <Button
                onClick={() => {
                  filterForm.resetFields();
                  filterForm.setFieldsValue({ status: '1' });
                  setPage(1);
                  setFilters({ status: '1' });
                }}
              >
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Spin spinning={loading}>
        <Table
          rowKey={(record) => `${record.entity_id}:${record.entry_id}`}
          columns={columns}
          dataSource={rows}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Spin>

      <NPCDialogueConfigDrawer
        open={drawerOpen}
        entityId={drawerEntityId}
        entryId={drawerEntryId}
        entryTitle={drawerEntryTitle}
        onClose={() => {
          setDrawerOpen(false);
          void loadRows(filters, page, pageSize);
        }}
      />
    </>
  );
}
