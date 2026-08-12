-- 世界权威移动参数后台读写权限；迁移由用户手动执行。
INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('world_movement:view', '查看世界移动配置', 'world_movement', 'view', '查看服务端权威移动速度与弱网容差', 1),
  ('world_movement:edit', '编辑世界移动配置', 'world_movement', 'edit', '编辑服务端权威移动速度与弱网容差', 1)
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
JOIN admin_permission p ON p.permission_key IN ('world_movement:view', 'world_movement:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
