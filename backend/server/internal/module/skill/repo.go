package skill

import "context"

// Repository 定义系统技能模板的数据访问边界。
type Repository interface {
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminList, error)
	FindForAdmin(ctx context.Context, skillID uint32) (*AdminDetail, error)
	CreateForAdmin(ctx context.Context, input AdminUpsertInput) (*AdminDetail, error)
	UpdateForAdmin(ctx context.Context, skillID uint32, input AdminUpsertInput) (*AdminDetail, error)
	DeleteForAdmin(ctx context.Context, skillID uint32) error
	ListEnabledRuntimeDefinitions(ctx context.Context) ([]RuntimeDefinition, error)
	MapUsableSkillIDs(ctx context.Context, skillIDs []uint32) (map[uint32]bool, error)
	MapSkillCategoriesByIDs(ctx context.Context, skillIDs []uint32) (map[uint32]string, error)
	MapSkillWeaponDisciplinesByIDs(ctx context.Context, skillIDs []uint32) (map[uint32]string, error)
}
