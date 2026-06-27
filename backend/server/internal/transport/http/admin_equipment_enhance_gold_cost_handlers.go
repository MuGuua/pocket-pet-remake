package httptransport

import (
	"errors"
	"net/http"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/equipment"
)

// AdminEquipmentEnhanceGoldCostHandler 负责后台强化铜币公式配置。
type AdminEquipmentEnhanceGoldCostHandler struct {
	adminService     *admin.Service
	equipmentService *equipment.Service
}

func (h *AdminEquipmentEnhanceGoldCostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPut:
		h.handlePut(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminEquipmentEnhanceGoldCostHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	detail, err := h.equipmentService.GetAdminEnhanceGoldCostConfig(r.Context())
	if err != nil {
		if errors.Is(err, equipment.ErrEnhanceGoldCostConfigNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "enhance gold cost config not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load enhance gold cost config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminEquipmentEnhanceGoldCostHandler) handlePut(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request equipment.AdminUpsertEnhanceGoldCostConfigInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.equipmentService.UpdateAdminEnhanceGoldCostConfig(r.Context(), request)
	if err != nil {
		if errors.Is(err, equipment.ErrInvalidEnhanceGoldCostConfig) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid enhance gold cost config", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update enhance gold cost config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}
