package pet

import "context"

type Repository interface {
	ListPetsByPlayerID(ctx context.Context, playerID uint64) ([]Pet, error)
	ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
	SetLineupByPlayerID(ctx context.Context, playerID uint64, petUIDs []uint64) error
	UpdatePetHPByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (Pet, error)
	UpdatePetHPAndExpByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (Pet, error)
	GrantRuntimePet(ctx context.Context, playerID uint64, petID uint32, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error)
	GrantWildCapturePet(ctx context.Context, playerID uint64, petID uint32, captureMonsterID uint32, reasonType string, reasonRefID uint64) (*RuntimeGrantResult, error)
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminPetList, error)
	FindAdminDetailByPetUID(ctx context.Context, petUID uint64) (*AdminPetDetail, error)
	CreateForAdmin(ctx context.Context, input AdminCreatePetInput) (*AdminPetDetail, error)
	UpdateForAdmin(ctx context.Context, petUID uint64, input AdminUpdatePetInput) (*AdminPetDetail, error)
	DeleteForAdmin(ctx context.Context, petUID uint64) error
	ListPetDefinitionsForAdmin(ctx context.Context, query AdminPetDefinitionListQuery) (*AdminPetDefinitionList, error)
	FindPetDefinitionForAdmin(ctx context.Context, petID uint32) (*AdminPetDefinitionDetail, error)
	FindPetSkinID(ctx context.Context, petID uint32) (string, error)
	CreatePetDefinitionForAdmin(ctx context.Context, input AdminUpsertPetDefinitionInput) (*AdminPetDefinitionDetail, error)
	UpdatePetDefinitionForAdmin(ctx context.Context, petID uint32, input AdminUpsertPetDefinitionInput) (*AdminPetDefinitionDetail, error)
	DeletePetDefinitionForAdmin(ctx context.Context, petID uint32) error
	MapUsablePetDefinitionIDs(ctx context.Context, petIDs []uint32) (map[uint32]bool, error)
}
