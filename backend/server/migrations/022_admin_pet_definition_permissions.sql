-- 022_admin_pet_definition_permissions.sql
-- 系统宠物模板后台权限，与玩家宠物实例管理权限区分。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('pet_definitions:view', '查看系统宠物', 'pet_definitions', 'view', '查看系统宠物模板列表与详情', 1),
  ('pet_definitions:edit', '编辑系统宠物', 'pet_definitions', 'edit', '新增、编辑、删除系统宠物模板', 1)
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
JOIN admin_permission p ON p.permission_key IN ('pet_definitions:view', 'pet_definitions:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
