package postgres

import (
	"context"
	"database/sql"
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
