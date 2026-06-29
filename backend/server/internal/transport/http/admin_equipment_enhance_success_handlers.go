package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/equipment"
)

// AdminEquipmentEnhanceSuccessHandler 负责后台全局强化成功率配置。
type AdminEquipmentEnhanceSuccessHandler struct {
	adminService     *admin.Service
	equipmentService *equipment.Service
}

func (h *AdminEquipmentEnhanceSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.equipmentService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin equipment service is unavailable", nil)
		return
	}
	permission := "equipment_definitions:view"
	if r.Method != http.MethodGet {
		permission = "equipment_definitions:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}

	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/equipment-enhance-success-configs"))
	segments := strings.Split(strings.Trim(path, "/"), "/")
	switch len(segments) {
	case 0, 1:
		if segments[0] == "" && r.Method == http.MethodGet {
			h.handleList(w, r)
			return
		}
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid enhance success config path", nil)
	case 2:
		requiredLevelMinValue, err := strconv.ParseUint(segments[0], 10, 32)
		if err != nil || !equipment.IsValidEnhanceRequiredLevelBandMin(uint32(requiredLevelMinValue)) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid required_level_min", nil)
			return
		}
		targetLevelValue, err := strconv.ParseUint(segments[1], 10, 32)
		if err != nil || targetLevelValue == 0 || targetLevelValue > uint64(equipment.MaxEnhanceTargetLevel) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid target_level", nil)
			return
		}
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		h.handleUpdate(w, r, uint32(requiredLevelMinValue), uint32(targetLevelValue))
	default:
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid enhance success config path", nil)
	}
}

func (h *AdminEquipmentEnhanceSuccessHandler) handleList(w http.ResponseWriter, r *http.Request) {
	var requiredLevelMin *uint32
	if raw := strings.TrimSpace(r.URL.Query().Get("required_level_min")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || !equipment.IsValidEnhanceRequiredLevelBandMin(uint32(parsed)) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid required_level_min", nil)
			return
		}
		value := uint32(parsed)
		requiredLevelMin = &value
	}
	result, err := h.equipmentService.ListAdminEnhanceSuccessConfigs(r.Context(), requiredLevelMin)
	if err != nil {
		if errors.Is(err, equipment.ErrInvalidEnhanceSuccessConfig) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid required_level_min", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load enhance success configs failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminEquipmentEnhanceSuccessHandler) handleUpdate(
	w http.ResponseWriter,
	r *http.Request,
	requiredLevelMin uint32,
	targetLevel uint32,
) {
	defer r.Body.Close()
	var request equipment.AdminUpsertEnhanceSuccessConfigInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.equipmentService.UpsertAdminEnhanceSuccessConfig(r.Context(), targetLevel, requiredLevelMin, request)
	if err != nil {
		if errors.Is(err, equipment.ErrInvalidEnhanceSuccessConfig) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid enhance success config", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update enhance success config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}
