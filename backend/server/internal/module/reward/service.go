package reward

import (
	"context"
	"errors"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
)

// Service 把正式玩法奖励收口到统一入口。
// 当前先复用 bag、wallet、pet、player、unlock 的既有权威链路，
// 让 quest 和 battle 不再各自拷贝一套发奖分支。
type Service struct {
	bagService    *bag.Service
	petService    *pet.Service
	playerService *player.Service
	unlockService *unlock.Service
	walletService *wallet.Service
}

// NewService 构造统一发奖服务。
func NewService(bagService *bag.Service, petService *pet.Service, playerService *player.Service, unlockService *unlock.Service, walletService *wallet.Service) *Service {
	return &Service{
		bagService:    bagService,
		petService:    petService,
		playerService: playerService,
		unlockService: unlockService,
		walletService: walletService,
	}
}

// GrantRuntimeRewards 逐条执行运行时奖励，并返回已经正式生效的聚合结果。
// 当前仍复用各领域已有持久化入口，后续若要升级成跨领域事务，可继续在这里统一下沉。
func (s *Service) GrantRuntimeRewards(ctx context.Context, input GrantInput) (*GrantResult, error) {
	result := &GrantResult{
		Granted:     make([]Entry, 0, len(input.Rewards)),
		GrantedPets: make([]pet.Pet, 0),
	}
	if input.PlayerID == 0 || input.ReasonType == "" || input.OperatorType == "" {
		return result, nil
	}
	rollbackState := rewardRollbackState{
		grantedItems: make([]grantedBagItem, 0),
	}
	// The execution order is intentional:
	// 1. reversible item grants first
	// 2. reversible wallet grants second
	// 3. hard-to-rollback rewards last
	// This reduces the chance of a gift box or quest reward leaving partial state.
	orderedRewards := orderGrantEntries(input.Rewards)
	for _, rewardEntry := range orderedRewards {
		if err := s.applyRewardEntry(ctx, input, result, &rollbackState, rewardEntry); err != nil {
			if rollbackErr := s.rollbackRuntimeRewards(ctx, input, rollbackState); rollbackErr != nil {
				return nil, errors.Join(err, rollbackErr)
			}
			return nil, err
		}
	}
	result.Granted = reorderGrantedEntries(input.Rewards, result.Granted)
	return result, nil
}

type grantedBagItem struct {
	SlotIndex uint32
	Quantity  uint64
}

type rewardRollbackState struct {
	grantedItems      []grantedBagItem
	grantedGoldCopper int64
}

// orderGrantEntries keeps the outward reward list untouched while executing
// reversible entries first inside the unified reward service.
func orderGrantEntries(entries []Entry) []Entry {
	reversibleItems := make([]Entry, 0, len(entries))
	reversibleWallet := make([]Entry, 0, len(entries))
	irreversible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		switch strings.ToLower(strings.TrimSpace(entry.Type)) {
		case "item":
			reversibleItems = append(reversibleItems, entry)
		case "gold":
			reversibleWallet = append(reversibleWallet, entry)
		default:
			irreversible = append(irreversible, entry)
		}
	}
	ordered := make([]Entry, 0, len(entries))
	ordered = append(ordered, reversibleItems...)
	ordered = append(ordered, reversibleWallet...)
	ordered = append(ordered, irreversible...)
	return ordered
}

// reorderGrantedEntries restores the outward reward list order expected by the
// protocol while still allowing the service to execute safer internal ordering.
func reorderGrantedEntries(original []Entry, granted []Entry) []Entry {
	if len(granted) <= 1 || len(original) <= 1 {
		return granted
	}
	used := make([]bool, len(granted))
	ordered := make([]Entry, 0, len(granted))
	for _, originEntry := range original {
		for index, grantedEntry := range granted {
			if used[index] || !sameRewardIdentity(originEntry, grantedEntry) {
				continue
			}
			ordered = append(ordered, grantedEntry)
			used[index] = true
			break
		}
	}
	for index, grantedEntry := range granted {
		if used[index] {
			continue
		}
		ordered = append(ordered, grantedEntry)
	}
	return ordered
}

func sameRewardIdentity(left Entry, right Entry) bool {
	return strings.EqualFold(strings.TrimSpace(left.Type), strings.TrimSpace(right.Type)) &&
		left.Value == right.Value &&
		left.ItemID == right.ItemID &&
		left.Count == right.Count &&
		left.PetID == right.PetID
}

// applyRewardEntry executes one formal reward branch and records enough state
// for best-effort rollback when a later reversible grant fails.
func (s *Service) applyRewardEntry(ctx context.Context, input GrantInput, result *GrantResult, rollbackState *rewardRollbackState, rewardEntry Entry) error {
	switch strings.ToLower(strings.TrimSpace(rewardEntry.Type)) {
	case "gold":
		if s.walletService == nil || rewardEntry.Value == 0 {
			return nil
		}
		changeTotalCopper := int64(rewardEntry.Value) * int64(wallet.CopperPerGold)
		adjusted, err := s.walletService.AdjustRuntimeWallet(ctx, input.PlayerID, wallet.RuntimeAdjustInput{
			ChangeTotalCopper: changeTotalCopper,
			ReasonType:        input.ReasonType,
			ReasonRefID:       input.ReasonRefID,
			OperatorType:      input.OperatorType,
			OperatorID:        input.OperatorID,
		})
		if err != nil {
			return err
		}
		rollbackState.grantedGoldCopper += changeTotalCopper
		walletCopy := adjusted.Wallet
		result.Wallet = &walletCopy
		result.Granted = append(result.Granted, rewardEntry)
		return nil
	case "exp":
		if s.playerService == nil || rewardEntry.Value == 0 {
			return nil
		}
		updatedProfile, err := s.playerService.AddExp(ctx, input.PlayerID, rewardEntry.Value)
		if err != nil {
			return err
		}
		result.PlayerProfile = updatedProfile
		result.Granted = append(result.Granted, rewardEntry)
		return nil
	case "item":
		if s.bagService == nil || rewardEntry.ItemID == 0 || rewardEntry.Count == 0 {
			return nil
		}
		grantedItem, err := s.bagService.GrantRuntimeItem(ctx, input.PlayerID, bag.ContainerTypeBag, rewardEntry.ItemID, rewardEntry.Count, input.ReasonType, input.ReasonRefID, input.OperatorType, input.OperatorID)
		if err != nil {
			return err
		}
		rollbackState.grantedItems = append(rollbackState.grantedItems, grantedBagItem{
			SlotIndex: grantedItem.SlotIndex,
			Quantity:  grantedItem.GrantedQty,
		})
		result.BagUpdated = true
		rewardEntry.ItemName = grantedItem.ItemName
		result.Granted = append(result.Granted, rewardEntry)
		return nil
	case "pet":
		if s.petService == nil || rewardEntry.PetID == 0 {
			return nil
		}
		grantedPet, err := s.petService.GrantRuntimePet(ctx, input.PlayerID, uint32(rewardEntry.PetID), input.ReasonType, input.ReasonRefID, input.OperatorType, input.OperatorID)
		if err != nil {
			return err
		}
		if grantedPet != nil {
			result.GrantedPets = append(result.GrantedPets, grantedPet.Pet)
			result.Granted = append(result.Granted, rewardEntry)
		}
		return nil
	case "feature_unlock":
		if s.unlockService == nil || rewardEntry.Value == 0 {
			return nil
		}
		if _, err := s.unlockService.GrantRuntimeFeature(ctx, input.PlayerID, rewardEntry.Value, input.ReasonType, input.ReasonRefID, input.OperatorType, input.OperatorID); err != nil {
			return err
		}
		result.Granted = append(result.Granted, rewardEntry)
		return nil
	default:
		return nil
	}
}

// rollbackRuntimeRewards only compensates the reversible branches that this
// service can safely undo today: bag item grants and wallet grants.
func (s *Service) rollbackRuntimeRewards(ctx context.Context, input GrantInput, rollbackState rewardRollbackState) error {
	var rollbackErr error
	for index := len(rollbackState.grantedItems) - 1; index >= 0; index-- {
		grantedItem := rollbackState.grantedItems[index]
		if grantedItem.SlotIndex == 0 || grantedItem.Quantity == 0 || s.bagService == nil {
			continue
		}
		if _, err := s.bagService.ConsumeRuntimeItemStack(ctx, input.PlayerID, bag.ContainerTypeBag, grantedItem.SlotIndex, grantedItem.Quantity, input.ReasonType+"_rollback", input.ReasonRefID); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	if rollbackState.grantedGoldCopper > 0 && s.walletService != nil {
		if _, err := s.walletService.AdjustRuntimeWallet(ctx, input.PlayerID, wallet.RuntimeAdjustInput{
			ChangeTotalCopper: -rollbackState.grantedGoldCopper,
			ReasonType:        input.ReasonType + "_rollback",
			ReasonRefID:       input.ReasonRefID,
			OperatorType:      input.OperatorType,
			OperatorID:        input.OperatorID,
		}); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}
