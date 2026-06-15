package admin

import "time"

// DashboardOverview 是后台控制台首页展示的业务概览快照。
type DashboardOverview struct {
	StatDate            string    `json:"stat_date"`
	Timezone            string    `json:"timezone"`
	GeneratedAt         time.Time `json:"generated_at"`
	OnlinePlayerCount   uint32    `json:"online_player_count"`
	DailyActiveAccounts uint64    `json:"daily_active_accounts"`
	NewAccountsToday    uint64    `json:"new_accounts_today"`
	TotalAccounts       uint64    `json:"total_accounts"`
	TotalPlayers        uint64    `json:"total_players"`
}
