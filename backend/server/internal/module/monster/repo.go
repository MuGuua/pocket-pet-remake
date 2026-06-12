package monster

import "context"

type Repository interface {
	ListDefinitionsForAdmin(ctx context.Context, query AdminDefinitionListQuery) (*AdminDefinitionList, error)
	FindDefinitionForAdmin(ctx context.Context, monsterID uint32) (*AdminDefinitionDetail, error)
	CreateDefinitionForAdmin(ctx context.Context, input AdminUpsertDefinitionInput) (*AdminDefinitionDetail, error)
	UpdateDefinitionForAdmin(ctx context.Context, monsterID uint32, input AdminUpsertDefinitionInput) (*AdminDefinitionDetail, error)
	DeleteDefinitionForAdmin(ctx context.Context, monsterID uint32) error
	MapUsableMonsterIDs(ctx context.Context, monsterIDs []uint32) (map[uint32]bool, error)
	FindRuntimeDefinition(ctx context.Context, monsterID uint32) (*RuntimeDefinition, error)
	FindCaptureConfig(ctx context.Context, monsterID uint32) (*CaptureConfig, error)

	ListEncountersForAdmin(ctx context.Context, query AdminEncounterListQuery) (*AdminEncounterList, error)
	FindEncounterForAdmin(ctx context.Context, entityID uint64) (*AdminEncounterDetail, error)
	CreateEncounterForAdmin(ctx context.Context, input AdminUpsertEncounterInput) (*AdminEncounterDetail, error)
	UpdateEncounterForAdmin(ctx context.Context, entityID uint64, input AdminUpsertEncounterInput) (*AdminEncounterDetail, error)
	DeleteEncounterForAdmin(ctx context.Context, entityID uint64) error
	FindRuntimeEncounter(ctx context.Context, entityID uint64) (*RuntimeEncounter, error)
	FindRuntimeWildEncounterConfig(ctx context.Context, sceneID uint32) (*RuntimeWildEncounterConfig, error)
	FindRuntimeWildEncounter(ctx context.Context, sceneID uint32) (*RuntimeWildEncounter, error)

	ListWildEncountersForAdmin(ctx context.Context, query AdminWildEncounterListQuery) (*AdminWildEncounterList, error)
	FindWildEncounterForAdmin(ctx context.Context, sceneID uint32) (*AdminWildEncounterDetail, error)
	CreateWildEncounterForAdmin(ctx context.Context, input AdminUpsertWildEncounterInput) (*AdminWildEncounterDetail, error)
	UpdateWildEncounterForAdmin(ctx context.Context, sceneID uint32, input AdminUpsertWildEncounterInput) (*AdminWildEncounterDetail, error)
	DeleteWildEncounterForAdmin(ctx context.Context, sceneID uint32) error
}
