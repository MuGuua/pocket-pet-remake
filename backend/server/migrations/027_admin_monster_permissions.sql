-- 027_admin_monster_permissions.sql
-- 系统怪物模板与遭遇配置后台权限。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('monster_definitions:view', '查看系统怪物', 'monster_definitions', 'view', '查看系统怪物模板列表与详情', 1),
  ('monster_definitions:edit', '编辑系统怪物', 'monster_definitions', 'edit', '新增、编辑、删除系统怪物模板', 1),
  ('monster_encounters:view', '查看怪物遭遇', 'monster_encounters', 'view', '查看怪物遭遇配置列表与详情', 1),
  ('monster_encounters:edit', '编辑怪物遭遇', 'monster_encounters', 'edit', '新增、编辑、删除怪物遭遇配置', 1)
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
JOIN admin_permission p ON p.permission_key IN (
  'monster_definitions:view', 'monster_definitions:edit',
  'monster_encounters:view', 'monster_encounters:edit'
)
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
