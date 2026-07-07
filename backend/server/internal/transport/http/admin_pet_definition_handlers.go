package httptransport

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/pet"
)

// AdminPetDefinitionHandler 负责后台系统宠物模板列表、详情与 CRUD。
// 模板决定玩家宠物是否可用，以及发放时的基础数值、成长资质与技能列表。
type AdminPetDefinitionHandler struct {
	adminService *admin.Service
	petService   *pet.Service
}

func (h *AdminPetDefinitionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.petService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin pet definition service is unavailable", nil)
		return
	}
	permission := "pet_definitions:view"
	if r.Method != http.MethodGet {
		permission = "pet_definitions:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/pet-definitions"))
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
	petIDValue, err := parseUintPathID(path)
	if err != nil || petIDValue == 0 || petIDValue > uint64(^uint32(0)) {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet_id", nil)
		return
	}
	petID := uint32(petIDValue)
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, petID)
	case http.MethodPut:
		h.handleUpdate(w, r, petID)
	case http.MethodDelete:
		h.handleDelete(w, r, petID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminPetDefinitionHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminPetDefinitionListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.petService.ListAdminPetDefinitions(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin pet definition list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminPetDefinitionHandler) handleDetail(w http.ResponseWriter, r *http.Request, petID uint32) {
	detail, err := h.petService.GetAdminPetDefinitionDetail(r.Context(), petID)
	if err != nil {
		if errors.Is(err, pet.ErrPetDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet definition not found", nil)
			return
		}
		log.Printf("[http-error] path=%s method=%s msg=load admin pet definition detail failed pet_id=%d err=%v", r.URL.Path, r.Method, petID, err)
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin pet definition detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminPetDefinitionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request pet.AdminUpsertPetDefinitionInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.petService.CreateAdminPetDefinition(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidAdminPetDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet definition payload", nil)
		case errors.Is(err, pet.ErrPetDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "pet definition already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin pet definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminPetDefinitionHandler) handleUpdate(w http.ResponseWriter, r *http.Request, petID uint32) {
	defer r.Body.Close()
	var request pet.AdminUpsertPetDefinitionInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petService.UpdateAdminPetDefinition(r.Context(), petID, request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidAdminPetDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet definition payload", nil)
		case errors.Is(err, pet.ErrPetDefinitionNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet definition not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin pet definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPetDefinitionHandler) handleDelete(w http.ResponseWriter, r *http.Request, petID uint32) {
	if err := h.petService.DeleteAdminPetDefinition(r.Context(), petID); err != nil {
		if errors.Is(err, pet.ErrPetDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin pet definition failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"pet_id": petID, "deleted": true})
}

func parseAdminPetDefinitionListQuery(r *http.Request) (pet.AdminPetDefinitionListQuery, error) {
	query := r.URL.Query()
	result := pet.AdminPetDefinitionListQuery{Name: strings.TrimSpace(query.Get("name"))}
	if raw := strings.TrimSpace(query.Get("pet_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return pet.AdminPetDefinitionListQuery{}, errors.New("pet_id must be an unsigned integer")
		}
		result.PetID = uint32(value)
	}
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return pet.AdminPetDefinitionListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return pet.AdminPetDefinitionListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}
