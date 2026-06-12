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
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
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
	handlers := NewAdminHandlers(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	createBody := marshalJSON(t, pet.AdminCreatePetInput{PlayerID: teststub.DemoPlayerID, PetID: 101, Level: 6, Exp: 240, Quality: 2, HP: 40, HPMax: 42, ATK: 18, DEF: 14, SPD: 16, MANA: 21, SkillIDs: []uint32{1001, 1004}})
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

	updateBody := marshalJSON(t, pet.AdminUpdatePetInput{PetID: 102, Level: 7, Exp: 300, Quality: 3, HP: 50, HPMax: 55, ATK: 20, DEF: 15, SPD: 19, MANA: 24, SkillIDs: []uint32{1002}})
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

	createBody := marshalJSON(t, bag.AdminCreateItemInput{PlayerID: teststub.DemoPlayerID, ContainerType: "bag", SlotIndex: 8, ItemID: 2002, Quantity: 5})
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

	updateBody := marshalJSON(t, bag.AdminUpdateItemInput{PlayerID: teststub.RivalPlayerID, ContainerType: "warehouse", SlotIndex: 9, ItemID: 2003, Quantity: 9})
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

func TestAdminItemsListHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/items?page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer "+issueAdminTokenForTest(t))
	response := httptest.NewRecorder()

	handlers.Items.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAdminWalletsListAndAdjustHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/wallets?page=1&page_size=20", nil)
	listRequest.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	handlers.Wallets.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("wallet list response.Code = %d, want %d, body=%s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}

	adjustBody := marshalJSON(t, wallet.AdminAdjustInput{ChangeTotalCopper: 5000, Reason: "test adjust"})
	adjustRequest := httptest.NewRequest(http.MethodPut, "/api/admin/wallets/"+strconv.FormatUint(teststub.DemoPlayerID, 10), bytes.NewReader(adjustBody))
	adjustRequest.Header.Set("Authorization", "Bearer "+token)
	adjustResponse := httptest.NewRecorder()
	handlers.Wallets.ServeHTTP(adjustResponse, adjustRequest)
	if adjustResponse.Code != http.StatusOK {
		t.Fatalf("wallet adjust response.Code = %d, want %d, body=%s", adjustResponse.Code, http.StatusOK, adjustResponse.Body.String())
	}
}

func TestAdminRewardsGrantHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	requestBody := marshalJSON(t, adminRewardGrantRequest{
		PlayerID: teststub.DemoPlayerID,
		Reason:   "test admin reward",
		Rewards: []reward.Entry{
			{Type: "gold", Value: 3},
			{Type: "item", ItemID: 2001, Count: 2},
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/admin/rewards", bytes.NewReader(requestBody))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handlers.Rewards.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("reward grant response.Code = %d, want %d, body=%s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		Code int                      `json:"code"`
		Msg  string                   `json:"msg"`
		Data adminRewardGrantResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	if len(payload.Data.Granted) != 2 {
		t.Fatalf("len(payload.Data.Granted) = %d, want 2", len(payload.Data.Granted))
	}
	if payload.Data.Wallet == nil || payload.Data.Wallet.Gold <= 2 {
		t.Fatalf("payload.Data.Wallet = %#v, want wallet gold above initial 2", payload.Data.Wallet)
	}
	if payload.Data.Bag == nil || payload.Data.Bag.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("payload.Data.Bag = %#v, want bag snapshot", payload.Data.Bag)
	}
}

func TestAdminPetDefinitionCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, pet.AdminUpsertPetDefinitionInput{
		PetID: 99001, PetName: "后台测试宠物", Description: "用于后台 CRUD 测试", AcquireMethod: "运营发放",
		IsEnabled: true, Level: 1, Quality: 1, HP: 20, HPMax: 20, ATK: 10, DEF: 8, SPD: 9, MANA: 12,
		HPApt: 10, ATKApt: 10, DEFApt: 10, SPDApt: 10, MANAApt: 10, SkillIDs: []uint32{1001},
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/pet-definitions", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.PetDefinitions.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create pet definition response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, pet.AdminUpsertPetDefinitionInput{
		PetName: "后台测试宠物已更新", Description: "更新后的描述", AcquireMethod: "活动奖励",
		IsEnabled: false, Level: 2, Quality: 2, HP: 24, HPMax: 24, ATK: 11, DEF: 9, SPD: 10, MANA: 14,
		HPApt: 11, ATKApt: 12, DEFApt: 10, SPDApt: 9, MANAApt: 8, SkillIDs: []uint32{1001, 1002},
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/pet-definitions/99001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.PetDefinitions.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update pet definition response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/pet-definitions/99001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.PetDefinitions.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete pet definition response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminSkillDefinitionCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, skill.AdminUpsertInput{
		SkillID: 88001, SkillCode: "test_skill", SkillName: "后台测试技能", SkillCategory: "pet", SkillType: "attack",
		TargetType: "enemy_single", AnimationKey: "slash", IsEnabled: true, AttackPct: 110, ManaPct: 20, SpeedPct: 10, AllowCrit: true,
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/skill-definitions", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.SkillDefinitions.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create skill definition response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, skill.AdminUpsertInput{
		SkillName: "后台测试技能已更新", SkillCategory: "pet", SkillType: "attack", TargetType: "enemy_single",
		AnimationKey: "burst", IsEnabled: false, AttackPct: 120, ManaPct: 25, SpeedPct: 15, AllowCrit: true,
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/skill-definitions/88001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.SkillDefinitions.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update skill definition response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/skill-definitions/88001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.SkillDefinitions.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete skill definition response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminMonsterDefinitionCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, monster.AdminUpsertDefinitionInput{
		MonsterID: 88001, MonsterName: "后台测试怪物", Description: "测试怪物模板", IsEnabled: true,
		Level: 3, Quality: 1, HP: 30, HPMax: 30, ATK: 14, DEF: 10, SPD: 9, MANA: 8, SkillIDs: []uint32{90001, 90002},
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/monster-definitions", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.MonsterDefinitions.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create monster definition response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, monster.AdminUpsertDefinitionInput{
		MonsterName: "后台测试怪物已更新", Description: "更新后的怪物模板", IsEnabled: false,
		Level: 4, Quality: 2, HP: 32, HPMax: 32, ATK: 15, DEF: 11, SPD: 10, MANA: 9, SkillIDs: []uint32{90002},
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/monster-definitions/88001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.MonsterDefinitions.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update monster definition response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/monster-definitions/88001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.MonsterDefinitions.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete monster definition response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminMonsterEncounterCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, monster.AdminUpsertEncounterInput{
		EntityID: 88001, EncounterName: "后台测试遭遇", Description: "测试遭遇配置", SpawnMonsterIDs: []uint32{9001}, IsEnabled: true,
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/monster-encounters", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.MonsterEncounters.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create monster encounter response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, monster.AdminUpsertEncounterInput{
		EncounterName: "后台测试遭遇已更新", Description: "更新后的遭遇配置", SpawnMonsterIDs: []uint32{9001, 9002}, IsEnabled: true,
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/monster-encounters/88001", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.MonsterEncounters.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update monster encounter response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/monster-encounters/88001", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.MonsterEncounters.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete monster encounter response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestAdminSceneWildEncounterCRUDHandler(t *testing.T) {
	handlers := newAdminHandlersForTest(t)
	token := issueAdminTokenForTest(t)

	createBody := marshalJSON(t, monster.AdminUpsertWildEncounterInput{
		SceneID: 6, EncounterName: "后台测试暗雷", Description: "测试地图暗雷配置",
		EncounterRate: 1000, SpawnMonsterIDs: []uint32{9001}, IsEnabled: true,
	})
	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/scene-wild-encounters", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createResponse := httptest.NewRecorder()
	handlers.SceneWildEncounters.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("create scene wild encounter response.Code = %d, want %d, body=%s", createResponse.Code, http.StatusOK, createResponse.Body.String())
	}

	updateBody := marshalJSON(t, monster.AdminUpsertWildEncounterInput{
		EncounterName: "后台测试暗雷已更新", Description: "更新后的暗雷配置",
		EncounterRate: 1200, SpawnMonsterIDs: []uint32{9001, 9002}, IsEnabled: true,
	})
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/admin/scene-wild-encounters/6", bytes.NewReader(updateBody))
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	handlers.SceneWildEncounters.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update scene wild encounter response.Code = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/admin/scene-wild-encounters/6", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	handlers.SceneWildEncounters.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete scene wild encounter response.Code = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func newAdminHandlersForTest(t *testing.T) AdminHandlers {
	t.Helper()
	adminRepo := &adminRepoStub{user: &admin.User{AdminUserID: 1, AccountName: "admin", PasswordHash: admin.HashPassword("admin123"), DisplayName: "默认超级管理员", Status: 1, RoleKeys: []string{"super_admin"}, Permissions: []string{"players:view", "players:edit", "pets:view", "pets:edit", "pet_definitions:view", "pet_definitions:edit", "skill_definitions:view", "skill_definitions:edit", "monster_definitions:view", "monster_definitions:edit", "monster_encounters:view", "monster_encounters:edit", "scene_wild_encounters:view", "scene_wild_encounters:edit", "bag:view", "bag:grant", "items:view", "items:edit", "wallet:view", "wallet:edit", "quest:view", "quest:edit", "npcs:view", "npcs:edit"}}}
	adminService := admin.NewService(adminRepo, admin.NewHMACSigner("test-secret", time.Hour))
	skillRepo := teststub.NewSkillRepository()
	skillService := skill.NewService(skillRepo)
	if err := skillService.RefreshRuntimeCache(context.Background()); err != nil {
		t.Fatalf("RefreshRuntimeCache() error = %v", err)
	}
	monsterRepo := teststub.NewMonsterRepository()
	petRepo := teststub.NewPetRepository()
	petService := pet.NewService(petRepo, skillService, monsterRepo)
	monsterService := monster.NewService(monsterRepo, skillService, petService)
	playerService := player.NewService(teststub.NewPlayerRepository(), skillService)
	bagService := bag.NewService(teststub.NewBagRepository())
	itemService := item.NewService(teststub.NewItemRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	npcService := npc.NewService(teststub.NewNPCRepository())
	walletService := wallet.NewService(teststub.NewWalletRepository())
	unlockService := unlock.NewService(teststub.NewUnlockRepository())
	return NewAdminHandlers(adminService, playerService, petService, bagService, itemService, skillService, monsterService, questService, npcService, walletService, unlockService)
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
