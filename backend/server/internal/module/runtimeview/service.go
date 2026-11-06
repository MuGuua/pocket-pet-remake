package runtimeview

import "context"

// PlayerCombatSnapshotRefresher 刷新人物最终战斗属性快照。
type PlayerCombatSnapshotRefresher interface {
	RefreshPlayerCombatSnapshot(ctx context.Context, playerID uint64) error
}

// PetCombatSnapshotRefresher 刷新宠物最终战斗属性快照。
type PetCombatSnapshotRefresher interface {
	RefreshPlayerPetCombatSnapshots(ctx context.Context, playerID uint64) error
}

// EquipmentSnapshotRefresher 刷新人物当前佩戴装备视图快照。
type EquipmentSnapshotRefresher interface {
	RefreshPlayerEquipmentSnapshot(ctx context.Context, playerID uint64) error
}

// SkillProgressSnapshotRefresher 刷新人物技能进度视图快照。
type SkillProgressSnapshotRefresher interface {
	RefreshPlayerSkillProgressSnapshot(ctx context.Context, playerID uint64) error
}

// Service 统一收口玩家运行时快照刷新入口。
// 当前 world / equipment / battle 入口只需依赖这一处，即可同步刷新人物、宠物、装备和技能进度快照。
type Service struct {
	playerRefresher    PlayerCombatSnapshotRefresher
	petRefresher       PetCombatSnapshotRefresher
	equipmentRefresher EquipmentSnapshotRefresher
	skillRefresher     SkillProgressSnapshotRefresher
}

// NewService 构造统一运行时快照刷新服务。
func NewService(
	playerRefresher PlayerCombatSnapshotRefresher,
	petRefresher PetCombatSnapshotRefresher,
	equipmentRefresher EquipmentSnapshotRefresher,
	skillRefresher SkillProgressSnapshotRefresher,
) *Service {
	return &Service{
		playerRefresher:    playerRefresher,
		petRefresher:       petRefresher,
		equipmentRefresher: equipmentRefresher,
		skillRefresher:     skillRefresher,
	}
}

// 当前用例内已成功刷新的视图标记，不跨请求保存，也不用于跳过后续用例所需刷新。
type RefreshResult struct{ Player, Pets, Equipment, Skills bool }

// 保留旧编排接口，所有原调用方仍按原顺序刷新完整视图。
func (s *Service) RefreshPlayerRuntimeSnapshots(ctx context.Context, playerID uint64) error {
	_, err := s.RefreshPlayerRuntimeSnapshotsWithResult(ctx, playerID)
	return err
}

// 返回本次实际执行成功的刷新标记，让随后读取复用刚生成的快照，避免同一用例重复重建。
func (s *Service) RefreshPlayerRuntimeSnapshotsWithResult(ctx context.Context, playerID uint64) (RefreshResult, error) {
	var result RefreshResult
	if s == nil || playerID == 0 {
		return result, nil
	}
	if s.playerRefresher != nil {
		if err := s.playerRefresher.RefreshPlayerCombatSnapshot(ctx, playerID); err != nil {
			return result, err
		}
		result.Player = true
	}
	if s.petRefresher != nil {
		if err := s.petRefresher.RefreshPlayerPetCombatSnapshots(ctx, playerID); err != nil {
			return result, err
		}
		result.Pets = true
	}
	if s.equipmentRefresher != nil {
		if err := s.equipmentRefresher.RefreshPlayerEquipmentSnapshot(ctx, playerID); err != nil {
			return result, err
		}
		result.Equipment = true
	}
	if s.skillRefresher != nil {
		if err := s.skillRefresher.RefreshPlayerSkillProgressSnapshot(ctx, playerID); err != nil {
			return result, err
		}
		result.Skills = true
	}
	return result, nil
}
