package httptransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
)

// AdminHandlers 聚合后台管理入口使用的基础 handler。
// 当前已经落地健康检查、管理员登录、当前用户信息，以及玩家/宠物管理 CRUD 一期闭环。
type AdminHandlers struct {
	Login              http.Handler
	Me                 http.Handler
	Health             http.Handler
	Players            http.Handler
	Pets               http.Handler
	Bags               http.Handler
	Items              http.Handler
	Quests             http.Handler
	NPCs               http.Handler
	Wallets            http.Handler
	Rewards            http.Handler
	PetDefinitions     http.Handler
	SkillDefinitions   http.Handler
	MonsterDefinitions     http.Handler
	MonsterEncounters      http.Handler
	SceneWildEncounters    http.Handler
	Dashboard              http.Handler
}

type AdminLoginHandler struct {
	service *admin.Service
}

type AdminPlayerHandler struct {
	adminService  *admin.Service
	playerService *player.Service
}

type AdminPetHandler struct {
	adminService *admin.Service
	petService   *pet.Service
}

type AdminBagHandler struct {
	adminService *admin.Service
	bagService   *bag.Service
}

type AdminQuestHandler struct {
	adminService *admin.Service
	questService *quest.Service
}

type AdminNPCHandler struct {
	adminService *admin.Service
	npcService   *npc.Service
}

type adminLoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func NewAdminHandlers(adminService *admin.Service, authService *auth.Service, sessionService *session.Service, playerService *player.Service, petService *pet.Service, bagService *bag.Service, itemService *item.Service, skillService *skill.Service, monsterService *monster.Service, questService *quest.Service, npcService *npc.Service, walletService *wallet.Service, unlockService *unlock.Service) AdminHandlers {
	rewardService := reward.NewService(bagService, petService, playerService, unlockService, walletService)
	return AdminHandlers{
		Login:              &AdminLoginHandler{service: adminService},
		Me:                 http.HandlerFunc(handleAdminMe(adminService)),
		Health:             http.HandlerFunc(handleAdminHealth),
		Dashboard:          &AdminDashboardHandler{adminService: adminService, authService: authService, playerService: playerService, sessionService: sessionService},
		Players:            &AdminPlayerHandler{adminService: adminService, playerService: playerService},
		Pets:               &AdminPetHandler{adminService: adminService, petService: petService},
		Bags:               &AdminBagHandler{adminService: adminService, bagService: bagService},
		Items:              &AdminItemHandler{adminService: adminService, itemService: itemService},
		Quests:             &AdminQuestHandler{adminService: adminService, questService: questService},
		NPCs:               &AdminNPCHandler{adminService: adminService, npcService: npcService},
		Wallets:            &AdminWalletHandler{adminService: adminService, walletService: walletService},
		Rewards:            &AdminRewardHandler{adminService: adminService, rewardService: rewardService, bagService: bagService},
		PetDefinitions:     &AdminPetDefinitionHandler{adminService: adminService, petService: petService},
		SkillDefinitions:   &AdminSkillDefinitionHandler{adminService: adminService, skillService: skillService},
		MonsterDefinitions:  &AdminMonsterDefinitionHandler{adminService: adminService, monsterService: monsterService},
		MonsterEncounters:   &AdminMonsterEncounterHandler{adminService: adminService, monsterService: monsterService},
		SceneWildEncounters: &AdminSceneWildEncounterHandler{adminService: adminService, monsterService: monsterService},
	}
}

func (h *AdminLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	if h.service == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin login service is unavailable", nil)
		return
	}
	defer r.Body.Close()

	var request adminLoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	result, err := h.service.Login(r.Context(), request.Account, request.Password)
	if err != nil {
		if errors.Is(err, admin.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, http.StatusUnauthorized, "invalid admin credentials", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin login failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminPlayerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.playerService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin player service is unavailable", nil)
		return
	}
	permission := "players:view"
	if r.Method != http.MethodGet {
		permission = "players:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/players"))
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}

	playerID, err := parseUintPathID(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player_id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, playerID)
	case http.MethodPut:
		h.handleUpdate(w, r, playerID)
	case http.MethodDelete:
		h.handleDelete(w, r, playerID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminPetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.petService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin pet service is unavailable", nil)
		return
	}
	permission := "pets:view"
	if r.Method != http.MethodGet {
		permission = "pets:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/pets"))
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}
	petUID, err := parseUintPathID(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet_uid", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, petUID)
	case http.MethodPut:
		h.handleUpdate(w, r, petUID)
	case http.MethodDelete:
		h.handleDelete(w, r, petUID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminBagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.bagService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin bag service is unavailable", nil)
		return
	}
	permission := "bag:view"
	if r.Method != http.MethodGet {
		permission = "bag:grant"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/bags"))
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}
	recordID, err := parseUintPathID(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid record_id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, recordID)
	case http.MethodPut:
		h.handleUpdate(w, r, recordID)
	case http.MethodDelete:
		h.handleDelete(w, r, recordID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminQuestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.questService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin quest service is unavailable", nil)
		return
	}
	permission := "quest:view"
	if r.Method != http.MethodGet {
		permission = "quest:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/quests"))
	switch {
	case path == "/templates" || path == "templates":
		switch r.Method {
		case http.MethodGet:
			h.handleTemplateList(w, r)
		case http.MethodPost:
			h.handleTemplateCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case strings.HasPrefix(path, "/templates/") || strings.HasPrefix(path, "templates/"):
		questID, err := parseNestedUintPathID(path, "templates")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid quest_id", nil)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handleTemplateDetail(w, r, questID)
		case http.MethodPut:
			h.handleTemplateUpdate(w, r, questID)
		case http.MethodDelete:
			h.handleTemplateDelete(w, r, questID)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case path == "/player-progress" || path == "player-progress":
		switch r.Method {
		case http.MethodGet:
			h.handlePlayerQuestList(w, r)
		case http.MethodPost:
			h.handlePlayerQuestCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case strings.HasPrefix(path, "/player-progress/") || strings.HasPrefix(path, "player-progress/"):
		recordID, err := parseNestedUintPathID(path, "player-progress")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid record_id", nil)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handlePlayerQuestDetail(w, r, recordID)
		case http.MethodPut:
			h.handlePlayerQuestUpdate(w, r, recordID)
		case http.MethodDelete:
			h.handlePlayerQuestDelete(w, r, recordID)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	default:
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "admin quest endpoint not found", nil)
	}
}

func (h *AdminNPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.npcService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin npc service is unavailable", nil)
		return
	}
	permission := "npcs:view"
	if r.Method != http.MethodGet {
		permission = "npcs:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/npcs"))
	switch {
	case path == "/entities" || path == "entities":
		switch r.Method {
		case http.MethodGet:
			h.handleEntityList(w, r)
		case http.MethodPost:
			h.handleEntityCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case strings.HasPrefix(path, "/entities/") || strings.HasPrefix(path, "entities/"):
		entityID, err := parseNestedUintPathID(path, "entities")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid entity_id", nil)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handleEntityDetail(w, r, entityID)
		case http.MethodPut:
			h.handleEntityUpdate(w, r, entityID)
		case http.MethodDelete:
			h.handleEntityDelete(w, r, entityID)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case path == "/menu-entries" || path == "menu-entries":
		switch r.Method {
		case http.MethodGet:
			h.handleMenuEntryList(w, r)
		case http.MethodPost:
			h.handleMenuEntryCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	case strings.HasPrefix(path, "/menu-entries/") || strings.HasPrefix(path, "menu-entries/"):
		entityID, entryID, err := parseNPCMenuEntryPath(path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid npc menu entry path", nil)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.handleMenuEntryDetail(w, r, entityID, entryID)
		case http.MethodPut:
			h.handleMenuEntryUpdate(w, r, entityID, entryID)
		case http.MethodDelete:
			h.handleMenuEntryDelete(w, r, entityID, entryID)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
	default:
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "admin npc endpoint not found", nil)
	}
}

func (h *AdminPlayerHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminPlayerListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.playerService.ListAdminPlayers(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin player list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminPlayerHandler) handleDetail(w http.ResponseWriter, r *http.Request, playerID uint64) {
	detail, err := h.playerService.GetAdminPlayerDetail(r.Context(), playerID)
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin player detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminPlayerHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request player.AdminCreatePlayerInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.playerService.CreateAdminPlayer(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, player.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player payload", nil)
		case errors.Is(err, player.ErrPlayerNameDuplicated):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "player name already exists", nil)
		case errors.Is(err, player.ErrAccountNameDuplicated):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "account name already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin player failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminPlayerHandler) handleUpdate(w http.ResponseWriter, r *http.Request, playerID uint64) {
	defer r.Body.Close()
	var request player.AdminUpdatePlayerInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.playerService.UpdateAdminPlayer(r.Context(), playerID, request)
	if err != nil {
		switch {
		case errors.Is(err, player.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player payload", nil)
		case errors.Is(err, player.ErrPlayerNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player not found", nil)
		case errors.Is(err, player.ErrPlayerNameDuplicated):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "player name already exists", nil)
		case errors.Is(err, skill.ErrInvalidSkillReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin player failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPlayerHandler) handleDelete(w http.ResponseWriter, r *http.Request, playerID uint64) {
	if err := h.playerService.DeleteAdminPlayer(r.Context(), playerID); err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin player failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"player_id": playerID, "deleted": true})
}

func (h *AdminPetHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminPetListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.petService.ListAdminPets(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin pet list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminPetHandler) handleDetail(w http.ResponseWriter, r *http.Request, petUID uint64) {
	detail, err := h.petService.GetAdminPetDetail(r.Context(), petUID)
	if err != nil {
		if errors.Is(err, pet.ErrPetNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin pet detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminPetHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request pet.AdminCreatePetInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.petService.CreateAdminPet(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidAdminPetInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet payload", nil)
		case errors.Is(err, pet.ErrPetNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player or pet not found", nil)
		case errors.Is(err, pet.ErrPetUnusable):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "pet definition unavailable", nil)
		case errors.Is(err, skill.ErrInvalidSkillReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin pet failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminPetHandler) handleUpdate(w http.ResponseWriter, r *http.Request, petUID uint64) {
	defer r.Body.Close()
	var request pet.AdminUpdatePetInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petService.UpdateAdminPet(r.Context(), petUID, request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidAdminPetInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet payload", nil)
		case errors.Is(err, pet.ErrPetNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet not found", nil)
		case errors.Is(err, skill.ErrInvalidSkillReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin pet failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPetHandler) handleDelete(w http.ResponseWriter, r *http.Request, petUID uint64) {
	if err := h.petService.DeleteAdminPet(r.Context(), petUID); err != nil {
		if errors.Is(err, pet.ErrPetNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin pet failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"pet_uid": petUID, "deleted": true})
}

func (h *AdminBagHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminBagListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.bagService.ListAdminItems(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin bag list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminBagHandler) handleDetail(w http.ResponseWriter, r *http.Request, recordID uint64) {
	detail, err := h.bagService.GetAdminItemDetail(r.Context(), recordID)
	if err != nil {
		if errors.Is(err, bag.ErrBagItemNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "bag item not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin bag detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminBagHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request bag.AdminCreateItemInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.bagService.CreateAdminItem(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrInvalidAdminBagInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid bag payload", nil)
		case errors.Is(err, bag.ErrBagItemConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "bag item already exists for player", nil)
		case errors.Is(err, bag.ErrBagItemNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin bag item failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminBagHandler) handleUpdate(w http.ResponseWriter, r *http.Request, recordID uint64) {
	defer r.Body.Close()
	var request bag.AdminUpdateItemInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.bagService.UpdateAdminItem(r.Context(), recordID, request)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrInvalidAdminBagInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid bag payload", nil)
		case errors.Is(err, bag.ErrBagItemConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "bag item already exists for player", nil)
		case errors.Is(err, bag.ErrBagItemNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "bag item or player not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin bag item failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminBagHandler) handleDelete(w http.ResponseWriter, r *http.Request, recordID uint64) {
	if err := h.bagService.DeleteAdminItem(r.Context(), recordID); err != nil {
		if errors.Is(err, bag.ErrBagItemNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "bag item not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin bag item failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"record_id": recordID, "deleted": true})
}

func (h *AdminQuestHandler) handleTemplateList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminQuestTemplateListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.questService.ListAdminTemplates(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin quest template list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminQuestHandler) handleTemplateDetail(w http.ResponseWriter, r *http.Request, questID uint64) {
	detail, err := h.questService.GetAdminTemplateDetail(r.Context(), questID)
	if err != nil {
		if errors.Is(err, quest.ErrAdminQuestTemplateNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "quest template not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin quest template detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminQuestHandler) handleTemplateCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request quest.AdminCreateTemplateInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.questService.CreateAdminTemplate(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, quest.ErrInvalidAdminQuestInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid quest template payload", nil)
		case errors.Is(err, quest.ErrAdminQuestConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "quest template already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin quest template failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminQuestHandler) handleTemplateUpdate(w http.ResponseWriter, r *http.Request, questID uint64) {
	defer r.Body.Close()
	var request quest.AdminUpdateTemplateInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.questService.UpdateAdminTemplate(r.Context(), questID, request)
	if err != nil {
		switch {
		case errors.Is(err, quest.ErrInvalidAdminQuestInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid quest template payload", nil)
		case errors.Is(err, quest.ErrAdminQuestTemplateNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "quest template not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin quest template failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminQuestHandler) handleTemplateDelete(w http.ResponseWriter, r *http.Request, questID uint64) {
	if err := h.questService.DeleteAdminTemplate(r.Context(), questID); err != nil {
		if errors.Is(err, quest.ErrAdminQuestTemplateNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "quest template not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin quest template failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"quest_id": questID, "deleted": true})
}

func (h *AdminQuestHandler) handlePlayerQuestList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminPlayerQuestListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.questService.ListAdminPlayerQuests(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin player quest list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminQuestHandler) handlePlayerQuestDetail(w http.ResponseWriter, r *http.Request, recordID uint64) {
	detail, err := h.questService.GetAdminPlayerQuestDetail(r.Context(), recordID)
	if err != nil {
		if errors.Is(err, quest.ErrAdminPlayerQuestNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player quest not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin player quest detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminQuestHandler) handlePlayerQuestCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request quest.AdminCreatePlayerQuestInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.questService.CreateAdminPlayerQuest(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, quest.ErrInvalidAdminQuestInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player quest payload", nil)
		case errors.Is(err, quest.ErrAdminQuestConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "player quest already exists", nil)
		case errors.Is(err, quest.ErrAdminQuestTemplateNotFound), errors.Is(err, quest.ErrAdminPlayerQuestNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player or quest template not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin player quest failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminQuestHandler) handlePlayerQuestUpdate(w http.ResponseWriter, r *http.Request, recordID uint64) {
	defer r.Body.Close()
	var request quest.AdminUpdatePlayerQuestInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.questService.UpdateAdminPlayerQuest(r.Context(), recordID, request)
	if err != nil {
		switch {
		case errors.Is(err, quest.ErrInvalidAdminQuestInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player quest payload", nil)
		case errors.Is(err, quest.ErrAdminQuestConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "player quest already exists", nil)
		case errors.Is(err, quest.ErrAdminQuestTemplateNotFound), errors.Is(err, quest.ErrAdminPlayerQuestNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player quest, player, or template not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin player quest failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminQuestHandler) handlePlayerQuestDelete(w http.ResponseWriter, r *http.Request, recordID uint64) {
	if err := h.questService.DeleteAdminPlayerQuest(r.Context(), recordID); err != nil {
		if errors.Is(err, quest.ErrAdminPlayerQuestNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player quest not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin player quest failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"record_id": recordID, "deleted": true})
}

func (h *AdminNPCHandler) handleEntityList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminNPCEntityListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.npcService.ListAdminEntities(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin npc entity list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminNPCHandler) handleEntityDetail(w http.ResponseWriter, r *http.Request, entityID uint64) {
	detail, err := h.npcService.GetAdminEntityDetail(r.Context(), entityID)
	if err != nil {
		if errors.Is(err, npc.ErrAdminNPCNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc entity not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin npc entity detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminNPCHandler) handleEntityCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request npc.AdminCreateEntityInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.npcService.CreateAdminEntity(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, npc.ErrInvalidAdminNPCInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid npc entity payload", nil)
		case errors.Is(err, npc.ErrAdminNPCConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "npc entity already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin npc entity failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminNPCHandler) handleEntityUpdate(w http.ResponseWriter, r *http.Request, entityID uint64) {
	defer r.Body.Close()
	var request npc.AdminUpdateEntityInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.npcService.UpdateAdminEntity(r.Context(), entityID, request)
	if err != nil {
		switch {
		case errors.Is(err, npc.ErrInvalidAdminNPCInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid npc entity payload", nil)
		case errors.Is(err, npc.ErrAdminNPCNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc entity not found", nil)
		case errors.Is(err, npc.ErrAdminNPCConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "npc entity already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin npc entity failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminNPCHandler) handleEntityDelete(w http.ResponseWriter, r *http.Request, entityID uint64) {
	if err := h.npcService.DeleteAdminEntity(r.Context(), entityID); err != nil {
		if errors.Is(err, npc.ErrAdminNPCNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc entity not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin npc entity failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"entity_id": entityID, "deleted": true})
}

func (h *AdminNPCHandler) handleMenuEntryList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminNPCMenuEntryListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.npcService.ListAdminMenuEntries(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin npc menu entry list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminNPCHandler) handleMenuEntryDetail(w http.ResponseWriter, r *http.Request, entityID uint64, entryID string) {
	detail, err := h.npcService.GetAdminMenuEntryDetail(r.Context(), entityID, entryID)
	if err != nil {
		if errors.Is(err, npc.ErrAdminNPCMenuEntryNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc menu entry not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin npc menu entry detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminNPCHandler) handleMenuEntryCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request npc.AdminCreateMenuEntryInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.npcService.CreateAdminMenuEntry(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, npc.ErrInvalidAdminNPCInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid npc menu entry payload", nil)
		case errors.Is(err, npc.ErrAdminNPCConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "npc menu entry already exists", nil)
		case errors.Is(err, npc.ErrAdminNPCNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc entity not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin npc menu entry failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminNPCHandler) handleMenuEntryUpdate(w http.ResponseWriter, r *http.Request, entityID uint64, entryID string) {
	defer r.Body.Close()
	var request npc.AdminUpdateMenuEntryInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.npcService.UpdateAdminMenuEntry(r.Context(), entityID, entryID, request)
	if err != nil {
		switch {
		case errors.Is(err, npc.ErrInvalidAdminNPCInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid npc menu entry payload", nil)
		case errors.Is(err, npc.ErrAdminNPCNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc entity not found", nil)
		case errors.Is(err, npc.ErrAdminNPCMenuEntryNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc menu entry not found", nil)
		case errors.Is(err, npc.ErrAdminNPCConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "npc menu entry already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin npc menu entry failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminNPCHandler) handleMenuEntryDelete(w http.ResponseWriter, r *http.Request, entityID uint64, entryID string) {
	if err := h.npcService.DeleteAdminMenuEntry(r.Context(), entityID, entryID); err != nil {
		if errors.Is(err, npc.ErrAdminNPCMenuEntryNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "npc menu entry not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin npc menu entry failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"entity_id": entityID, "entry_id": entryID, "deleted": true})
}

func handleAdminMe(service *admin.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		if service == nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin profile service is unavailable", nil)
			return
		}

		profile, ok := authenticateAdminRequest(w, r, service)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", profile)
	}
}

func handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"status": "ok", "scope": "admin"})
}

func authenticateAdminRequest(w http.ResponseWriter, r *http.Request, service *admin.Service, permissions ...string) (*admin.SessionProfile, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, http.StatusUnauthorized, "missing admin bearer token", nil)
		return nil, false
	}
	profile, err := service.ProfileByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, admin.ErrTokenInvalid) {
			writeJSON(w, http.StatusUnauthorized, http.StatusUnauthorized, "invalid admin token", nil)
			return nil, false
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin profile failed", nil)
		return nil, false
	}
	for _, permission := range permissions {
		if !hasAdminPermission(profile, permission) {
			writeJSON(w, http.StatusForbidden, http.StatusForbidden, fmt.Sprintf("missing permission: %s", permission), nil)
			return nil, false
		}
	}
	return profile, true
}

func hasAdminPermission(profile *admin.SessionProfile, permission string) bool {
	if profile == nil {
		return false
	}
	for _, current := range profile.Permissions {
		if strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(permission)) {
			return true
		}
	}
	return false
}

func parseAdminPlayerListQuery(r *http.Request) (player.AdminListQuery, error) {
	values := r.URL.Query()
	query := player.AdminListQuery{Name: values.Get("name")}
	if raw := strings.TrimSpace(values.Get("player_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return player.AdminListQuery{}, fmt.Errorf("invalid player_id")
		}
		query.PlayerID = parsed
	}
	if raw := strings.TrimSpace(values.Get("status")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return player.AdminListQuery{}, fmt.Errorf("invalid status")
		}
		status := uint32(parsed)
		query.Status = &status
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return player.AdminListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return player.AdminListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseAdminPetListQuery(r *http.Request) (pet.AdminListQuery, error) {
	values := r.URL.Query()
	query := pet.AdminListQuery{}
	if raw := strings.TrimSpace(values.Get("pet_uid")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return pet.AdminListQuery{}, fmt.Errorf("invalid pet_uid")
		}
		query.PetUID = parsed
	}
	if raw := strings.TrimSpace(values.Get("player_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return pet.AdminListQuery{}, fmt.Errorf("invalid player_id")
		}
		query.PlayerID = parsed
	}
	if raw := strings.TrimSpace(values.Get("pet_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return pet.AdminListQuery{}, fmt.Errorf("invalid pet_id")
		}
		query.PetID = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return pet.AdminListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return pet.AdminListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseAdminBagListQuery(r *http.Request) (bag.AdminListQuery, error) {
	values := r.URL.Query()
	query := bag.AdminListQuery{}
	if raw := strings.TrimSpace(values.Get("record_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return bag.AdminListQuery{}, fmt.Errorf("invalid record_id")
		}
		query.RecordID = parsed
	}
	if raw := strings.TrimSpace(values.Get("player_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return bag.AdminListQuery{}, fmt.Errorf("invalid player_id")
		}
		query.PlayerID = parsed
	}
	if raw := strings.TrimSpace(values.Get("item_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return bag.AdminListQuery{}, fmt.Errorf("invalid item_id")
		}
		query.ItemID = parsed
	}
	query.ContainerType = strings.TrimSpace(values.Get("container_type"))
	query.ItemUID = strings.TrimSpace(values.Get("item_uid"))
	page, pageSize, err := parsePageParams(values.Get("page"), values.Get("page_size"))
	if err != nil {
		return bag.AdminListQuery{}, err
	}
	query.Page = page
	query.PageSize = pageSize
	return query.Normalize(), nil
}

func parseAdminQuestTemplateListQuery(r *http.Request) (quest.AdminTemplateListQuery, error) {
	values := r.URL.Query()
	query := quest.AdminTemplateListQuery{QuestType: values.Get("quest_type"), Title: values.Get("title")}
	if raw := strings.TrimSpace(values.Get("quest_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return quest.AdminTemplateListQuery{}, fmt.Errorf("invalid quest_id")
		}
		query.QuestID = parsed
	}
	if raw := strings.TrimSpace(values.Get("status")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return quest.AdminTemplateListQuery{}, fmt.Errorf("invalid status")
		}
		status := uint32(parsed)
		query.Status = &status
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return quest.AdminTemplateListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return quest.AdminTemplateListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseAdminPlayerQuestListQuery(r *http.Request) (quest.AdminPlayerQuestListQuery, error) {
	values := r.URL.Query()
	query := quest.AdminPlayerQuestListQuery{State: values.Get("state")}
	if raw := strings.TrimSpace(values.Get("record_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid record_id")
		}
		query.RecordID = parsed
	}
	if raw := strings.TrimSpace(values.Get("player_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid player_id")
		}
		query.PlayerID = parsed
	}
	if raw := strings.TrimSpace(values.Get("quest_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid quest_id")
		}
		query.QuestID = parsed
	}
	if raw := strings.TrimSpace(values.Get("tracked")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid tracked")
		}
		query.Tracked = &parsed
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return quest.AdminPlayerQuestListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseAdminNPCEntityListQuery(r *http.Request) (npc.AdminEntityListQuery, error) {
	values := r.URL.Query()
	query := npc.AdminEntityListQuery{Name: values.Get("name")}
	if raw := strings.TrimSpace(values.Get("entity_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid entity_id")
		}
		query.EntityID = parsed
	}
	if raw := strings.TrimSpace(values.Get("scene_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid scene_id")
		}
		query.SceneID = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("entity_type")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid entity_type")
		}
		entityType := uint32(parsed)
		query.EntityType = &entityType
	}
	if raw := strings.TrimSpace(values.Get("status")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid status")
		}
		status := uint32(parsed)
		query.Status = &status
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminEntityListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseAdminNPCMenuEntryListQuery(r *http.Request) (npc.AdminMenuEntryListQuery, error) {
	values := r.URL.Query()
	query := npc.AdminMenuEntryListQuery{EntryID: values.Get("entry_id")}
	if raw := strings.TrimSpace(values.Get("entity_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return npc.AdminMenuEntryListQuery{}, fmt.Errorf("invalid entity_id")
		}
		query.EntityID = parsed
	}
	if raw := strings.TrimSpace(values.Get("status")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminMenuEntryListQuery{}, fmt.Errorf("invalid status")
		}
		status := uint32(parsed)
		query.Status = &status
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminMenuEntryListQuery{}, fmt.Errorf("invalid page")
		}
		query.Page = uint32(parsed)
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return npc.AdminMenuEntryListQuery{}, fmt.Errorf("invalid page_size")
		}
		query.PageSize = uint32(parsed)
	}
	return query.Normalize(), nil
}

func parseUintPathID(path string) (uint64, error) {
	valueText := strings.Trim(path, "/")
	value, err := strconv.ParseUint(valueText, 10, 64)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("invalid path id")
	}
	return value, nil
}

func parseNestedUintPathID(path string, prefix string) (uint64, error) {
	trimmed := strings.Trim(path, "/")
	segments := strings.Split(trimmed, "/")
	if len(segments) != 2 || segments[0] != prefix {
		return 0, fmt.Errorf("invalid nested path id")
	}
	return parseUintPathID(segments[1])
}

func parseNPCMenuEntryPath(path string) (uint64, string, error) {
	trimmed := strings.Trim(path, "/")
	segments := strings.Split(trimmed, "/")
	if len(segments) != 3 || segments[0] != "menu-entries" {
		return 0, "", fmt.Errorf("invalid npc menu entry path")
	}
	entityID, err := parseUintPathID(segments[1])
	if err != nil {
		return 0, "", err
	}
	entryID := strings.TrimSpace(segments[2])
	if entryID == "" {
		return 0, "", fmt.Errorf("invalid npc menu entry path")
	}
	return entityID, entryID, nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid request body")
	}
	return nil
}
