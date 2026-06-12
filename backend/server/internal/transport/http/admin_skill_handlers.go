package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/skill"
)

// AdminSkillDefinitionHandler 负责后台系统技能模板列表、详情与 CRUD。
type AdminSkillDefinitionHandler struct {
	adminService *admin.Service
	skillService *skill.Service
}

func (h *AdminSkillDefinitionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.skillService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin skill definition service is unavailable", nil)
		return
	}
	permission := "skill_definitions:view"
	if r.Method != http.MethodGet {
		permission = "skill_definitions:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/skill-definitions"))
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
	skillIDValue, err := parseUintPathID(path)
	if err != nil || skillIDValue == 0 || skillIDValue > uint64(^uint32(0)) {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill_id", nil)
		return
	}
	skillID := uint32(skillIDValue)
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, skillID)
	case http.MethodPut:
		h.handleUpdate(w, r, skillID)
	case http.MethodDelete:
		h.handleDelete(w, r, skillID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminSkillDefinitionHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminSkillListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.skillService.ListAdminSkillDefinitions(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin skill definition list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminSkillDefinitionHandler) handleDetail(w http.ResponseWriter, r *http.Request, skillID uint32) {
	detail, err := h.skillService.GetAdminSkillDefinitionDetail(r.Context(), skillID)
	if err != nil {
		if errors.Is(err, skill.ErrSkillDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "skill definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin skill definition detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminSkillDefinitionHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request skill.AdminUpsertInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.skillService.CreateAdminSkillDefinition(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, skill.ErrInvalidAdminSkillDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill definition payload", nil)
		case errors.Is(err, skill.ErrSkillDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "skill definition already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin skill definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminSkillDefinitionHandler) handleUpdate(w http.ResponseWriter, r *http.Request, skillID uint32) {
	defer r.Body.Close()
	var request skill.AdminUpsertInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.skillService.UpdateAdminSkillDefinition(r.Context(), skillID, request)
	if err != nil {
		switch {
		case errors.Is(err, skill.ErrInvalidAdminSkillDefinitionInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid skill definition payload", nil)
		case errors.Is(err, skill.ErrSkillDefinitionNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "skill definition not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin skill definition failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminSkillDefinitionHandler) handleDelete(w http.ResponseWriter, r *http.Request, skillID uint32) {
	if err := h.skillService.DeleteAdminSkillDefinition(r.Context(), skillID); err != nil {
		if errors.Is(err, skill.ErrSkillDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "skill definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin skill definition failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"skill_id": skillID, "deleted": true})
}

func parseAdminSkillListQuery(r *http.Request) (skill.AdminListQuery, error) {
	query := r.URL.Query()
	result := skill.AdminListQuery{Name: strings.TrimSpace(query.Get("name"))}
	if raw := strings.TrimSpace(query.Get("skill_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return skill.AdminListQuery{}, errors.New("skill_id must be an unsigned integer")
		}
		result.SkillID = uint32(value)
	}
	result.Category = strings.TrimSpace(query.Get("category"))
	result.Type = strings.TrimSpace(query.Get("skill_type"))
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return skill.AdminListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return skill.AdminListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}
