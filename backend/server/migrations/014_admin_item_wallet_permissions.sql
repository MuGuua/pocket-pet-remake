-- 014_admin_item_wallet_permissions.sql
--
-- 为新的物品模板与钱包后台页面补权限定义，并把权限挂到 super_admin。
-- 这样执行完 migration 后，现有超级管理员可以直接访问新页面与新接口。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('items:view', '查看物品模板', 'items', 'view', '查看物品模板列表与详情', 1),
  ('items:edit', '编辑物品模板', 'items', 'edit', '新增、编辑、删除物品模板', 1),
  ('wallet:view', '查看钱包', 'wallet', 'view', '查看玩家钱包列表与详情', 1),
  ('wallet:edit', '编辑钱包', 'wallet', 'edit', '调整玩家钱包余额', 1)
ON CONFLICT (permission_key) DO UPDATE SET
  permission_name = EXCLUDED.permission_name,
  module_key = EXCLUDED.module_key,
  action_key = EXCLUDED.action_key,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO admin_role_permission (admin_role_id, admin_permission_id)
SELECT r.id, p.id
FROM admin_role r
JOIN admin_permission p ON p.permission_key IN ('items:view', 'items:edit', 'wallet:view', 'wallet:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
