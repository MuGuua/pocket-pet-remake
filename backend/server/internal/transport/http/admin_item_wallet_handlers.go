package httptransport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/wallet"
)

// AdminItemHandler 负责后台物品模板列表、详情与 CRUD。
// 所有正式物品模板都应通过这里进入数据库，避免前端本地维护一套假配置。
type AdminItemHandler struct {
	adminService *admin.Service
	itemService  *item.Service
}

// AdminWalletHandler 负责后台玩家钱包查询与手动调整。
// 钱包调整使用总铜币增量作为唯一真值，避免运营误把展示态拆分金额当作数据库字段。
type AdminWalletHandler struct {
	adminService  *admin.Service
	walletService *wallet.Service
}

// AdminRewardHandler 负责后台统一补发入口。
// 后台补发物品、货币等正式奖励应优先走这里，而不是直接改背包格子或钱包真值。
type AdminRewardHandler struct {
	adminService  *admin.Service
	rewardService *reward.Service
	bagService    *bag.Service
}

type adminRewardGrantRequest struct {
	PlayerID uint64         `json:"player_id"`
	Reason   string         `json:"reason"`
	Rewards  []reward.Entry `json:"rewards"`
}

type adminRewardGrantResponse struct {
	PlayerID    uint64                        `json:"player_id"`
	Reason      string                        `json:"reason"`
	Granted     []reward.Entry                `json:"granted"`
	Wallet      *wallet.Snapshot              `json:"wallet,omitempty"`
	Bag         *bag.RuntimeContainerSnapshot `json:"bag,omitempty"`
	GrantedPets []pet.Pet                     `json:"granted_pets"`
}

func (h *AdminItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.itemService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin item service is unavailable", nil)
		return
	}
	permission := "items:view"
	if r.Method != http.MethodGet {
		permission = "items:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/items"))
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
	if err != nil {
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

func (h *AdminItemHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminItemListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.itemService.ListAdminItems(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin item list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminItemHandler) handleDetail(w http.ResponseWriter, r *http.Request, itemID uint64) {
	detail, err := h.itemService.GetAdminItemDetail(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, item.ErrItemDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "item definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin item detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminItemHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request item.AdminUpsertItemInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	created, err := h.itemService.CreateAdminItem(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, item.ErrInvalidAdminItemInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid item payload", nil)
		case errors.Is(err, item.ErrItemDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "item definition already exists", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "create admin item failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", created)
}

func (h *AdminItemHandler) handleUpdate(w http.ResponseWriter, r *http.Request, itemID uint64) {
	defer r.Body.Close()
	var request item.AdminUpsertItemInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.itemService.UpdateAdminItem(r.Context(), itemID, request)
	if err != nil {
		switch {
		case errors.Is(err, item.ErrInvalidAdminItemInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid item payload", nil)
		case errors.Is(err, item.ErrItemDefinitionConflict):
			writeJSON(w, http.StatusConflict, http.StatusConflict, "item definition already exists", nil)
		case errors.Is(err, item.ErrItemDefinitionNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "item definition not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "update admin item failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminItemHandler) handleDelete(w http.ResponseWriter, r *http.Request, itemID uint64) {
	if err := h.itemService.DeleteAdminItem(r.Context(), itemID); err != nil {
		if errors.Is(err, item.ErrItemDefinitionNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "item definition not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "delete admin item failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", map[string]any{"item_id": itemID, "deleted": true})
}

func (h *AdminWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.walletService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin wallet service is unavailable", nil)
		return
	}
	permission := "wallet:view"
	if r.Method != http.MethodGet {
		permission = "wallet:edit"
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, permission); !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/wallets"))
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}
	playerID, err := parseUintPathID(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid player_id", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleDetail(w, r, playerID)
	case http.MethodPut:
		h.handleAdjust(w, r, playerID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *AdminWalletHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query, err := parseAdminWalletListQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	result, err := h.walletService.ListAdminWallets(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin wallet list failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", result)
}

func (h *AdminWalletHandler) handleDetail(w http.ResponseWriter, r *http.Request, playerID uint64) {
	detail, err := h.walletService.GetAdminWalletDetail(r.Context(), playerID)
	if err != nil {
		if errors.Is(err, wallet.ErrWalletNotFound) {
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "wallet not found", nil)
			return
		}
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load admin wallet detail failed", nil)
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", detail)
}

func (h *AdminWalletHandler) handleAdjust(w http.ResponseWriter, r *http.Request, playerID uint64) {
	defer r.Body.Close()
	var request wallet.AdminAdjustInput
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.walletService.AdjustAdminWallet(r.Context(), playerID, request)
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrInvalidAdminWalletInput):
			writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "invalid wallet payload", nil)
		case errors.Is(err, wallet.ErrWalletNotFound):
			writeJSON(w, http.StatusNotFound, http.StatusNotFound, "wallet not found", nil)
		default:
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "adjust admin wallet failed", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", updated)
}

func (h *AdminRewardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.rewardService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin reward service is unavailable", nil)
		return
	}
	adminProfile, ok := authenticateAdminRequest(w, r, h.adminService, "bag:grant", "wallet:edit")
	if !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/admin/rewards"))
	if path != "" && path != "/" {
		writeJSON(w, http.StatusNotFound, http.StatusNotFound, "admin reward endpoint not found", nil)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	defer r.Body.Close()
	var request adminRewardGrantRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, err.Error(), nil)
		return
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if request.PlayerID == 0 || request.Reason == "" || len(request.Rewards) == 0 {
		writeJSON(w, http.StatusBadRequest, http.StatusBadRequest, "player_id, reason and rewards are required", nil)
		return
	}
	result, err := h.rewardService.GrantRuntimeRewards(r.Context(), reward.GrantInput{
		PlayerID:     request.PlayerID,
		ReasonType:   "admin_reward",
		ReasonRefID:  0,
		OperatorType: "admin",
		OperatorID:   adminProfile.AdminUserID,
		Rewards:      request.Rewards,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "grant admin rewards failed", nil)
		return
	}
	var bagSnapshot *bag.RuntimeContainerSnapshot
	if result.BagUpdated && h.bagService != nil {
		snapshot, snapshotErr := h.bagService.ListRuntimeContainer(r.Context(), request.PlayerID, bag.ContainerTypeBag)
		if snapshotErr != nil {
			writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load bag snapshot after admin reward failed", nil)
			return
		}
		bagSnapshot = snapshot
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", adminRewardGrantResponse{
		PlayerID:    request.PlayerID,
		Reason:      request.Reason,
		Granted:     result.Granted,
		Wallet:      result.Wallet,
		Bag:         bagSnapshot,
		GrantedPets: append([]pet.Pet{}, result.GrantedPets...),
	})
}

func parseAdminItemListQuery(r *http.Request) (item.AdminListQuery, error) {
	query := r.URL.Query()
	result := item.AdminListQuery{}
	if raw := strings.TrimSpace(query.Get("item_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return item.AdminListQuery{}, errors.New("item_id must be an unsigned integer")
		}
		result.ItemID = value
	}
	result.ItemType = strings.TrimSpace(query.Get("item_type"))
	result.Keyword = strings.TrimSpace(query.Get("keyword"))
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return item.AdminListQuery{}, errors.New("enabled must be a boolean")
		}
		result.Enabled = &value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return item.AdminListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}

func parseAdminWalletListQuery(r *http.Request) (wallet.AdminListQuery, error) {
	query := r.URL.Query()
	result := wallet.AdminListQuery{Keyword: strings.TrimSpace(query.Get("keyword"))}
	if raw := strings.TrimSpace(query.Get("player_id")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return wallet.AdminListQuery{}, errors.New("player_id must be an unsigned integer")
		}
		result.PlayerID = value
	}
	page, pageSize, err := parsePageParams(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return wallet.AdminListQuery{}, err
	}
	result.Page = page
	result.PageSize = pageSize
	return result.Normalize(), nil
}

// parsePageParams 统一解析后台分页参数，避免每个列表页重复拷贝同样的 page/page_size 校验代码。
func parsePageParams(rawPage, rawPageSize string) (uint32, uint32, error) {
	var (
		page     uint32
		pageSize uint32
	)
	if raw := strings.TrimSpace(rawPage); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return 0, 0, errors.New("page must be an unsigned integer")
		}
		page = uint32(value)
	}
	if raw := strings.TrimSpace(rawPageSize); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return 0, 0, errors.New("page_size must be an unsigned integer")
		}
		pageSize = uint32(value)
	}
	return page, pageSize, nil
}
