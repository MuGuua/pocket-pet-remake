package httptransport

import (
	"errors"
	"net/http"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/pet"
)

// AdminPetCombatStatCapHandler 负责宠物战斗属性封顶配置的后台读写。
type AdminPetCombatStatCapHandler struct {
	adminService *admin.Service
	petService   *pet.Service
}

func (h *AdminPetCombatStatCapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.petService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin pet combat stat cap service is unavailable", nil)
		return
	}
	permission := "pet_combat_stat_cap:view"
	if r.Method != http.MethodGet {
		permission = "pet_combat_stat_cap:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/pet-combat-stat-caps"))
	if path == "" || path == "/" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		h.handleList(w, r)
		return
	}
	statKey := strings.Trim(strings.TrimPrefix(path, "/"), "/")
	if statKey == "" {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid stat_key", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, statKey)
	case http.MethodPut:
		h.handleUpdate(w, r, statKey)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminPetCombatStatCapHandler) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := h.petService.ListAdminPetCombatStatCaps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet combat stat caps failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", items)
}

func (h *AdminPetCombatStatCapHandler) handleDetail(w http.ResponseWriter, r *http.Request, statKey string) {
	item, err := h.petService.GetAdminPetCombatStatCap(r.Context(), statKey)
	if err != nil {
		if errors.Is(err, pet.ErrPetCombatStatCapNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet combat stat cap not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load pet combat stat cap failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", item)
}

func (h *AdminPetCombatStatCapHandler) handleUpdate(w http.ResponseWriter, r *http.Request, statKey string) {
	defer r.Body.Close()
	var request pet.AdminUpsertPetCombatStatCapInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.petService.UpdateAdminPetCombatStatCap(r.Context(), statKey, request)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrInvalidPetCombatStatCapInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid pet combat stat cap payload", nil)
		case errors.Is(err, pet.ErrPetCombatStatCapNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "pet combat stat cap not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update pet combat stat cap failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}
