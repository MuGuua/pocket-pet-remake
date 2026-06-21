package provider

import (
	"fmt"

	"pocket-pet-remake/server/internal/config"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/npcdialogue"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/petprogression"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
)

type Bundle struct {
	Accounts     auth.AccountRepository
	Admins       admin.UserRepository
	Battles      battle.Repository
	Bags         bag.Repository
	Items        item.Repository
	Equipment    equipment.Repository
	Monsters     monster.Repository
	Skills       skill.Repository
	Players      player.Repository
	Progression     progression.Repository
	PetProgression  petprogression.Repository
	Pets            pet.Repository
	Quests       quest.Repository
	Unlocks      unlock.Repository
	NPCs         npc.Repository
	NPCDialogues npcdialogue.Repository
	Wallets      wallet.Repository
	World        world.Repository
	WSTokens     auth.WSTokenRepository
}

type Dependencies struct {
	Postgres pgrepo.DBTX
	Redis    redisrepo.Client
}

func NewConfiguredBundle(cfg config.Config, deps Dependencies) (Bundle, error) {
	if deps.Postgres == nil {
		return Bundle{}, fmt.Errorf("postgres query executor is required")
	}
	if deps.Redis == nil {
		return Bundle{}, fmt.Errorf("redis client is required")
	}

	return Bundle{
		Accounts:     pgrepo.NewAccountRepository(deps.Postgres),
		Admins:       pgrepo.NewAdminRepository(deps.Postgres),
		Battles:      pgrepo.NewBattleRepository(deps.Postgres),
		Bags:         pgrepo.NewBagRepository(deps.Postgres),
		Items:        pgrepo.NewItemRepository(deps.Postgres),
		Equipment:    pgrepo.NewEquipmentRepository(deps.Postgres),
		Monsters:     pgrepo.NewMonsterRepository(deps.Postgres),
		Skills:       pgrepo.NewSkillRepository(deps.Postgres),
		Players:      pgrepo.NewPlayerRepository(deps.Postgres),
		Progression:    pgrepo.NewProgressionRepository(deps.Postgres),
		PetProgression: pgrepo.NewPetProgressionRepository(deps.Postgres),
		Pets:           pgrepo.NewPetRepository(deps.Postgres),
		Quests:       pgrepo.NewQuestRepository(deps.Postgres),
		Unlocks:      pgrepo.NewUnlockRepository(deps.Postgres),
		NPCs:         pgrepo.NewNPCRepository(deps.Postgres),
		NPCDialogues: pgrepo.NewNPCDialogueRepository(deps.Postgres),
		Wallets:      pgrepo.NewWalletRepository(deps.Postgres),
		World:        pgrepo.NewWorldRepository(deps.Postgres),
		WSTokens:     redisrepo.NewWSTokenRepository(deps.Redis, cfg.Redis.KeyPrefix),
	}, nil
}
