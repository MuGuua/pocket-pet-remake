package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
)

type AdminRepository struct {
	db DBTX
}

func NewAdminRepository(db DBTX) *AdminRepository {
	return &AdminRepository{db: db}
}

const findAdminByAccountNameQuery = `
SELECT
  u.id,
  u.account_name,
  u.password_hash,
  u.display_name,
  u.status,
  COALESCE(string_agg(DISTINCT r.role_key, ','), '') AS role_keys,
  COALESCE(string_agg(DISTINCT p.permission_key, ','), '') AS permission_keys
FROM admin_user u
LEFT JOIN admin_user_role ur ON ur.admin_user_id = u.id
LEFT JOIN admin_role r ON r.id = ur.admin_role_id AND r.status = 1
LEFT JOIN admin_role_permission rp ON rp.admin_role_id = r.id
LEFT JOIN admin_permission p ON p.id = rp.admin_permission_id AND p.status = 1
WHERE u.account_name = $1
GROUP BY u.id, u.account_name, u.password_hash, u.display_name, u.status
LIMIT 1
`

const findAdminByIDQuery = `
SELECT
  u.id,
  u.account_name,
  u.password_hash,
  u.display_name,
  u.status,
  COALESCE(string_agg(DISTINCT r.role_key, ','), '') AS role_keys,
  COALESCE(string_agg(DISTINCT p.permission_key, ','), '') AS permission_keys
FROM admin_user u
LEFT JOIN admin_user_role ur ON ur.admin_user_id = u.id
LEFT JOIN admin_role r ON r.id = ur.admin_role_id AND r.status = 1
LEFT JOIN admin_role_permission rp ON rp.admin_role_id = r.id
LEFT JOIN admin_permission p ON p.id = rp.admin_permission_id AND p.status = 1
WHERE u.id = $1
GROUP BY u.id, u.account_name, u.password_hash, u.display_name, u.status
LIMIT 1
`

const touchAdminLastLoginAtQuery = `
UPDATE admin_user
SET last_login_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`

func (r *AdminRepository) FindByAccountName(ctx context.Context, accountName string) (*admin.User, error) {
	return r.findOne(ctx, findAdminByAccountNameQuery, strings.TrimSpace(accountName))
}

func (r *AdminRepository) FindByID(ctx context.Context, adminUserID uint64) (*admin.User, error) {
	return r.findOne(ctx, findAdminByIDQuery, adminUserID)
}

func (r *AdminRepository) TouchLastLoginAt(ctx context.Context, adminUserID uint64) error {
	_, err := r.db.ExecContext(ctx, touchAdminLastLoginAtQuery, adminUserID)
	return err
}

func (r *AdminRepository) findOne(ctx context.Context, query string, arg any) (*admin.User, error) {
	var (
		user              admin.User
		roleKeysRaw       string
		permissionKeysRaw string
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&user.AdminUserID,
		&user.AccountName,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Status,
		&roleKeysRaw,
		&permissionKeysRaw,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.RoleKeys = splitCSV(roleKeysRaw)
	user.Permissions = splitCSV(permissionKeysRaw)
	return &user, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	values := strings.Split(raw, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

const insertAdminOperationLogQuery = `
INSERT INTO admin_operation_log (
  admin_user_id, module_key, action_key, target_type, target_id, reason, after_snapshot, extra_payload
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
`

// InsertOperationLog 把后台操作审计写入 admin_operation_log。
// 快照与附加信息序列化为 JSON；nil 时写入空对象，满足表上 NOT NULL DEFAULT '{}' 的约束口径。
func (r *AdminRepository) InsertOperationLog(ctx context.Context, entry admin.OperationLogEntry) error {
	afterJSON, err := marshalAdminLogJSON(entry.AfterSnapshot)
	if err != nil {
		return err
	}
	extraJSON, err := marshalAdminLogJSON(entry.ExtraPayload)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, insertAdminOperationLogQuery,
		entry.AdminUserID, entry.ModuleKey, entry.ActionKey, entry.TargetType, entry.TargetID, entry.Reason, afterJSON, extraJSON)
	return err
}

// marshalAdminLogJSON 把任意值序列化为审计 JSON 文本；nil 统一为 "{}"。
func marshalAdminLogJSON(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
