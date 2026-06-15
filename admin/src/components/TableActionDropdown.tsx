import { DownOutlined } from '@ant-design/icons';
import { Button, Dropdown, Modal } from 'antd';
import type { MenuProps } from 'antd';

export interface TableActionItem {
  key: string;
  label: string;
  onClick: () => void | Promise<void>;
  danger?: boolean;
  disabled?: boolean;
  confirm?: {
    title: string;
    description?: string;
    okText?: string;
    cancelText?: string;
  };
}

interface TableActionDropdownProps {
  actions: TableActionItem[];
  buttonText?: string;
  buttonType?: 'link' | 'default' | 'primary' | 'text';
  loading?: boolean;
  disabled?: boolean;
}

// 表格行与详情抽屉的统一操作入口：把查看/编辑/删除等按钮收敛到一个下拉菜单。
export function TableActionDropdown({
  actions,
  buttonText = '操作',
  buttonType = 'link',
  loading = false,
  disabled = false,
}: TableActionDropdownProps) {
  const visibleActions = actions.filter((action) => !action.disabled);
  if (visibleActions.length === 0) {
    return null;
  }

  const runAction = (action: TableActionItem) => {
    if (action.confirm) {
      Modal.confirm({
        title: action.confirm.title,
        content: action.confirm.description,
        okText: action.confirm.okText ?? '确认',
        cancelText: action.confirm.cancelText ?? '取消',
        okButtonProps: action.danger ? { danger: true } : undefined,
        onOk: () => {
          void action.onClick();
        },
      });
      return;
    }
    void action.onClick();
  };

  const items: MenuProps['items'] = visibleActions.map((action) => ({
    key: action.key,
    label: action.label,
    danger: action.danger,
    onClick: () => runAction(action),
  }));

  return (
    <Dropdown menu={{ items }} trigger={['click']} disabled={disabled}>
      <Button type={buttonType} loading={loading}>
        {buttonText}
        <DownOutlined />
      </Button>
    </Dropdown>
  );
}
