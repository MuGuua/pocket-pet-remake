package reward

import (
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/wallet"
)

// Entry 描述一条可以走统一发奖服务的正式奖励。
// 当前先覆盖 quest、battle、后台补发第一阶段会共用的几种基础奖励。
type Entry struct {
	Type     string `json:"type"`
	Value    uint64 `json:"value"`
	ItemID   uint64 `json:"item_id"`
	ItemName string `json:"item_name"`
	Count    uint64 `json:"count"`
	PetID    uint64 `json:"pet_id"`
}

// GrantInput 描述一次统一发奖请求的完整归因信息。
// 所有正式发奖都要求携带原因类型和来源主键，避免后续审计无法追踪。
type GrantInput struct {
	PlayerID     uint64  `json:"player_id"`
	ReasonType   string  `json:"reason_type"`
	ReasonRefID  uint64  `json:"reason_ref_id"`
	OperatorType string  `json:"operator_type"`
	OperatorID   uint64  `json:"operator_id"`
	Rewards      []Entry `json:"rewards"`
}

// GrantResult 返回统一发奖已经正式生效后的聚合结果。
// Handler 只需要消费这里的权威结果做推送，不再自己重复判断奖励是否成功。
type GrantResult struct {
	Granted       []Entry          `json:"granted"`
	BagUpdated    bool             `json:"bag_updated"`
	GrantedPets   []pet.Pet        `json:"granted_pets"`
	Wallet        *wallet.Snapshot `json:"wallet,omitempty"`
	PlayerProfile *player.Profile  `json:"player_profile,omitempty"`
}
