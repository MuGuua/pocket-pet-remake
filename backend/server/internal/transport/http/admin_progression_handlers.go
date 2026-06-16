package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/progression"
)

// AdminPlayerProgressionHandler 负责后台玩家成长配置管理。
type AdminPlayerProgressionHandler struct {
	adminService        *admin.Service
	progressionService  *progression.Service
}

func (h *AdminPlayerProgressionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.progressionService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin player progression service is unavailable", nil)
		return
	}
	permission := "player_progression:view"
	if r.Method != http.MethodGet {
		permission = "player_progression:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/player-progression"))
	segments := strings.Split(strings.Trim(path, "/"), "/")
	switch segments[0] {
	case "level-configs":
		h.handleLevelConfigs(w, r, segments)
	case "attr-convert-configs":
		h.handleAttrConvertConfigs(w, r, segments)
	default:
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "player progression route not found", nil)
	}
}

func (h *AdminPlayerProgressionHandler) handleLevelConfigs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		items, err := h.progressionService.ListAdminLevelConfigs(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load player level configs failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": items})
		return
	}
	levelValue, err := strconv.ParseUint(segments[1], 10, 32)
	if err != nil || levelValue == 0 || levelValue > uint64(progression.MaxPlayerLevel) {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid level", nil)
		return
	}
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request progression.AdminUpsertLevelConfigInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.progressionService.UpsertAdminLevelConfig(r.Context(), uint32(levelValue), request)
	if err != nil {
		if errors.Is(err, progression.ErrInvalidAllocateInput) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update player level config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPlayerProgressionHandler) handleAttrConvertConfigs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		items, err := h.progressionService.ListAdminAttrConvertConfigs(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load player attr convert configs failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": items})
		return
	}
	configID, err := strconv.ParseUint(segments[1], 10, 64)
	if err != nil || configID == 0 {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid config id", nil)
		return
	}
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request progression.AdminUpsertAttrConvertInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.progressionService.UpsertAdminAttrConvertConfig(r.Context(), configID, request)
	if err != nil {
		switch {
		case errors.Is(err, progression.ErrInvalidAllocateInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, progression.ErrConvertConfigNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "attr convert config not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update player attr convert config failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}
