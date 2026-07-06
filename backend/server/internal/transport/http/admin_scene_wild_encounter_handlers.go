package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/monster"
)

// AdminSceneWildEncounterHandler 负责后台地图暗雷遭遇配置列表、详情与 CRUD。
type AdminSceneWildEncounterHandler struct {
	adminService   *admin.Service
	monsterService *monster.Service
}

func (h *AdminSceneWildEncounterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.monsterService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin scene wild encounter service is unavailable", nil)
		return
	}
	permission := "scene_wild_encounters:view"
	if r.Method != http.MethodGet {
		permission = "scene_wild_encounters:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/scene-wild-encounters"))
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
	sceneIDValue, err := parseUintPathID(path)
	if err != nil || sceneIDValue == 0 || sceneIDValue > uint64(^uint32(0)) {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene_id", nil)
		return
	}
	sceneID := uint32(sceneIDValue)
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, sceneID)
	case http.MethodPut:
		h.handleUpdate(w, r, sceneID)
	case http.MethodDelete:
		h.handleDelete(w, r, sceneID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminSceneWildEncounterHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminSceneWildEncounterListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.monsterService.ListAdminSceneWildEncounters(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin scene wild encounter list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminSceneWildEncounterHandler) handleDetail(w http.ResponseWriter, r *http.Request, sceneID uint32) {
	detail, err := h.monsterService.GetAdminSceneWildEncounterDetail(r.Context(), sceneID)
	if err != nil {
		if errors.Is(err, monster.ErrSceneWildEncounterNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "scene wild encounter not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin scene wild encounter detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminSceneWildEncounterHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request monster.AdminUpsertWildEncounterInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.monsterService.CreateAdminSceneWildEncounter(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminSceneWildEncounterInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene wild encounter payload", nil)
		case errors.Is(err, monster.ErrInvalidBattleRewardInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene wild encounter reward payload", nil)
		case errors.Is(err, monster.ErrSceneWildEncounterConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "scene wild encounter already exists", nil)
		case errors.Is(err, monster.ErrInvalidMonsterReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin scene wild encounter failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminSceneWildEncounterHandler) handleUpdate(w http.ResponseWriter, r *http.Request, sceneID uint32) {
	defer r.Body.Close()
	var request monster.AdminUpsertWildEncounterInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.monsterService.UpdateAdminSceneWildEncounter(r.Context(), sceneID, request)
	if err != nil {
		switch {
		case errors.Is(err, monster.ErrInvalidAdminSceneWildEncounterInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene wild encounter payload", nil)
		case errors.Is(err, monster.ErrInvalidBattleRewardInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene wild encounter reward payload", nil)
		case errors.Is(err, monster.ErrSceneWildEncounterNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "scene wild encounter not found", nil)
		case errors.Is(err, monster.ErrInvalidMonsterReference):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid monster reference", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin scene wild encounter failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminSceneWildEncounterHandler) handleDelete(w http.ResponseWriter, r *http.Request, sceneID uint32) {
	if err := h.monsterService.DeleteAdminSceneWildEncounter(r.Context(), sceneID); err != nil {
		if errors.Is(err, monster.ErrSceneWildEncounterNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "scene wild encounter not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin scene wild encounter failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"scene_id": sceneID, "deleted": true})
}

func parseAdminSceneWildEncounterListQuery(r *http.Request) (monster.AdminWildEncounterListQuery, error) {
	query := r.URL.Query()
	result := monster.AdminWildEncounterListQuery{Name: strings.TrimSpace(query.Get("name"))}
	if raw := strings.TrimSpace(query.Get("scene_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return monster.AdminWildEncounterListQuery{}, errors.New("scene_id must be an unsigned integer")
		}
		result.SceneID = uint32(value)
	}
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return monster.AdminWildEncounterListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return monster.AdminWildEncounterListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}
