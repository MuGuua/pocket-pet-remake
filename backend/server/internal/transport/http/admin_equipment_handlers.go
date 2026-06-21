package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/equipment"
)

// AdminEquipmentDefinitionHandler 负责后台人物装备模板 CRUD。
type AdminEquipmentDefinitionHandler struct {
	adminService      *admin.Service
	equipmentService  *equipment.Service
}

func (h *AdminEquipmentDefinitionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/equipment-definitions"))
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
	itemID, err := parseUintPathID(path)
	if err != nil || itemID == 0 {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid item_id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, itemID)
	case http.MethodPut:
		h.handleUpdate(w, r, itemID)
	case http.MethodDelete:
		h.handleDelete(w, r, itemID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminEquipmentDefinitionHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminEquipmentListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.equipmentService.ListAdminEquipmentDefinitions(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin equipment list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminEquipmentDefinitionHandler) handleDetail(w http.ResponseWriter, r *http.Request, itemID uint64) {
	detail, err := h.equipmentService.GetAdminEquipmentDetail(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, equipment.ErrEquipmentDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "equipment definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin equipment detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminEquipmentDefinitionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request equipment.AdminUpsertEquipmentInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.equipmentService.CreateAdminEquipmentDefinition(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, equipment.ErrInvalidAdminEquipmentInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid equipment payload", nil)
		case errors.Is(err, equipment.ErrEquipmentDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "equipment definition already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin equipment failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminEquipmentDefinitionHandler) handleUpdate(w http.ResponseWriter, r *http.Request, itemID uint64) {
	defer r.Body.Close()
	var request equipment.AdminUpsertEquipmentInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.equipmentService.UpdateAdminEquipmentDefinition(r.Context(), itemID, request)
	if err != nil {
		switch {
		case errors.Is(err, equipment.ErrInvalidAdminEquipmentInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid equipment payload", nil)
		case errors.Is(err, equipment.ErrEquipmentDefinitionNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "equipment definition not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin equipment failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminEquipmentDefinitionHandler) handleDelete(w http.ResponseWriter, r *http.Request, itemID uint64) {
	if err := h.equipmentService.DeleteAdminEquipmentDefinition(r.Context(), itemID); err != nil {
		if errors.Is(err, equipment.ErrEquipmentDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "equipment definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin equipment failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"item_id": itemID, "deleted": true})
}

func parseAdminEquipmentListQuery(r *http.Request) (equipment.AdminListQuery, error) {
	query := r.URL.Query()
	result := equipment.AdminListQuery{Keyword: strings.TrimSpace(query.Get("keyword")), EquipSlot: strings.TrimSpace(query.Get("equip_slot"))}
	if raw := strings.TrimSpace(query.Get("item_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return equipment.AdminListQuery{}, err
		}
		result.ItemID = value
	}
	if raw := strings.TrimSpace(query.Get("set_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return equipment.AdminListQuery{}, err
		}
		result.SetID = value
	}
	if raw := strings.TrimSpace(query.Get("is_enabled")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true")
		result.Enabled = &value
	}
	if raw := strings.TrimSpace(query.Get("page")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return equipment.AdminListQuery{}, err
		}
		result.Page = uint32(value)
	}
	if raw := strings.TrimSpace(query.Get("page_size")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return equipment.AdminListQuery{}, err
		}
		result.PageSize = uint32(value)
	}
	return result, nil
}
