package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/teststub"
)

type adminRepoStub struct {
	user *admin.User
}

func (r *adminRepoStub) FindByAccountName(_ context.Context, accountName string) (*admin.User, error) {
	if r.user != nil && r.user.AccountName == accountName {
		copied := *r.user
		return &copied, nil
	}
	return nil, nil
}

func (r *adminRepoStub) FindByID(_ context.Context, adminUserID uint64) (*admin.User, error) {
	if r.user != nil && r.user.AdminUserID == adminUserID {
		copied := *r.user
		return &copied, nil
	}
	return nil, nil
}

func (r *adminRepoStub) TouchLastLoginAt(_ context.Context, adminUserID uint64) error {
	if r.user != nil && r.user.AdminUserID == adminUserID {
		now := time.Now()
		r.user.LastLoginAt = &now
	}
	return nil
}

func TestAdminHealthHandler(t *testing.T) {
	handlers := NewAdminHandlers(nil, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/healthz", nil)
	response := httptest.NewRecorder()

	handlers.Health.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestAdminPlayersListHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/players?page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer "+issueAdminTokenForTest(t))
	response := httptest.NewRecorder()

	handlers.Players.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAdminPlayerCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, player.AdminCreatePlayerInput{AccountName: "ops_user_1", Password: "ops123456", Name: "OpsTrainer", Level: 5, Gold: 999, SceneID: 2, HP: 180, HPMax: 180, Energy: 120, EnergyMax: 120, ATK: 30, DEF: 18, SPD: 22, MANA: 25, SkillIDs: []uint32{1101, 1001}})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/players", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.Players.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	var createPayload struct {
		Data player.AdminPlayerDetail `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}

	updateBody := marshalJSON(t, player.AdminUpdatePlayerInput{Name: "OpsTrainerUpdated", Level: 8, Exp: 500, Gold: 1200, SceneID: 3, PosX: 11, PosY: 9, HP: 220, HPMax: 220, Energy: 140, EnergyMax: 140, ATK: 35, DEF: 21, SPD: 24, MANA: 28, Status: 1, SkillIDs: []uint32{1101}})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/players/"+strconv.FormatUint(createPayload.Data.PlayerID, 10), bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.Players.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/players/"+strconv.FormatUint(createPayload.Data.PlayerID, 10), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.Players.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminPetsListHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/pets?page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer "+issueAdminTokenForTest(t))
	response := httptest.NewRecorder()

	handlers.Pets.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAdminPetCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, pet.AdminCreatePetInput{PlayerID: teststub.DemoPlayerID, PetID: 103, Level: 6, Exp: 240, Quality: 2, HP: 40, HPMax: 42, ATK: 18, DEF: 14, SPD: 16, MANA: 21, SkillIDs: []uint32{1001, 1004}})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/pets", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.Pets.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create pet response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	var createPayload struct {
		Data pet.AdminPetDetail `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("json.Unmarshal(create pet) error = %v", err)
	}

	updateBody := marshalJSON(t, pet.AdminUpdatePetInput{PetID: 104, Level: 7, Exp: 300, Quality: 3, HP: 50, HPMax: 55, ATK: 20, DEF: 15, SPD: 19, MANA: 24, SkillIDs: []uint32{1002}})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/pets/"+strconv.FormatUint(createPayload.Data.PetUID, 10), bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.Pets.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update pet response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/pets/"+strconv.FormatUint(createPayload.Data.PetUID, 10), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.Pets.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete pet response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminBagsListHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bags?page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer "+issueAdminTokenForTest(t))
	response := httptest.NewRecorder()

	handlers.Bags.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAdminBagCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, bag.AdminCreateItemInput{PlayerID: teststub.DemoPlayerID, ItemID: 2002, Count: 5})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/bags", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.Bags.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create bag response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	var createPayload struct {
		Data bag.AdminItemDetail `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("json.Unmarshal(create bag) error = %v", err)
	}

	updateBody := marshalJSON(t, bag.AdminUpdateItemInput{PlayerID: teststub.RivalPlayerID, ItemID: 2003, Count: 9})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/bags/"+strconv.FormatUint(createPayload.Data.RecordID, 10), bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.Bags.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update bag response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/bags/"+strconv.FormatUint(createPayload.Data.RecordID, 10), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.Bags.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete bag response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminQuestTemplateCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, quest.AdminCreateTemplateInput{
		QuestID: 9001, Name: "ops_quest", QuestType: "SIDE", Title: "后台新增任务", Description: "用于验证后台任务模板 CRUD。",
		Chapter: 9, SortOrder: 1, AcceptMode: "AUTO", SubmitMode: "AUTO", AutoTrack: true, MinPlayerLevel: 1, Status: 1,
		Objectives: []quest.AdminObjectiveInput{{ObjectiveID: 1, EventType: "ENTER_SCENE", Description: "进入测试场景", TargetValue: 1, TargetSelector: map[string]any{"scene_id": 99}}},
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/quests/templates", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create quest template response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, quest.AdminUpdateTemplateInput{
		Name: "ops_quest_updated", QuestType: "SIDE", Title: "后台更新任务", Description: "更新描述", Chapter: 10, SortOrder: 2,
		AcceptMode: "NPC", SubmitMode: "NPC", AutoTrack: false, StartNPCID: 93001, SubmitNPCID: 93001, MinPlayerLevel: 2, Status: 1,
		PreQuestIDs: []uint64{1001},
		Objectives:  []quest.AdminObjectiveInput{{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "和测试 NPC 对话", TargetValue: 1, TargetSelector: map[string]any{"npc_id": 93001}}},
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/quests/templates/9001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update quest template response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/quests/templates/9001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete quest template response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminPlayerQuestCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, quest.AdminCreatePlayerQuestInput{
		PlayerID: teststub.DemoPlayerID, QuestID: 1002, State: quest.StateAccepted, Tracked: true, RewardClaimed: false,
		Objectives: []quest.AdminPlayerObjectiveInput{{ObjectiveID: 1, Description: "与市场理萌交谈", CurrentValue: 0, TargetValue: 1, Completed: false}},
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/quests/player-progress", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create player quest response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	var createPayload struct {
		Data quest.AdminPlayerQuestDetail `json:"data"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("json.Unmarshal(create player quest) error = %v", err)
	}

	updateBody := marshalJSON(t, quest.AdminUpdatePlayerQuestInput{
		PlayerID: teststub.DemoPlayerID, QuestID: 1002, State: quest.StateCompleted, Tracked: false, RewardClaimed: true,
		Objectives: []quest.AdminPlayerObjectiveInput{{ObjectiveID: 1, Description: "与市场理萌交谈", CurrentValue: 1, TargetValue: 1, Completed: true}},
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/quests/player-progress/"+strconv.FormatUint(createPayload.Data.RecordID, 10), bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update player quest response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/quests/player-progress/"+strconv.FormatUint(createPayload.Data.RecordID, 10), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.Quests.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete player quest response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminNPCEntityCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, npc.AdminCreateEntityInput{EntityID: 99001, EntityCode: "ops_npc", DisplayName: "后台测试 NPC", EntityType: 2, SceneID: 7, PosX: 5, PosY: 9, Dir: 2, Speed: 0, Status: 1})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/npcs/entities", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create npc entity response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, npc.AdminUpdateEntityInput{EntityCode: "ops_npc_updated", DisplayName: "后台测试 NPC 已更新", EntityType: 2, SceneID: 8, PosX: 6, PosY: 10, Dir: 1, Speed: 0, Status: 1})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/npcs/entities/99001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update npc entity response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/npcs/entities/99001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete npc entity response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminNPCMenuEntryCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, npc.AdminCreateMenuEntryInput{EntityID: 93001, EntryID: "ops_dialog", EntryType: "dialog", Title: "后台菜单项", Subtitle: "给运营测试用", State: "available", Priority: 90, SortOrder: 30, ActionResultType: "notice", ActionNotice: "这是一条后台新增提示。", Status: 1})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/npcs/menu-entries", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create npc menu entry response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, npc.AdminUpdateMenuEntryInput{EntityID: 93002, EntryType: "shop", Title: "后台菜单项已更新", Subtitle: "改到另一个 NPC", State: "available", Priority: 91, SortOrder: 31, ActionResultType: "notice", ActionNotice: "后台已更新这条菜单项。", Status: 1})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/npcs/menu-entries/93001/ops_dialog", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update npc menu entry response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/npcs/menu-entries/93002/ops_dialog", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.NPCs.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete npc menu entry response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func newAdminHandlersForTest(t *testing.T) AdminHandlers {
	t.Helper()
	adminRepo := &adminRepoStub{user: &admin.User{AdminUserID: 1, AccountName: "admin", PasswordHash: admin.HashPassword("admin123"), DisplayName: "默认超级管理员", Status: 1, RoleKeys: []string{"super_admin"}, Permissions: []string{"players:view", "players:edit", "pets:view", "pets:edit", "bag:view", "bag:grant", "quest:view", "quest:edit", "npcs:view", "npcs:edit"}}}
	adminService := admin.NewService(adminRepo, admin.NewHMACSigner("test-secret", time.Hour))
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	bagService := bag.NewService(teststub.NewBagRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	npcService := npc.NewService(teststub.NewNPCRepository())
	return NewAdminHandlers(adminService, playerService, petService, bagService, questService, npcService)
}

func issueAdminTokenForTest(t *testing.T) string {
	t.Helper()
	signer := admin.NewHMACSigner("test-secret", time.Hour)
	token, _, err := signer.Sign(1, "admin")
	if err != nil {
		t.Fatalf("signer.Sign() error = %v", err)
	}
	return token
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}
