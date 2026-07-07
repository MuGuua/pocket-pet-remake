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

// RefreshPlayerRuntimeSnapshots 顺序刷新该玩家所有运行时快照。
func (s *Service) RefreshPlayerRuntimeSnapshots(ctx context.Context, playerID uint64) error {
	if s == nil || playerID == 0 {
		return nil
	}
	if s.playerRefresher != nil {
		if err := s.playerRefresher.RefreshPlayerCombatSnapshot(ctx, playerID); err != nil {
			return err
		}
	}
	if s.petRefresher != nil {
		if err := s.petRefresher.RefreshPlayerPetCombatSnapshots(ctx, playerID); err != nil {
			return err
		}
	}
	if s.equipmentRefresher != nil {
		if err := s.equipmentRefresher.RefreshPlayerEquipmentSnapshot(ctx, playerID); err != nil {
			return err
		}
	}
	if s.skillRefresher != nil {
		if err := s.skillRefresher.RefreshPlayerSkillProgressSnapshot(ctx, playerID); err != nil {
			return err
		}
	}
	return nil
}
