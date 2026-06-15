package httptransport

import (
	"net/http"
	"time"

	"pocket-pet-remake/server/internal/module/admin"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
)

var dashboardStatLocation = time.FixedZone("Asia/Shanghai", 8*3600)

type AdminDashboardHandler struct {
	adminService   *admin.Service
	authService    *auth.Service
	playerService  *player.Service
	sessionService *session.Service
}

func (h *AdminDashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.adminService == nil || h.authService == nil || h.playerService == nil || h.sessionService == nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "admin dashboard service is unavailable", nil)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	if _, ok := authenticateAdminRequest(w, r, h.adminService, "dashboard:view"); !ok {
		return
	}

	now := time.Now().In(dashboardStatLocation)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, dashboardStatLocation)
	dayEnd := dayStart.Add(24 * time.Hour)

	accountMetrics, err := h.authService.GetDashboardAccountMetrics(r.Context(), dayStart, dayEnd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load dashboard account metrics failed", nil)
		return
	}
	totalPlayers, err := h.playerService.CountActivePlayers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, http.StatusInternalServerError, "load dashboard player metrics failed", nil)
		return
	}

	overview := admin.DashboardOverview{
		StatDate:            dayStart.Format("2006-01-02"),
		Timezone:            dashboardStatLocation.String(),
		GeneratedAt:         time.Now().UTC(),
		OnlinePlayerCount:   h.sessionService.CountOnlinePlayers(),
		DailyActiveAccounts: accountMetrics.DailyActiveAccounts,
		NewAccountsToday:    accountMetrics.NewAccountsToday,
		TotalAccounts:       accountMetrics.TotalAccounts,
		TotalPlayers:        totalPlayers,
	}
	writeJSON(w, http.StatusOK, http.StatusOK, "success", overview)
}
