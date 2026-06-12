package wallet

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrWalletNotFound 表示后台请求的玩家钱包不存在。
	ErrWalletNotFound = errors.New("wallet not found")
	// ErrInvalidAdminWalletInput 表示后台钱包调整请求缺少必要字段或金额非法。
	ErrInvalidAdminWalletInput = errors.New("invalid admin wallet input")
	// ErrInvalidRuntimeAdjustInput 表示运行时货币变更缺少归因信息或金额非法。
	ErrInvalidRuntimeAdjustInput = errors.New("invalid runtime wallet adjust input")
)

const (
	// CopperPerSilver 定义银币兑换成铜币时使用的固定进位比例。
	CopperPerSilver uint64 = 1000
	// CopperPerGold 定义金币兑换成铜币时使用的固定进位比例。
	CopperPerGold uint64 = CopperPerSilver * CopperPerSilver
)

// AdminListQuery 描述后台钱包列表的筛选与分页参数。
// 钱包页默认按玩家分页，方便运营快速按玩家定位资产问题。
type AdminListQuery struct {
	PlayerID uint64
	Keyword  string
	Page     uint32
	PageSize uint32
}

// Normalize 收口默认分页并清洗关键字。
func (q AdminListQuery) Normalize() AdminListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	return q
}

// Snapshot 统一返回钱包的总铜币与金银铜拆分结果。
type Snapshot struct {
	TotalCopper uint64 `json:"total_copper"`
	Gold        uint64 `json:"gold"`
	Silver      uint64 `json:"silver"`
	Copper      uint64 `json:"copper"`
}

// AdminWalletSummary 描述后台钱包列表页展示的摘要字段。
type AdminWalletSummary struct {
	PlayerID   uint64    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Wallet     Snapshot  `json:"wallet"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// AdminWalletList 是后台钱包分页响应。
type AdminWalletList struct {
	Items    []AdminWalletSummary `json:"items"`
	Total    uint64               `json:"total"`
	Page     uint32               `json:"page"`
	PageSize uint32               `json:"page_size"`
}

// AdminWalletDetail 是后台详情抽屉消费的钱包完整快照。
type AdminWalletDetail struct {
	PlayerID   uint64    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Wallet     Snapshot  `json:"wallet"`
	Version    uint64    `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AdminAdjustInput 描述后台手动加减货币时允许提交的字段。
// 这里使用增量调整而不是直接覆写总铜币，避免运营误把展示态当成数据库真值。
type AdminAdjustInput struct {
	ChangeTotalCopper int64  `json:"change_total_copper"`
	Reason            string `json:"reason"`
}

// Normalize 清洗后台说明字段，避免空白原因直接落库。
func (input AdminAdjustInput) Normalize() AdminAdjustInput {
	input.Reason = strings.TrimSpace(input.Reason)
	return input
}

// RuntimeAdjustInput 描述正式玩法链路里的一次钱包增减。
// 这里要求调用方显式带上原因与操作者信息，保证所有运行时货币变化都能追溯到具体玩法事件。
type RuntimeAdjustInput struct {
	ChangeTotalCopper int64  `json:"change_total_copper"`
	ReasonType        string `json:"reason_type"`
	ReasonRefID       uint64 `json:"reason_ref_id"`
	OperatorType      string `json:"operator_type"`
	OperatorID        uint64 `json:"operator_id"`
}

// Normalize 清洗运行时流水归因字段，避免空白字符串直接落库。
func (input RuntimeAdjustInput) Normalize() RuntimeAdjustInput {
	input.ReasonType = strings.TrimSpace(input.ReasonType)
	input.OperatorType = strings.TrimSpace(input.OperatorType)
	return input
}

// RuntimeAdjustResult 返回本次运行时调账后的最新钱包快照与归因信息。
// WebSocket handler 可以直接拿它构造钱包推送，避免重复查库拼装。
type RuntimeAdjustResult struct {
	Wallet      Snapshot `json:"wallet"`
	Version     uint64   `json:"version"`
	ReasonType  string   `json:"reason_type"`
	ReasonRefID uint64   `json:"reason_ref_id"`
}
