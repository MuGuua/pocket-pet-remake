package pet

import "context"

type Repository interface {
	ListPetsByPlayerID(ctx context.Context, playerID uint64) ([]Pet, error)
	ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
	SetLineupByPlayerID(ctx context.Context, playerID uint64, petUIDs []uint64) error
	UpdatePetHPByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (Pet, error)
	UpdatePetHPAndExpByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (Pet, error)
	ListForAdmin(ctx context.Context, query AdminListQuery) (*AdminPetList, error)
	FindAdminDetailByPetUID(ctx context.Context, petUID uint64) (*AdminPetDetail, error)
	CreateForAdmin(ctx context.Context, input AdminCreatePetInput) (*AdminPetDetail, error)
	UpdateForAdmin(ctx context.Context, petUID uint64, input AdminUpdatePetInput) (*AdminPetDetail, error)
	DeleteForAdmin(ctx context.Context, petUID uint64) error
}
