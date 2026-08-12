package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/world"
)

const adminSceneBoundariesPath = "/api/admin/world/scene-boundaries"

// AdminWorldMovementHandler 管理数据库中的服务端权威移动配置和场景矩形边界。
type AdminWorldMovementHandler struct {
	adminService *admin.Service
	worldService *world.Service
}

// ServeHTTP 根据请求路径分派移动参数或场景边界接口，并统一执行后台权限校验。
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
	if r.URL.Path == "/api/admin/world/movement-config" {
		h.serveMovementConfig(w, r, profile.AdminUserID)
		return
	}
	if r.URL.Path == adminSceneBoundariesPath || strings.HasPrefix(r.URL.Path, adminSceneBoundariesPath+"/") {
		h.serveSceneBoundaries(w, r, profile.AdminUserID)
		return
	}
	writeJSON(w, http.StatusNotFound, http.StatusNotFound, "world movement endpoint not found", nil)
}

// serveMovementConfig 提供单例移动参数读取和更新，传输层只负责解析与错误映射。
func (h *AdminWorldMovementHandler) serveMovementConfig(w http.ResponseWriter, r *http.Request, adminUserID uint64) {
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
		input.AdminUserID = adminUserID
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

// serveSceneBoundaries 提供启用场景边界列表和单场景更新，运行时缓存刷新由领域服务完成。
func (h *AdminWorldMovementHandler) serveSceneBoundaries(w http.ResponseWriter, r *http.Request, adminUserID uint64) {
	if r.Method == http.MethodGet && r.URL.Path == adminSceneBoundariesPath {
		boundaries, err := h.worldService.GetAdminSceneBoundaries(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load world scene boundaries failed", nil)
			return
		}
		writeJSON(w, http.StatusOK, http.StatusOK, "success", boundaries)
		return
	}
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	sceneID, ok := parseSceneBoundaryID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene id", nil)
		return
	}
	defer r.Body.Close()
	var input world.AdminUpdateSceneBoundaryInput
	if err := decodeJSONBody(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	input.AdminUserID = adminUserID
	boundary, err := h.worldService.UpdateAdminSceneBoundary(r.Context(), sceneID, input)
	switch {
	case errors.Is(err, world.ErrSceneBoundaryInvalid):
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid world scene boundary", nil)
	case errors.Is(err, world.ErrSceneBoundaryUnavailable):
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "world scene boundary not found", nil)
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update world scene boundary failed", nil)
	default:
		writeJSON(w, http.StatusOK, http.StatusOK, "success", boundary)
	}
}

// parseSceneBoundaryID 仅接受 /scene-boundaries/{scene_id} 形式，避免模糊路径误更新场景。
func parseSceneBoundaryID(path string) (uint32, bool) {
	rawID := strings.TrimPrefix(path, adminSceneBoundariesPath+"/")
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, false
	}
	value, err := strconv.ParseUint(rawID, 10, 32)
	if err != nil || value == 0 {
		return 0, false
	}
	return uint32(value), true
}
