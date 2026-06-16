package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/skill"
)

// AdminMonsterDefinitionHandler 负责后台系统怪物模板列表、详情与 CRUD。
type AdminMonsterDefinitionHandler struct {
	adminService   *admin.Service
	monsterService *monster.Service
}

func (h *AdminMonsterDefinitionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.monsterService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin monster definition service is unavailable", nil)
		return
	}
	permission := "monster_definitions:view"
	if r.Method != http.MethodGet {
		permission = "monster_definitions:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/monster-definitions"))
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
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
	monsterIDValue, err := strconv.ParseUint(segments[0], 10, 32)
	if err != nil || monsterIDValue == 0 {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster_id", nil)
		return
	}
	monsterID := uint32(monsterIDValue)
	if len(segments) >= 2 && segments[1] == "battle-rewards" {
		h.handleBattleRewards(w, r, monsterID)
		return
	}
	if len(segments) != 1 {
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster definition route not found", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, monsterID)
	case http.MethodPut:
		h.handleUpdate(w, r, monsterID)
	case http.MethodDelete:
		h.handleDelete(w, r, monsterID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminMonsterDefinitionHandler) handleBattleRewards(w http.ResponseWriter, r *http.Request, monsterID uint32) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.monsterService.ListAdminBattleRewards(r.Context(), monsterID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load monster battle rewards failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": items})
	case http.MethodPut:
		defer r.Body.Close()
		var request monster.AdminReplaceBattleRewardsInput
		if err := decodeJSONBody(w, r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		updated, err := h.monsterService.ReplaceAdminBattleRewards(r.Context(), monsterID, request)
		if err != nil {
			if errors.Is(err, monster.ErrInvalidBattleRewardInput) {
				writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
				return
			}
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update monster battle rewards failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": updated})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminMonsterDefinitionHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminMonsterDefinitionListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.monsterService.ListAdminMonsterDefinitions(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin monster definition list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminMonsterDefinitionHandler) handleDetail(w http.ResponseWriter, r *http.Request, monsterID uint32) {
	detail, err := h.monsterService.GetAdminMonsterDefinitionDetail(r.Context(), monsterID)
	if err != nil {
		if errors.Is(err, monster.ErrMonsterDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin monster definition detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminMonsterDefinitionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request monster.AdminUpsertDefinitionInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.monsterService.CreateAdminMonsterDefinition(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminMonsterDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster definition payload", nil)
		case errors.Is(err, monster.ErrMonsterDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "monster definition already exists", nil)
		case errors.Is(err, skill.ErrInvalidSkillReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin monster definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminMonsterDefinitionHandler) handleUpdate(w http.ResponseWriter, r *http.Request, monsterID uint32) {
	defer r.Body.Close()
	var request monster.AdminUpsertDefinitionInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.monsterService.UpdateAdminMonsterDefinition(r.Context(), monsterID, request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminMonsterDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster definition payload", nil)
		case errors.Is(err, monster.ErrMonsterDefinitionNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster definition not found", nil)
		case errors.Is(err, skill.ErrInvalidSkillReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin monster definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminMonsterDefinitionHandler) handleDelete(w http.ResponseWriter, r *http.Request, monsterID uint32) {
	if err := h.monsterService.DeleteAdminMonsterDefinition(r.Context(), monsterID); err != nil {
		if errors.Is(err, monster.ErrMonsterDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin monster definition failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"monster_id": monsterID, "deleted": true})
}

// AdminMonsterEncounterHandler 负责后台怪物遭遇配置列表、详情与 CRUD。
type AdminMonsterEncounterHandler struct {
	adminService   *admin.Service
	monsterService *monster.Service
}

func (h *AdminMonsterEncounterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.monsterService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin monster encounter service is unavailable", nil)
		return
	}
	permission := "monster_encounters:view"
	if r.Method != http.MethodGet {
		permission = "monster_encounters:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/monster-encounters"))
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
	entityID, err := parseUintPathID(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid entity_id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, entityID)
	case http.MethodPut:
		h.handleUpdate(w, r, entityID)
	case http.MethodDelete:
		h.handleDelete(w, r, entityID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminMonsterEncounterHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminMonsterEncounterListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.monsterService.ListAdminMonsterEncounters(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin monster encounter list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminMonsterEncounterHandler) handleDetail(w http.ResponseWriter, r *http.Request, entityID uint64) {
	detail, err := h.monsterService.GetAdminMonsterEncounterDetail(r.Context(), entityID)
	if err != nil {
		if errors.Is(err, monster.ErrMonsterEncounterNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster encounter not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin monster encounter detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminMonsterEncounterHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request monster.AdminUpsertEncounterInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.monsterService.CreateAdminMonsterEncounter(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminMonsterEncounterInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster encounter payload", nil)
		case errors.Is(err, monster.ErrMonsterEncounterConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "monster encounter already exists", nil)
		case errors.Is(err, monster.ErrInvalidMonsterReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin monster encounter failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminMonsterEncounterHandler) handleUpdate(w http.ResponseWriter, r *http.Request, entityID uint64) {
	defer r.Body.Close()
	var request monster.AdminUpsertEncounterInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.monsterService.UpdateAdminMonsterEncounter(r.Context(), entityID, request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminMonsterEncounterInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster encounter payload", nil)
		case errors.Is(err, monster.ErrMonsterEncounterNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster encounter not found", nil)
		case errors.Is(err, monster.ErrInvalidMonsterReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin monster encounter failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminMonsterEncounterHandler) handleDelete(w http.ResponseWriter, r *http.Request, entityID uint64) {
	if err := h.monsterService.DeleteAdminMonsterEncounter(r.Context(), entityID); err != nil {
		if errors.Is(err, monster.ErrMonsterEncounterNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "monster encounter not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin monster encounter failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"entity_id": entityID, "deleted": true})
}

func parseAdminMonsterDefinitionListQuery(r *http.Request) (monster.AdminDefinitionListQuery, error) {
	query := r.URL.Query()
	result := monster.AdminDefinitionListQuery{Name: strings.TrimSpace(query.Get("name"))}
	if raw := strings.TrimSpace(query.Get("monster_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return monster.AdminDefinitionListQuery{}, errors.New("monster_id must be an unsigned integer")
		}
		result.MonsterID = uint32(value)
	}
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return monster.AdminDefinitionListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return monster.AdminDefinitionListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}

func parseAdminMonsterEncounterListQuery(r *http.Request) (monster.AdminEncounterListQuery, error) {
	query := r.URL.Query()
	result := monster.AdminEncounterListQuery{Name: strings.TrimSpace(query.Get("name"))}
	if raw := strings.TrimSpace(query.Get("entity_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return monster.AdminEncounterListQuery{}, errors.New("entity_id must be an unsigned integer")
		}
		result.EntityID = value
	}
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return monster.AdminEncounterListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return monster.AdminEncounterListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}
