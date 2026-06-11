INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('npcs:view', '查看 NPC 配置', 'npcs', 'view', '查看 NPC 与地图实体配置', 1),
  ('npcs:edit', '编辑 NPC 配置', 'npcs', 'edit', '编辑 NPC 与地图实体配置', 1)
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
JOIN admin_permission p ON p.permission_key IN ('npcs:view', 'npcs:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
