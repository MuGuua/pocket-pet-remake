-- 永久删除禁用账号属于不可恢复操作，使用独立权限与普通玩家编辑能力隔离。
INSERT INTO admin_permission (
  permission_key,
  permission_name,
  module_key,
  action_key,
  description,
  status
) VALUES (
  'players:purge',
  '永久删除禁用账号',
  'players',
  'purge',
  '永久删除已禁用账号及其全部玩家数据',
  1
)
ON CONFLICT (permission_key) DO UPDATE SET
  permission_name = EXCLUDED.permission_name,
  module_key = EXCLUDED.module_key,
  action_key = EXCLUDED.action_key,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- 当前超级管理员自动获得永久删除权限；其他角色需要管理员显式授权。
INSERT INTO admin_role_permission (admin_role_id, admin_permission_id)
SELECT role.id, permission.id
FROM admin_role AS role
JOIN admin_permission AS permission ON permission.permission_key = 'players:purge'
WHERE role.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
