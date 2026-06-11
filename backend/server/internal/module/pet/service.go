package pet

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPets(ctx context.Context, playerID uint64) ([]Pet, error) {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if pets == nil {
		pets = []Pet{}
	}

	lineup, err := s.ListLineup(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if len(lineup) == 0 {
		return pets, nil
	}

	lineupSet := make(map[uint64]struct{}, len(lineup))
	for _, lineupPet := range lineup {
		lineupSet[lineupPet.PetUID] = struct{}{}
	}
	for index := range pets {
		_, inLineup := lineupSet[pets[index].PetUID]
		pets[index].InLineup = inLineup
	}
	return pets, nil
}

func (s *Service) ListLineup(ctx context.Context, playerID uint64) ([]LineupPet, error) {
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if lineup == nil {
		return []LineupPet{}, nil
	}
	return lineup, nil
}

func (s *Service) SetLineup(ctx context.Context, playerID uint64, petUIDs []uint64) ([]LineupPet, error) {
	if len(petUIDs) == 0 {
		return nil, ErrInvalidLineup
	}

	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if len(pets) == 0 {
		return nil, ErrPetNotFound
	}

	owned := make(map[uint64]struct{}, len(pets))
	for _, item := range pets {
		owned[item.PetUID] = struct{}{}
	}

	seen := make(map[uint64]struct{}, len(petUIDs))
	for _, petUID := range petUIDs {
		if petUID == 0 {
			return nil, ErrInvalidLineup
		}
		if _, exists := seen[petUID]; exists {
			return nil, ErrDuplicateLineup
		}
		if _, exists := owned[petUID]; !exists {
			return nil, ErrPetNotFound
		}
		seen[petUID] = struct{}{}
	}

	if err := s.repo.SetLineupByPlayerID(ctx, playerID, petUIDs); err != nil {
		return nil, err
	}
	return s.ListLineup(ctx, playerID)
}

func (s *Service) UpdatePetHP(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (Pet, error) {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return Pet{}, err
	}

	var target *Pet
	for index := range pets {
		if pets[index].PetUID == petUID {
			target = &pets[index]
			break
		}
	}
	if target == nil {
		return Pet{}, ErrPetNotFound
	}

	if hp > target.HPMax {
		hp = target.HPMax
	}
	return s.repo.UpdatePetHPByUID(ctx, playerID, petUID, hp)
}

func (s *Service) UpdatePetBattleProgress(ctx context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (Pet, error) {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return Pet{}, err
	}

	var target *Pet
	for index := range pets {
		if pets[index].PetUID == petUID {
			target = &pets[index]
			break
		}
	}
	if target == nil {
		return Pet{}, ErrPetNotFound
	}
	if hp > target.HPMax {
		hp = target.HPMax
	}
	return s.repo.UpdatePetHPAndExpByUID(ctx, playerID, petUID, hp, expGain)
}

func (s *Service) ListAdminPets(ctx context.Context, query AdminListQuery) (*AdminPetList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPetList{Items: []AdminPetSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminPetDetail(ctx context.Context, petUID uint64) (*AdminPetDetail, error) {
	result, err := s.repo.FindAdminDetailByPetUID(ctx, petUID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminPet(ctx context.Context, input AdminCreatePetInput) (*AdminPetDetail, error) {
	input = input.Normalize()
	if input.PlayerID == 0 || input.PetID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	return s.repo.CreateForAdmin(ctx, input)
}

func (s *Service) UpdateAdminPet(ctx context.Context, petUID uint64, input AdminUpdatePetInput) (*AdminPetDetail, error) {
	input = input.Normalize()
	if input.PetID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	result, err := s.repo.UpdateForAdmin(ctx, petUID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminPet(ctx context.Context, petUID uint64) error {
	return s.repo.DeleteForAdmin(ctx, petUID)
}
