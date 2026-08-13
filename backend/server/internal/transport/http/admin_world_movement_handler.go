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
const adminSceneNavigationsPath = "/api/admin/world/scene-navigations"

// AdminWorldMovementHandler 管理数据库中的权威移动参数、场景边界和静态通行版本。
type AdminWorldMovementHandler struct {
	adminService *admin.Service
	worldService *world.Service
}

// ServeHTTP 根据请求路径分派移动参数、场景边界或导航版本接口，并统一执行后台权限校验。
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
	if r.URL.Path == adminSceneNavigationsPath || strings.HasPrefix(r.URL.Path, adminSceneNavigationsPath+"/") {
		h.serveSceneNavigations(w, r, profile.AdminUserID)
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

// serveSceneNavigations 提供导航版本列表、草稿上传、发布和回滚，具体事务规则由领域服务和仓储负责。
func (h *AdminWorldMovementHandler) serveSceneNavigations(w http.ResponseWriter, r *http.Request, adminUserID uint64) {
	if r.URL.Path == adminSceneNavigationsPath {
		h.serveSceneNavigationCollection(w, r, adminUserID)
		return
	}
	navigationID, sceneID, action, ok := parseSceneNavigationAction(r.URL.Path)
	if !ok || r.Method != http.MethodPost {
		if !ok {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene navigation path", nil)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}
	defer r.Body.Close()
	switch action {
	case "publish":
		var input world.AdminPublishSceneNavigationInput
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		input.AdminUserID = adminUserID
		navigation, err := h.worldService.PublishAdminSceneNavigation(r.Context(), navigationID, input)
		h.writeSceneNavigationMutationResult(w, navigation, err)
	case "rollback":
		var input world.AdminRollbackSceneNavigationInput
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		input.AdminUserID = adminUserID
		navigation, err := h.worldService.RollbackAdminSceneNavigation(r.Context(), sceneID, input)
		h.writeSceneNavigationMutationResult(w, navigation, err)
	default:
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene navigation action", nil)
	}
}

// serveSceneNavigationCollection 处理按场景查询版本和上传新草稿。
func (h *AdminWorldMovementHandler) serveSceneNavigationCollection(w http.ResponseWriter, r *http.Request, adminUserID uint64) {
	switch r.Method {
	case http.MethodGet:
		sceneID, ok := parseRequiredUint32(r.URL.Query().Get("scene_id"))
		if !ok {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid scene id", nil)
			return
		}
		navigations, err := h.worldService.GetAdminSceneNavigations(r.Context(), sceneID)
		switch {
		case errors.Is(err, world.ErrSceneNavigationNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "world scene navigation not found", nil)
		case errors.Is(err, world.ErrSceneNavigationInvalid):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid world scene navigation query", nil)
		case err != nil:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load world scene navigations failed", nil)
		default:
			writeJSON(w, http.StatusOK, http.StatusOK, "success", navigations)
		}
	case http.MethodPost:
		defer r.Body.Close()
		var input world.AdminCreateSceneNavigationInput
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
			return
		}
		input.AdminUserID = adminUserID
		navigation, err := h.worldService.CreateAdminSceneNavigationDraft(r.Context(), input)
		h.writeSceneNavigationMutationResult(w, navigation, err)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

// writeSceneNavigationMutationResult 统一映射导航写接口错误，避免三个入口出现不一致语义。
func (h *AdminWorldMovementHandler) writeSceneNavigationMutationResult(w http.ResponseWriter, navigation world.SceneNavigation, err error) {
	switch {
	case errors.Is(err, world.ErrSceneNavigationInvalid):
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid world scene navigation", nil)
	case errors.Is(err, world.ErrSceneNavigationNotFound):
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "world scene navigation not found", nil)
	case errors.Is(err, world.ErrSceneNavigationStateInvalid):
		writeJSON(w, http.StatusConflict, http.StatusConflict, "world scene navigation status does not allow this operation", nil)
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update world scene navigation failed", nil)
	default:
		writeJSON(w, http.StatusOK, http.StatusOK, "success", navigation)
	}
}

// parseSceneNavigationAction 区分版本发布路径和场景回滚路径，禁止把两个标识混用。
func parseSceneNavigationAction(path string) (uint64, uint32, string, bool) {
	raw := strings.TrimPrefix(path, adminSceneNavigationsPath+"/")
	parts := strings.Split(raw, "/")
	if len(parts) == 2 && parts[1] == "publish" {
		navigationID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil || navigationID == 0 {
			return 0, 0, "", false
		}
		return navigationID, 0, "publish", true
	}
	if len(parts) == 3 && parts[0] == "scenes" && parts[2] == "rollback" {
		sceneID, ok := parseRequiredUint32(parts[1])
		if !ok {
			return 0, 0, "", false
		}
		return 0, sceneID, "rollback", true
	}
	return 0, 0, "", false
}

// parseRequiredUint32 只接受大于零的十进制无符号整数。
func parseRequiredUint32(raw string) (uint32, bool) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 {
		return 0, false
	}
	return uint32(value), true
}
