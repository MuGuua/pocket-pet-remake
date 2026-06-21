package httptransport

import (
	"errors"
	"net/http"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/pet"
)

// AdminPetSkillSlotUnlockHandler 负责神符槽解锁道具映射的后台 CRUD。
type AdminPetSkillSlotUnlockHandler struct {
	adminService *admin.Service
	petService   *pet.Service
}

func (h *AdminPetSkillSlotUnlockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.petService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin pet skill slot unlock service is unavailable", nil)
		return
	}
	permission := "pet_skill_slot_unlock:view"
	if r.Method != http.MethodGet {
		permission = "pet_skill_slot_unlock:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/pet-skill-slot-unlock-items"))
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
	slotKey := strings.Trim(strings.TrimPrefix(path, "/"), "/")
	if slotKey == "" {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid slot_key", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, slotKey)
	case http.MethodPut:
		h.handleUpdate(w, r, slotKey)
	case http.MethodDelete:
		h.handleDelete(w, r, slotKey)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminPetSkillSlotUnlockHandler) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := h.petService.ListAdminPetSkillSlotUnlockItems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet skill slot unlock list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", items)
}

func (h *AdminPetSkillSlotUnlockHandler) handleDetail(w http.ResponseWriter, r *http.Request, slotKey string) {
	item, err := h.petService.GetAdminPetSkillSlotUnlockItem(r.Context(), slotKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet skill slot unlock detail failed", nil)
		return
	}
	if item == nil {
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet skill slot unlock config not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", item)
}

func (h *AdminPetSkillSlotUnlockHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request pet.AdminUpsertPetSkillSlotUnlockInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.petService.CreateAdminPetSkillSlotUnlockItem(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidPetSkillSlotUnlockInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet skill slot unlock payload", nil)
		case errors.Is(err, pet.ErrPetSkillSlotUnlockConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "slot_key already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create pet skill slot unlock config failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminPetSkillSlotUnlockHandler) handleUpdate(w http.ResponseWriter, r *http.Request, slotKey string) {
	defer r.Body.Close()
	var request pet.AdminUpsertPetSkillSlotUnlockInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petService.UpdateAdminPetSkillSlotUnlockItem(r.Context(), slotKey, request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidPetSkillSlotUnlockInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet skill slot unlock payload", nil)
		case errors.Is(err, pet.ErrPetSkillSlotUnlockNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet skill slot unlock config not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update pet skill slot unlock config failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminPetSkillSlotUnlockHandler) handleDelete(w http.ResponseWriter, r *http.Request, slotKey string) {
	if err := h.petService.DeleteAdminPetSkillSlotUnlockItem(r.Context(), slotKey); err != nil {
		if errors.Is(err, pet.ErrPetSkillSlotUnlockNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet skill slot unlock config not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete pet skill slot unlock config failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", nil)
}
