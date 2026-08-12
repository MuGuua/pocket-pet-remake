package httptransport

import (
	"errors"
	"net/http"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/world"
)

// AdminWorldMovementHandler 管理数据库中的服务端权威移动配置。
type AdminWorldMovementHandler struct {
	adminService *admin.Service
	worldService *world.Service
}

// ServeHTTP 提供单例配置的读取和更新，并按操作类型检查后台权限。
func (h *AdminWorldMovementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.worldService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin world movement service is unavailable", nil)
		return
	}
	permission := "world_movement:view"
	if r.Method != http.MethodGet {
		permission = "world_movement:edit"
	}
	profile, ok := authenticateAdminRequest(w, r, h.adminService, permission)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		config, err := h.worldService.GetAdminMovementConfig(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load world movement config failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", config)
	case http.MethodPut:
		defer r.Body.Close()
		var input world.AdminUpdateMovementConfigInput
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		input.AdminUserID = profile.AdminUserID
		config, err := h.worldService.UpdateAdminMovementConfig(r.Context(), input)
		if errors.Is(err, world.ErrMovementConfigInvalid) {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid world movement config", nil)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update world movement config failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", config)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}
