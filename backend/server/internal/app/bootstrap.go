package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/provider"
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
	"pocket-pet-remake/server/internal/module/playerskill"
	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/runtimeview"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
	httptransport "pocket-pet-remake/server/internal/transport/http"
	wstransport "pocket-pet-remake/server/internal/transport/ws"
)

// App is the main application struct.
// It holds the HTTP server, session service, logger, and cleanup closers.
type App struct {
	server         *http.Server
	sessionService *session.Service
	battleHandler  *wstransport.BattleHandler
	logger         *log.Logger
	cleanupClosers []io.Closer
}

func New(cfg config.Config, logger *log.Logger) (*App, error) {
	deps, closers, err := provider.OpenDependencies(cfg)
	if err != nil {
		return nil, err
	}
	return newApp(cfg, logger, deps, closers)
}

func NewWithDependencies(cfg config.Config, logger *log.Logger, deps provider.Dependencies) (*App, error) {
	return newApp(cfg, logger, deps, nil)
}

func newApp(cfg config.Config, logger *log.Logger, deps provider.Dependencies, closers []io.Closer) (*App, error) {
	repos, err := provider.NewConfiguredBundle(cfg, deps)
	if err != nil {
		return nil, err
	}

	signer := auth.NewHMACSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	adminSigner := admin.NewHMACSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(repos.Accounts, repos.WSTokens, signer, cfg.WSTokenTTL)
	adminService := admin.NewService(repos.Admins, adminSigner)
	bagService := bag.NewService(repos.Bags)
	itemService := item.NewService(repos.Items)
	skillService := skill.NewService(repos.Skills)
	if err := skillService.RefreshRuntimeCache(context.Background()); err != nil {
		return nil, fmt.Errorf("load skill runtime cache: %w", err)
	}
	battle.SetRuntimeSkillResolver(skillService.BattleResolver())
	progressionService := progression.NewService(repos.Progression)
	if err := progressionService.RefreshRuntimeCache(context.Background()); err != nil {
		return nil, fmt.Errorf("load player progression runtime cache: %w", err)
	}
	equipmentService := equipment.NewService(repos.Equipment, progressionService, repos.Players, repos.Pets, skillService)
	petProgressionService := petprogression.NewService(repos.PetProgression)
	if err := petProgressionService.RefreshRuntimeCache(context.Background()); err != nil {
		return nil, fmt.Errorf("load pet progression runtime cache: %w", err)
	}
	playerService := player.NewService(repos.Players, skillService, progressionService, equipmentService)
	playerSkillService := playerskill.NewService(repos.PlayerSkills)
	petService := pet.NewService(repos.Pets, skillService, repos.Monsters, petProgressionService)
	var playerSnapshotRefresher runtimeview.PlayerCombatSnapshotRefresher
	if refresher, ok := repos.Players.(runtimeview.PlayerCombatSnapshotRefresher); ok {
		playerSnapshotRefresher = refresher
	}
	var petSnapshotRefresher runtimeview.PetCombatSnapshotRefresher
	if refresher, ok := repos.Pets.(runtimeview.PetCombatSnapshotRefresher); ok {
		petSnapshotRefresher = refresher
	}
	var equipmentSnapshotRefresher runtimeview.EquipmentSnapshotRefresher
	if refresher, ok := repos.Equipment.(runtimeview.EquipmentSnapshotRefresher); ok {
		equipmentSnapshotRefresher = refresher
	}
	var skillSnapshotRefresher runtimeview.SkillProgressSnapshotRefresher
	if refresher, ok := repos.PlayerSkills.(runtimeview.SkillProgressSnapshotRefresher); ok {
		skillSnapshotRefresher = refresher
	}
	runtimeSnapshotService := runtimeview.NewService(
		playerSnapshotRefresher,
		petSnapshotRefresher,
		equipmentSnapshotRefresher,
		skillSnapshotRefresher,
	)
	questService := quest.NewService(repos.Quests)
	unlockService := unlock.NewService(repos.Unlocks)
	npcService := npc.NewService(repos.NPCs)
	npcDialogueService := npcdialogue.NewService(repos.NPCDialogues, &npcdialogue.QuestServiceAdapter{Service: questService})
	walletService := wallet.NewService(repos.Wallets)
	worldService := world.NewService(repos.World)
	monsterService := monster.NewService(repos.Monsters, skillService, petService)
	if err := monsterService.RefreshBattleRewardCache(context.Background()); err != nil {
		return nil, fmt.Errorf("load monster battle reward cache: %w", err)
	}
	battleService := battle.NewService(monsterService)
	if err := battleService.EnsureNextBattleID(context.Background(), repos.Battles); err != nil {
		return nil, fmt.Errorf("sync battle id cursor: %w", err)
	}
	equipmentService.SetBattleChecker(battleService)
	playerService.SetBattleChecker(battleService)
	battle.SetPetSkinResolver(petService.ResolveSkinID)
	sessionService := session.NewService(logger, cfg.HeartbeatInterval, cfg.HeartbeatTimeout)

	authHandler := wstransport.NewAuthHandler(authService, sessionService)
	worldHandler := wstransport.NewWorldHandler(sessionService, playerService, petService, questService, walletService, worldService, monsterService, equipmentService)
	petHandler := wstransport.NewPetHandler(sessionService, petService, petProgressionService)
	playerHandler := wstransport.NewPlayerHandler(sessionService, playerService)
	equipmentHandler := wstransport.NewEquipmentHandler(sessionService, equipmentService)
	battleHandler := wstransport.NewBattleHandler(sessionService, playerService, petService, bagService, walletService, worldService, questService, npcService, npcDialogueService, battleService, repos.Battles, equipmentService, playerSkillService, itemService)
	worldHandler.SetRuntimeSnapshotService(runtimeSnapshotService)
	equipmentHandler.SetRuntimeSnapshotService(runtimeSnapshotService)
	battleHandler.SetRuntimeSnapshotService(runtimeSnapshotService)
	bagHandler := wstransport.NewBagHandler(sessionService, bagService, itemService, walletService, playerService, petService, equipmentService, worldService, npcService)
	sessionService.SetDisconnectHandler(battleHandler.HandleSessionDisconnect)
	questHandler := wstransport.NewQuestHandler(questService, sessionService, bagService, petService, walletService, unlockService, playerService)
	wsRouter := wstransport.NewRouter(authHandler, worldHandler, petHandler, playerHandler, equipmentHandler, battleHandler, bagHandler, questHandler, sessionService)
	wsHub := wstransport.NewHub(logger, wsRouter, sessionService)
	loginHandler := httptransport.NewLoginHandler(authService)
	registerHandler := httptransport.NewRegisterHandler(playerService)
	adminHandlers := httptransport.NewAdminHandlers(adminService, authService, sessionService, playerService, petService, bagService, itemService, equipmentService, skillService, monsterService, questService, npcService, npcDialogueService, walletService, unlockService, progressionService, petProgressionService)
	httpHandler := buildHTTPHandler(loginHandler, registerHandler, adminHandlers, wsHub)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		server:         server,
		sessionService: sessionService,
		battleHandler:  battleHandler,
		logger:         logger,
		cleanupClosers: closers,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.closeResources()
	go a.sessionService.StartSweeper(ctx)
	if a.battleHandler != nil {
		go a.battleHandler.StartCustodySweeper(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.Printf("game server listening on %s", a.server.Addr)
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) closeResources() {
	for _, closer := range a.cleanupClosers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil {
			a.logger.Printf("close resource: %v", err)
		}
	}
}
