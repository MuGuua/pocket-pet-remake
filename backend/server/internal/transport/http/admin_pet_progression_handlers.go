package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/petprogression"
)

// AdminPetProgressionHandler 负责后台宠物成长配置管理。
type AdminPetProgressionHandler struct {
	adminService          *admin.Service
	petProgressionService *petprogression.Service
}

func (h *AdminPetProgressionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.petProgressionService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin pet progression service is unavailable", nil)
		return
	}
	permission := "pet_progression:view"
	if r.Method != http.MethodGet {
		permission = "pet_progression:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/pet-progression"))
	segments := strings.Split(strings.Trim(path, "/"), "/")
	switch segments[0] {
	case "level-configs":
		h.handleLevelConfigs(w, r, segments)
	case "attr-convert-configs":
		h.handleAttrConvertConfigs(w, r, segments)
	case "recalculate-combat-stats":
		h.handleRecalculateCombatStats(w, r)
	default:
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet progression route not found", nil)
	}
}

func (h *AdminPetProgressionHandler) handleLevelConfigs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		items, err := h.petProgressionService.ListAdminLevelConfigs(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet level configs failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": items})
		return
	}
	levelValue, err := strconv.ParseUint(segments[1], 10, 32)
	if err != nil || levelValue == 0 || levelValue > uint64(petprogression.MaxPetLevel) {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid level", nil)
		return
	}
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request petprogression.AdminUpsertLevelConfigInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petProgressionService.UpsertAdminLevelConfig(r.Context(), uint32(levelValue), request)
	if err != nil {
		if errors.Is(err, petprogression.ErrInvalidAllocateInput) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update pet level config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPetProgressionHandler) handleAttrConvertConfigs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		items, err := h.petProgressionService.ListAdminConvertConfigs(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet attr convert configs failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"items": items})
		return
	}
	attrType := strings.TrimSpace(segments[1])
	if attrType == "" {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid attr type", nil)
		return
	}
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request petprogression.AdminUpsertConvertConfigInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petProgressionService.UpsertAdminConvertConfig(r.Context(), attrType, request)
	if err != nil {
		switch {
		case errors.Is(err, petprogression.ErrInvalidAllocateInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, petprogression.ErrConvertConfigNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "attr convert config not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update pet attr convert config failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPetProgressionHandler) handleRecalculateCombatStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request petprogression.AdminRecalculateCombatStatsInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.petProgressionService.RecalculateAllPetCombatStats(r.Context(), request)
	if err != nil {
		if errors.Is(err, petprogression.ErrInvalidAllocateInput) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "recalculate pet combat stats failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}
