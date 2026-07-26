package protocol

type WsAuthReq struct {
	WSToken       string `json:"ws_token"`
	ClientVersion string `json:"client_version"`
	DeviceID      string `json:"device_id"`
}

type WsAuthResp struct {
	PlayerID       uint64 `json:"player_id"`
	SessionID      string `json:"session_id"`
	ReconnectToken string `json:"reconnect_token"`
	HeartbeatSec   uint32 `json:"heartbeat_sec"`
	ServerTimeMS   int64  `json:"server_time_ms"`
}

type HeartbeatReq struct {
	ClientTimeMS int64 `json:"client_time_ms"`
}

type HeartbeatResp struct {
	ServerTimeMS int64 `json:"server_time_ms"`
}

type ReconnectReq struct {
	ReconnectToken string `json:"reconnect_token"`
	BattleID       uint64 `json:"battle_id"`
	LastFrame      uint32 `json:"last_frame"`
}

type ReconnectResp struct {
	PlayerID           uint64            `json:"player_id"`
	SessionID          string            `json:"session_id"`
	ReconnectToken     string            `json:"reconnect_token"`
	HeartbeatSec       uint32            `json:"heartbeat_sec"`
	ServerTimeMS       int64             `json:"server_time_ms"`
	World              *EnterWorldResp   `json:"world,omitempty"`
	BattleStart        *BattleStartPush  `json:"battle_start,omitempty"`
	BattleState        *BattleStatePush  `json:"battle_state,omitempty"`
	BattleResult       *BattleResultPush `json:"battle_result,omitempty"`
	BattleReplayStates []BattleStatePush `json:"battle_replay_states,omitempty"`
	ActiveDialogue     *ActiveDialogue   `json:"active_dialogue,omitempty"`
}

type ForceOfflinePush struct {
	Reason string `json:"reason"`
}

type ErrorPush struct {
	Code uint32 `json:"code"`
	Msg  string `json:"msg"`
}

type Vec2i struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type PlayerBrief struct {
	PlayerID uint64 `json:"player_id"`
	Name     string `json:"name"`
	Level    uint32 `json:"level"`
}

type PlayerSnapshot struct {
	PlayerID           uint64                       `json:"player_id"`
	Name               string                       `json:"name"`
	Level              uint32                       `json:"level"`
	Exp                uint64                       `json:"exp"`
	ExpToNext          uint64                       `json:"exp_to_next"`
	FreeAttrPoints     uint32                       `json:"free_attr_points"`
	Strength           uint32                       `json:"strength"`
	Vitality           uint32                       `json:"vitality"`
	Agility            uint32                       `json:"agility"`
	Mind               uint32                       `json:"mind"`
	Gold               uint32                       `json:"gold"`
	HP                 uint32                       `json:"hp"`
	HPMax              uint32                       `json:"hp_max"`
	Vigor              uint32                       `json:"vigor"`
	VigorMax           uint32                       `json:"vigor_max"`
	Spirit             uint32                       `json:"spirit"`
	SpiritMax          uint32                       `json:"spirit_max"`
	ATK                uint32                       `json:"atk"`
	DEF                uint32                       `json:"def"`
	SPD                uint32                       `json:"spd"`
	MANA               uint32                       `json:"mana"`
	HitPct             uint32                       `json:"hit_pct"`
	DodgePct           uint32                       `json:"dodge_pct"`
	CritRatePct        uint32                       `json:"crit_rate_pct"`
	CritDmgPct         uint32                       `json:"crit_dmg_pct"`
	PhysicalResistPct  uint32                       `json:"physical_resist_pct"`
	SkillResistPct     uint32                       `json:"skill_resist_pct"`
	ConfusionResistPct uint32                       `json:"confusion_resist_pct"`
	SleepResistPct     uint32                       `json:"sleep_resist_pct"`
	ParalysisResistPct uint32                       `json:"paralysis_resist_pct"`
	SealResistPct      uint32                       `json:"seal_resist_pct"`
	CurseResistPct     uint32                       `json:"curse_resist_pct"`
	CritResistPct      uint32                       `json:"crit_resist_pct"`
	CritDmgResistPct   uint32                       `json:"crit_dmg_resist_pct"`
	CharacterResistPct uint32                       `json:"character_resist_pct"`
	PetResistPct       uint32                       `json:"pet_resist_pct"`
	MercenaryResistPct uint32                       `json:"mercenary_resist_pct"`
	GenericShieldPct   uint32                       `json:"generic_shield_pct"`
	SkillIDs           []uint32                     `json:"skill_ids"`
	SkinID             string                       `json:"skin_id"`
	EquippedItems      []PlayerEquippedItemSnapshot `json:"equipped_items,omitempty"`
}

type EntityBrief struct {
	EntityID uint64 `json:"entity_id"`
	// PlayerID is only populated when the entity represents a real player in the
	// world snapshot so clients can safely target PVP actions.
	PlayerID   uint64 `json:"player_id"`
	EntityType uint32 `json:"entity_type"`
	Pos        Vec2i  `json:"pos"`
	Dir        uint32 `json:"dir"`
	Speed      uint32 `json:"speed"`
	Name       string `json:"name"`
	Level      uint32 `json:"level,omitempty"`
	Exp        uint64 `json:"exp,omitempty"`
	HP         uint32 `json:"hp,omitempty"`
	HPMax      uint32 `json:"hp_max,omitempty"`
	Vigor      uint32 `json:"vigor,omitempty"`
	VigorMax   uint32 `json:"vigor_max,omitempty"`
	Spirit     uint32 `json:"spirit,omitempty"`
	SpiritMax  uint32 `json:"spirit_max,omitempty"`
	SkinID     string `json:"skin_id,omitempty"`
	// FollowingPet 仅用于玩家实体，表示当前编队首只宠物的权威世界展示摘要。
	FollowingPet *PetBrief `json:"following_pet,omitempty"`
}

type PetBrief struct {
	PetUID    uint64 `json:"pet_uid"`
	PetID     uint32 `json:"pet_id"`
	Name      string `json:"name,omitempty"`
	Exp       uint64 `json:"exp,omitempty"`
	Level     uint32 `json:"level"`
	HP        uint32 `json:"hp"`
	HPMax     uint32 `json:"hp_max"`
	Spirit    uint32 `json:"spirit,omitempty"`
	SpiritMax uint32 `json:"spirit_max,omitempty"`
	SkinID    string `json:"skin_id"`
}

type PetGrowthAptitudes struct {
	HPApt   uint32 `json:"hp_apt"`
	ATKApt  uint32 `json:"atk_apt"`
	DEFApt  uint32 `json:"def_apt"`
	SPDApt  uint32 `json:"spd_apt"`
	MANAApt uint32 `json:"mana_apt"`
}

// PetSkillSlotEntry 单个技能槽位。
type PetSkillSlotEntry struct {
	SlotIndex     uint32 `json:"slot_index"`
	SkillID       uint32 `json:"skill_id"`
	Enabled       bool   `json:"enabled,omitempty"`
	SkillName     string `json:"skill_name,omitempty"`
	Description   string `json:"description,omitempty"`
	SkillVisualID string `json:"skill_visual_id,omitempty"`
	SkillQuality  string `json:"skill_quality,omitempty"`
}

// PetSkillSlots 宠物分分类技能槽；artifact 仅在查看技能详情时填充 skill_id。
type PetSkillSlots struct {
	Innate         []PetSkillSlotEntry `json:"innate"`
	ActiveTalisman PetSkillSlotEntry   `json:"active_talisman"`
	TalismanHero   PetSkillSlotEntry   `json:"talisman_hero"`
	Talisman1      PetSkillSlotEntry   `json:"talisman_1"`
	Talisman2      PetSkillSlotEntry   `json:"talisman_2"`
	Talisman3      PetSkillSlotEntry   `json:"talisman_3"`
	Normal         []PetSkillSlotEntry `json:"normal"`
	Artifact       []PetSkillSlotEntry `json:"artifact"`
}

type PetDetail struct {
	PetUID uint64 `json:"pet_uid"`
	PetID  uint32 `json:"pet_id"`
	// PetName 是系统宠物名称；没有 custom_name 时客户端展示该字段。
	PetName string `json:"pet_name,omitempty"`
	// CustomName 是单只宠物的自定义名称，服务端只在真实设置后下发非空值。
	CustomName string `json:"custom_name,omitempty"`
	// Name 是服务端计算后的最终展示名，保留给旧客户端直接读取。
	Name string `json:"name,omitempty"`
	// SkinID 是服务端权威外观资源 ID，客户端按它加载宠物待机首帧。
	SkinID   string   `json:"skin_id,omitempty"`
	Level    uint32   `json:"level"`
	Exp      uint64   `json:"exp"`
	Quality  uint32   `json:"quality"`
	HP       uint32   `json:"hp"`
	HPMax    uint32   `json:"hp_max"`
	ATK      uint32   `json:"atk"`
	DEF      uint32   `json:"def"`
	SPD      uint32   `json:"spd"`
	MANA     uint32   `json:"mana,omitempty"`
	SkillIDs []uint32 `json:"skill_ids"`
	// SkillSlots 结构化技能槽；列表接口中 artifact 槽 skill_id 为 0。
	SkillSlots *PetSkillSlots `json:"skill_slots,omitempty"`
	InLineup   bool           `json:"in_lineup"`
	// IsUsable 表示该实例对应的 pet_id 是否存在于启用中的系统宠物模板列表。
	IsUsable                 bool               `json:"is_usable"`
	ExpToNext                uint64             `json:"exp_to_next,omitempty"`
	FreeAttrPoints           uint32             `json:"free_attr_points,omitempty"`
	AllocHPPoints            uint32             `json:"alloc_hp_points,omitempty"`
	AllocATKPoints           uint32             `json:"alloc_atk_points,omitempty"`
	AllocSPDPoints           uint32             `json:"alloc_spd_points,omitempty"`
	AllocMANAPoints          uint32             `json:"alloc_mana_points,omitempty"`
	AllocDEFPoints           uint32             `json:"alloc_def_points,omitempty"`
	BaseHPApt                uint32             `json:"base_hp_apt,omitempty"`
	BaseATKApt               uint32             `json:"base_atk_apt,omitempty"`
	BaseDEFApt               uint32             `json:"base_def_apt,omitempty"`
	BaseSPDApt               uint32             `json:"base_spd_apt,omitempty"`
	BaseMANAApt              uint32             `json:"base_mana_apt,omitempty"`
	ExtraHPApt               uint32             `json:"extra_hp_apt,omitempty"`
	ExtraATKApt              uint32             `json:"extra_atk_apt,omitempty"`
	ExtraDEFApt              uint32             `json:"extra_def_apt,omitempty"`
	ExtraSPDApt              uint32             `json:"extra_spd_apt,omitempty"`
	ExtraMANAApt             uint32             `json:"extra_mana_apt,omitempty"`
	GrowthAptitudes          PetGrowthAptitudes `json:"growth_aptitudes,omitempty"`
	AutoHPPoints             uint32             `json:"auto_hp_points,omitempty"`
	AutoATKPoints            uint32             `json:"auto_atk_points,omitempty"`
	AutoSPDPoints            uint32             `json:"auto_spd_points,omitempty"`
	AutoMANAPoints           uint32             `json:"auto_mana_points,omitempty"`
	AutoDEFPoints            uint32             `json:"auto_def_points,omitempty"`
	Spirit                   uint32             `json:"spirit,omitempty"`
	SpiritMax                uint32             `json:"spirit_max,omitempty"`
	HitPct                   uint32             `json:"hit_pct,omitempty"`
	DodgePct                 uint32             `json:"dodge_pct,omitempty"`
	CritRatePct              uint32             `json:"crit_rate_pct,omitempty"`
	CritDmgPct               uint32             `json:"crit_dmg_pct,omitempty"`
	PhysicalResistPct        uint32             `json:"physical_resist_pct,omitempty"`
	ReversePhysicalResistPct uint32             `json:"reverse_physical_resist_pct,omitempty"`
	SkillResistPct           uint32             `json:"skill_resist_pct,omitempty"`
	ReverseSkillResistPct    uint32             `json:"reverse_skill_resist_pct,omitempty"`
	ConfusionResistPct       uint32             `json:"confusion_resist_pct,omitempty"`
	SleepResistPct           uint32             `json:"sleep_resist_pct,omitempty"`
	ParalysisResistPct       uint32             `json:"paralysis_resist_pct,omitempty"`
	SealResistPct            uint32             `json:"seal_resist_pct,omitempty"`
	CurseResistPct           uint32             `json:"curse_resist_pct,omitempty"`
	CritDmgResistPct         uint32             `json:"crit_dmg_resist_pct,omitempty"`
	CritResistPct            uint32             `json:"crit_resist_pct,omitempty"`
	CharacterResistPct       uint32             `json:"character_resist_pct,omitempty"`
	PetResistPct             uint32             `json:"pet_resist_pct,omitempty"`
}

type PetAllocateAttrReq struct {
	PetUID uint64 `json:"pet_uid"`
	HP     uint32 `json:"hp"`
	ATK    uint32 `json:"atk"`
	SPD    uint32 `json:"spd"`
	MANA   uint32 `json:"mana"`
	DEF    uint32 `json:"def"`
}

type PetAllocateAttrResp struct {
	Pet PetDetail `json:"pet"`
}

// PetListReq 拉取宠物列表摘要；完整属性需再按 pet_uid 请求 PetSkillDetailReq。
type PetListReq struct{}

type PetListResp struct {
	Pets   []PetDetail `json:"pets"`
	Lineup []PetBrief  `json:"lineup"`
}

type PetUpdatePush struct {
	Pet PetDetail `json:"pet"`
}

type PetLineupSetReq struct {
	OpID    uint32   `json:"op_id"`
	PetUIDs []uint64 `json:"pet_uids"`
}

type PetLineupSetResp struct {
	Accepted bool       `json:"accepted"`
	Lineup   []PetBrief `json:"lineup"`
	Reason   string     `json:"reason"`
}

// PetArtifactEquipReq 从背包装备法宝到宠物指定槽位。
type PetArtifactEquipReq struct {
	PetUID        uint64 `json:"pet_uid"`
	SlotIndex     uint32 `json:"slot_index"`
	ContainerType string `json:"container_type"`
	BagSlotIndex  uint32 `json:"bag_slot_index"`
}

type PetArtifactEquipResp struct {
	Pet PetDetail `json:"pet"`
}

// PetArtifactUnequipReq 卸下宠物法宝槽技能。
type PetArtifactUnequipReq struct {
	PetUID    uint64 `json:"pet_uid"`
	SlotIndex uint32 `json:"slot_index"`
}

type PetArtifactUnequipResp struct {
	Pet PetDetail `json:"pet"`
}

// PetSkillDetailReq 拉取单宠完整属性和技能分槽（含法宝技）。
type PetSkillDetailReq struct {
	PetUID uint64 `json:"pet_uid"`
}

type PetSkillDetailResp struct {
	Pet PetDetail `json:"pet"`
}

type EnterWorldReq struct{}

type EnterWorldResp struct {
	Self           PlayerBrief         `json:"self"`
	Player         PlayerSnapshot      `json:"player"`
	SceneID        uint32              `json:"scene_id"`
	SelfPos        Vec2i               `json:"self_pos"`
	SceneVersion   uint32              `json:"scene_version"`
	NearbyEntities []EntityBrief       `json:"nearby_entities"`
	Lineup         []PetBrief          `json:"lineup"`
	Gold           uint32              `json:"gold"`
	WildEncounter  WildEncounterConfig `json:"wild_encounter"`
}

type MoveIntentReq struct {
	OpID          uint32 `json:"op_id"`
	MoveSeq       uint32 `json:"move_seq"`
	SceneID       uint32 `json:"scene_id"`
	TargetSceneID uint32 `json:"target_scene_id"`
	PortalID      uint32 `json:"portal_id"`
	// MapTeleport 表示玩家从世界地图发起快速传送；服务端会忽略客户端落点并读取数据库中的地图中心坐标。
	MapTeleport bool `json:"map_teleport,omitempty"`
	// TargetPos 仅用于同场景移动上报；保留为可选字段以兼容只负责切图的旧客户端。
	TargetPos *Vec2i `json:"target_pos,omitempty"`
	// PrecisePos 使用千分之一场景格的定点整数，仅用于同场景实时表现；持久化仍使用 TargetPos。
	PrecisePos *Vec2i `json:"precise_pos,omitempty"`
	// Facing 是客户端当前四方向朝向，服务端归一化后再广播，避免观察端根据延迟坐标猜测朝向。
	Facing *Vec2i `json:"facing,omitempty"`
	// Moving 表示客户端当前是否仍在移动；指针用于区分旧客户端未发送字段与明确停止。
	Moving *bool `json:"moving,omitempty"`
}

type MoveIntentResp struct {
	Accepted     bool   `json:"accepted"`
	MoveSeq      uint32 `json:"move_seq"`
	SceneID      uint32 `json:"scene_id"`
	CorrectedPos Vec2i  `json:"corrected_pos"`
	Reason       string `json:"reason"`
}

// SceneTriggerPush 通知客户端播放服务端判定的一次性场景剧情。
// 客户端播放完成后必须发送 SceneTriggerAckReq，让服务端落库 flag、解锁 NPC 和接取任务。
type SceneTriggerPush struct {
	TriggerCode        string `json:"trigger_code"`
	SceneID            uint32 `json:"scene_id"`
	ClientAnimationKey string `json:"client_animation_key"`
	PromptText         string `json:"prompt_text"`
	BlockMovement      bool   `json:"block_movement"`
}

// SceneTriggerAckReq 表示客户端已完成指定场景剧情播放。
type SceneTriggerAckReq struct {
	TriggerCode string `json:"trigger_code"`
}

// SceneTriggerAckResp 返回服务端是否接受本次剧情完成确认。
type SceneTriggerAckResp struct {
	Accepted    bool   `json:"accepted"`
	Reason      string `json:"reason"`
	TriggerCode string `json:"trigger_code"`
}

type InteractReq struct {
	EntityID uint64 `json:"entity_id"`
	SelfPos  *Vec2i `json:"self_pos,omitempty"`
}

type NpcMenuEntry struct {
	EntryID    string `json:"entry_id"`
	EntryType  string `json:"entry_type"`
	QuestID    uint64 `json:"quest_id"`
	QuestState string `json:"quest_state"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	State      string `json:"state"`
	Priority   uint32 `json:"priority"`
}

type InteractResp struct {
	Accepted     bool           `json:"accepted"`
	Reason       string         `json:"reason"`
	ResponseType string         `json:"response_type"`
	EntityID     uint64         `json:"entity_id"`
	NPCName      string         `json:"npc_name"`
	MenuEntries  []NpcMenuEntry `json:"menu_entries"`
}

type NPCMenuReq struct {
	EntityID uint64 `json:"entity_id"`
}

type NPCMenuResp struct {
	Accepted    bool           `json:"accepted"`
	Reason      string         `json:"reason"`
	EntityID    uint64         `json:"entity_id"`
	NPCName     string         `json:"npc_name"`
	MenuEntries []NpcMenuEntry `json:"menu_entries"`
}

type NPCActionReq struct {
	EntityID uint64 `json:"entity_id"`
	EntryID  string `json:"entry_id"`
	SelfPos  *Vec2i `json:"self_pos,omitempty"`
}

type NPCActionResp struct {
	Accepted           bool             `json:"accepted"`
	Reason             string           `json:"reason"`
	EntityID           uint64           `json:"entity_id"`
	EntryID            string           `json:"entry_id"`
	ResultType         string           `json:"result_type"`
	Notice             string           `json:"notice"`
	NPCName            string           `json:"npc_name"`
	MenuEntries        []NpcMenuEntry   `json:"menu_entries"`
	Dialogue           *NPCDialogueNode `json:"dialogue"`
	Shop               *NPCShopPayload  `json:"shop,omitempty"`
	ClientAnimationKey string           `json:"client_animation_key,omitempty"`
}

type NPCShopGood struct {
	ItemID      uint64 `json:"item_id"`
	ItemName    string `json:"item_name"`
	PriceCopper uint64 `json:"price_copper"`
}

type NPCShopPayload struct {
	Goods  []NPCShopGood  `json:"goods"`
	Wallet WalletSnapshot `json:"wallet"`
}

type ActiveDialogue struct {
	EntityID uint64           `json:"entity_id"`
	NPCName  string           `json:"npc_name"`
	Node     *NPCDialogueNode `json:"node"`
}

type NPCDialogueOption struct {
	OptionID string `json:"option_id"`
	Text     string `json:"text"`
	Format   string `json:"format"`
}

type NPCDialogueNode struct {
	DialogueID           int64               `json:"dialogue_id"`
	NodeID               string              `json:"node_id"`
	NodeType             string              `json:"node_type"`
	Speaker              string              `json:"speaker"`
	IsPlayerSpeaker      bool                `json:"is_player_speaker"`
	Content              string              `json:"content"`
	ContentFormat        string              `json:"content_format"`
	PortraitKey          string              `json:"portrait_key"`
	ClientAnimationKey   string              `json:"client_animation_key"`
	ClientAnimationBlock bool                `json:"client_animation_block"`
	Options              []NPCDialogueOption `json:"options"`
	MentionedItems       []NPCDialogueItem   `json:"mentioned_items"`
	IsEnd                bool                `json:"is_end"`
	EffectNotice         string              `json:"effect_notice"`
}

type NPCDialogueItem struct {
	ItemID   uint64 `json:"item_id"`
	ItemName string `json:"item_name"`
	Icon     string `json:"icon,omitempty"`
}

// ItemDescriptionMention 描述文案中通过 {item:ID} / {pet:ID} 引入的其他模板。
type ItemDescriptionMention struct {
	ItemID   uint64 `json:"item_id,omitempty"`
	PetID    uint64 `json:"pet_id,omitempty"`
	ItemName string `json:"item_name"`
}

type NPCDialogueNextReq struct {
	EntityID   uint64 `json:"entity_id"`
	DialogueID int64  `json:"dialogue_id"`
	NodeID     string `json:"node_id"`
}

type NPCDialogueChooseReq struct {
	EntityID   uint64 `json:"entity_id"`
	DialogueID int64  `json:"dialogue_id"`
	NodeID     string `json:"node_id"`
	OptionID   string `json:"option_id"`
}

type NPCDialogueResp struct {
	Accepted bool             `json:"accepted"`
	Reason   string           `json:"reason"`
	EntityID uint64           `json:"entity_id"`
	Node     *NPCDialogueNode `json:"node"`
}

type PVPChallengeReq struct {
	OpID           uint32 `json:"op_id"`
	TargetPlayerID uint64 `json:"target_player_id"`
}

type PVPChallengeResp struct {
	Accepted       bool   `json:"accepted"`
	Reason         string `json:"reason"`
	ChallengeID    uint64 `json:"challenge_id"`
	TargetPlayerID uint64 `json:"target_player_id"`
}

type PVPChallengePush struct {
	ChallengeID uint64      `json:"challenge_id"`
	Challenger  PlayerBrief `json:"challenger"`
	ExpiresAtMS int64       `json:"expires_at_ms"`
}

type PVPChallengeReplyReq struct {
	ChallengeID uint64 `json:"challenge_id"`
	Accept      bool   `json:"accept"`
}

type PVPChallengeReplyResp struct {
	Accepted    bool   `json:"accepted"`
	Reason      string `json:"reason"`
	ChallengeID uint64 `json:"challenge_id"`
}

type EntityMovePush struct {
	SceneID      uint32 `json:"scene_id"`
	SceneVersion uint32 `json:"scene_version"`
	EntityID     uint64 `json:"entity_id"`
	MoveSeq      uint32 `json:"move_seq"`
	FromPos      Vec2i  `json:"from_pos"`
	ToPos        Vec2i  `json:"to_pos"`
	// PrecisePos 是服务端限制在 ToPos 半格范围内的千分之一格表现坐标。
	PrecisePos Vec2i `json:"precise_pos"`
	// Facing 是服务端归一化后的四方向朝向。
	Facing Vec2i `json:"facing"`
	// Moving 决定观察端播放行走还是待机动画。
	Moving bool   `json:"moving"`
	Speed  uint32 `json:"speed"`
}

// EntityEnterPush 通知同场景客户端创建一个新进入视野的实体。
type EntityEnterPush struct {
	SceneID uint32      `json:"scene_id"`
	Entity  EntityBrief `json:"entity"`
}

// EntityLeavePush 通知同场景客户端移除已经离线或切图的实体。
type EntityLeavePush struct {
	SceneID  uint32 `json:"scene_id"`
	EntityID uint64 `json:"entity_id"`
}

type WorldResyncPush struct {
	SceneID        uint32              `json:"scene_id"`
	SelfPos        Vec2i               `json:"self_pos"`
	SceneVersion   uint32              `json:"scene_version"`
	NearbyEntities []EntityBrief       `json:"nearby_entities"`
	WildEncounter  WildEncounterConfig `json:"wild_encounter"`
}

// WildEncounterConfig 描述当前地图暗雷遭遇参数，由服务端在进图/切图时下发，客户端本地按步判定。
type WildEncounterConfig struct {
	Enabled         bool     `json:"enabled"`
	SceneID         uint32   `json:"scene_id"`
	EncounterRate   uint32   `json:"encounter_rate"`
	SpawnMonsterIDs []uint32 `json:"spawn_monster_ids"`
}

type WildEncounterReq struct {
	SceneID uint32 `json:"scene_id"`
	MoveSeq uint32 `json:"move_seq"`
	SelfPos *Vec2i `json:"self_pos,omitempty"`
}

type WildEncounterResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
}

type PlayerAllocateAttrReq struct {
	Strength uint32 `json:"strength"`
	Vitality uint32 `json:"vitality"`
	Agility  uint32 `json:"agility"`
	Mind     uint32 `json:"mind"`
}

type PlayerAllocateAttrResp struct {
	Player PlayerSnapshot `json:"player"`
}

// PlayerProfileReq 请求当前会话玩家的人物属性，不携带客户端可修改字段。
type PlayerProfileReq struct{}

// PlayerProfileResp 只返回人物面板需要的服务端权威属性，不包含背包、宠物、地图或任务数据。
type PlayerProfileResp struct {
	Player PlayerSnapshot `json:"player"`
}

type PlayerEquipmentBonusSnapshot struct {
	HPMax                    uint32 `json:"hp_max"`
	MANA                     uint32 `json:"mana"`
	ATK                      uint32 `json:"atk"`
	DEF                      uint32 `json:"def"`
	SPD                      uint32 `json:"spd"`
	Spirit                   uint32 `json:"spirit"`
	SpiritMax                uint32 `json:"spirit_max"`
	HitPct                   uint32 `json:"hit_pct"`
	DodgePct                 uint32 `json:"dodge_pct"`
	CritRatePct              uint32 `json:"crit_rate_pct"`
	CritDmgPct               uint32 `json:"crit_dmg_pct"`
	PhysicalResistPct        uint32 `json:"physical_resist_pct"`
	ReversePhysicalResistPct uint32 `json:"reverse_physical_resist_pct"`
	SkillResistPct           uint32 `json:"skill_resist_pct"`
	ReverseSkillResistPct    uint32 `json:"reverse_skill_resist_pct"`
	ConfusionResistPct       uint32 `json:"confusion_resist_pct"`
	SleepResistPct           uint32 `json:"sleep_resist_pct"`
	ParalysisResistPct       uint32 `json:"paralysis_resist_pct"`
	SealResistPct            uint32 `json:"seal_resist_pct"`
	CurseResistPct           uint32 `json:"curse_resist_pct"`
	CritDmgResistPct         uint32 `json:"crit_dmg_resist_pct"`
	CritResistPct            uint32 `json:"crit_resist_pct"`
	CharacterResistPct       uint32 `json:"character_resist_pct"`
	PetResistPct             uint32 `json:"pet_resist_pct"`
}

type PlayerEquippedItemSnapshot struct {
	EquipSlot           string                       `json:"equip_slot"`
	EquipSlotLabel      string                       `json:"equip_slot_label"`
	ItemUID             string                       `json:"item_uid"`
	ItemID              uint64                       `json:"item_id"`
	ItemName            string                       `json:"item_name"`
	Icon                string                       `json:"icon,omitempty"`
	RequiredLevel       uint32                       `json:"required_level"`
	EnhanceLevel        uint32                       `json:"enhance_level"`
	IsDamaged           bool                         `json:"is_damaged,omitempty"`
	AppearanceSkinID    string                       `json:"appearance_skin_id,omitempty"`
	AppearanceOnly      bool                         `json:"appearance_only"`
	Description         string                       `json:"description,omitempty"`
	DescriptionMentions []ItemDescriptionMention     `json:"description_mentions,omitempty"`
	Bonus               PlayerEquipmentBonusSnapshot `json:"bonus"`
}

type PlayerEquipmentListReq struct{}

type PlayerEquipmentListResp struct {
	Items []PlayerEquippedItemSnapshot `json:"items"`
}

type PlayerEquipReq struct {
	ContainerType string `json:"container_type"`
	BagSlotIndex  uint32 `json:"bag_slot_index"`
}

type PlayerEquipResp struct {
	Equipped    PlayerEquippedItemSnapshot   `json:"equipped"`
	Unequipped  *PlayerEquippedItemSnapshot  `json:"unequipped,omitempty"`
	AllEquipped []PlayerEquippedItemSnapshot `json:"all_equipped"`
	Player      PlayerSnapshot               `json:"player"`
}

type PlayerUnequipReq struct {
	EquipSlot     string `json:"equip_slot"`
	ContainerType string `json:"container_type"`
}

type PlayerUnequipResp struct {
	Unequipped  PlayerEquippedItemSnapshot   `json:"unequipped"`
	AllEquipped []PlayerEquippedItemSnapshot `json:"all_equipped"`
	Player      PlayerSnapshot               `json:"player"`
}

type PlayerEquipmentEnhanceReq struct {
	ItemUID    string `json:"item_uid"`
	CostItemID uint64 `json:"cost_item_id,omitempty"`
}

type PlayerEquipmentEnhanceResp struct {
	Success        bool                         `json:"success"`
	OldLevel       uint32                       `json:"old_level"`
	NewLevel       uint32                       `json:"new_level"`
	RatePct        uint32                       `json:"rate_pct"`
	RollPct        uint32                       `json:"roll_pct"`
	FailurePenalty string                       `json:"failure_penalty,omitempty"`
	Item           PlayerEquippedItemSnapshot   `json:"item"`
	AllEquipped    []PlayerEquippedItemSnapshot `json:"all_equipped"`
	Player         *PlayerSnapshot              `json:"player,omitempty"`
}

type PlayerEquipmentRepairReq struct {
	ItemUID string `json:"item_uid"`
}

type PlayerEquipmentRepairResp struct {
	Item        PlayerEquippedItemSnapshot   `json:"item"`
	AllEquipped []PlayerEquippedItemSnapshot `json:"all_equipped"`
}

type BattleActorSnapshot struct {
	ActorID       uint64                `json:"actor_id"`
	ActorType     uint32                `json:"actor_type"`
	UnitClass     uint32                `json:"unit_class"`
	OwnerPlayerID uint64                `json:"owner_player_id"`
	PetUID        uint64                `json:"pet_uid"`
	PetID         uint32                `json:"pet_id"`
	Name          string                `json:"name"`
	SkinID        string                `json:"skin_id"`
	HP            uint32                `json:"hp"`
	HPMax         uint32                `json:"hp_max"`
	ATK           uint32                `json:"atk"`
	DEF           uint32                `json:"def"`
	SPD           uint32                `json:"spd"`
	Skills        []BattleSkillSnapshot `json:"skills"`
	SkillIDs      []uint32              `json:"skill_ids"`
	StatusIDs     []uint32              `json:"status_ids"`
	LineupIndex   uint32                `json:"lineup_index"`
}

type BattleSkillSnapshot struct {
	SkillID       uint32 `json:"skill_id"`
	Name          string `json:"name"`
	TargetType    string `json:"target_type"`
	TargetCount   uint32 `json:"target_count"`
	AnimationKey  string `json:"animation_key"`
	SkillVisualID string `json:"skill_visual_id"`
	CastColor     string `json:"cast_color"`
	ImpactColor   string `json:"impact_color"`
	Projectile    bool   `json:"projectile"`
	IsBasicAttack bool   `json:"is_basic_attack"`
	Level         uint32 `json:"level,omitempty"`
}

type BattleStartPush struct {
	BattleID             uint64                `json:"battle_id"`
	BattleType           uint32                `json:"battle_type"`
	BattleVersion        uint32                `json:"battle_version"`
	Frame                uint32                `json:"frame"`
	ParticipantPlayerIDs []uint64              `json:"participant_player_ids"`
	Allies               []BattleActorSnapshot `json:"allies"`
	Enemies              []BattleActorSnapshot `json:"enemies"`
	Round                uint32                `json:"round"`
	Phase                string                `json:"phase"`
	ActiveActorID        uint64                `json:"active_actor_id"`
	ActivePetUID         uint64                `json:"active_pet_uid"`
	CommandDeadlineMS    int64                 `json:"command_deadline_ms"`
	AutoBattleEnabled    bool                  `json:"auto_battle_enabled"`
	PendingActorIDs      []uint64              `json:"pending_actor_ids"`
	ControllableActorIDs []uint64              `json:"controllable_actor_ids"`
}

type BattleActionReq struct {
	OpID              uint32 `json:"op_id"`
	BattleID          uint64 `json:"battle_id"`
	Round             uint32 `json:"round"`
	ActionType        uint32 `json:"action_type"`
	ActorID           uint64 `json:"actor_id"`
	SkillID           uint32 `json:"skill_id"`
	TargetID          uint64 `json:"target_id"`
	ItemUID           uint64 `json:"item_uid"`
	ItemID            uint32 `json:"item_id"`
	BagSlotIndex      uint32 `json:"bag_slot_index"`
	SwitchPet         uint64 `json:"switch_pet_uid"`
	AutoBattleEnabled bool   `json:"auto_battle_enabled"`
}

type BattleActionResp struct {
	Accepted       bool   `json:"accepted"`
	Reason         string `json:"reason"`
	CaptureSuccess bool   `json:"capture_success,omitempty"`
}

type BattleEvent struct {
	EventType uint32 `json:"event_type"`
	SourceID  uint64 `json:"source_id"`
	TargetID  uint64 `json:"target_id"`
	SkillID   uint32 `json:"skill_id"`
	Value     int32  `json:"value"`
	StateID   uint32 `json:"state_id"`
	Label     string `json:"label"`
}

type BattleActorState struct {
	ActorID    uint64   `json:"actor_id"`
	HP         uint32   `json:"hp"`
	HPMax      uint32   `json:"hp_max"`
	Dead       bool     `json:"dead"`
	CanAct     bool     `json:"can_act"`
	StatusIDs  []uint32 `json:"status_ids"`
	ChargeDone bool     `json:"charge_done"`
}

type BattleStatePush struct {
	BattleID             uint64             `json:"battle_id"`
	BattleVersion        uint32             `json:"battle_version"`
	Frame                uint32             `json:"frame"`
	ParticipantPlayerIDs []uint64           `json:"participant_player_ids"`
	Round                uint32             `json:"round"`
	Phase                string             `json:"phase"`
	Events               []BattleEvent      `json:"events"`
	Actors               []BattleActorState `json:"actors"`
	ActiveActorID        uint64             `json:"active_actor_id"`
	ActivePetUID         uint64             `json:"active_pet_uid"`
	CommandDeadlineMS    int64              `json:"command_deadline_ms"`
	AutoBattleEnabled    bool               `json:"auto_battle_enabled"`
	PendingActorIDs      []uint64           `json:"pending_actor_ids"`
	ControllableActorIDs []uint64           `json:"controllable_actor_ids"`
}

// LevelUpBonus 描述一次升级后累加的裸装战斗属性加成，供客户端弹窗展示。
type LevelUpBonus struct {
	HPMax uint32 `json:"hp_max"`
	ATK   uint32 `json:"atk"`
	SPD   uint32 `json:"spd"`
	MANA  uint32 `json:"mana"`
}

type BattleResultPush struct {
	BattleID         uint64                    `json:"battle_id"`
	Win              bool                      `json:"win"`
	ReturnSceneID    uint32                    `json:"return_scene_id"`
	ReturnPos        Vec2i                     `json:"return_pos"`
	Reason           string                    `json:"reason"`
	RewardGold       uint32                    `json:"reward_gold"`
	RewardPlayerExp  uint64                    `json:"reward_player_exp"`
	PlayerGold       uint32                    `json:"player_gold"`
	PlayerExp        uint64                    `json:"player_exp"`
	PlayerLevel      uint32                    `json:"player_level"`
	LevelUpCount     uint32                    `json:"level_up_count"`
	AttrPointsGained uint32                    `json:"attr_points_gained"`
	LevelUpBonus     *LevelUpBonus             `json:"level_up_bonus,omitempty"`
	FreeAttrPoints   uint32                    `json:"free_attr_points"`
	ExpToNext        uint64                    `json:"exp_to_next"`
	PetRewards       []BattlePetReward         `json:"pet_rewards"`
	Rewards          []QuestReward             `json:"rewards,omitempty"`
	DropTexts        []string                  `json:"drop_texts"`
	CaptureSuccess   bool                      `json:"capture_success,omitempty"`
	CaptureMonsterID uint32                    `json:"capture_monster_id,omitempty"`
	CapturedPetID    uint32                    `json:"captured_pet_id,omitempty"`
	CapturedPetUID   uint64                    `json:"captured_pet_uid,omitempty"`
	SkillProgress    []BattleSkillProgressPush `json:"skill_progress,omitempty"`
}

// BattleSkillProgressPush 描述战斗结算弹窗中展示的武器技能学习进度。
type BattleSkillProgressPush struct {
	SkillID          uint32 `json:"skill_id"`
	SkillName        string `json:"skill_name"`
	SkillExp         uint32 `json:"skill_exp"`
	LearnExpRequired uint32 `json:"learn_exp_required"`
	ExpGained        uint32 `json:"exp_gained"`
	NewlyLearned     bool   `json:"newly_learned"`
}

type BattlePetReward struct {
	PetUID           uint64 `json:"pet_uid"`
	PetID            uint32 `json:"pet_id,omitempty"`
	Level            uint32 `json:"level,omitempty"`
	Exp              uint64 `json:"exp"`
	ExpGained        uint64 `json:"exp_gained,omitempty"`
	LevelUpCount     uint32 `json:"level_up_count,omitempty"`
	AttrPointsGained uint32 `json:"attr_points_gained,omitempty"`
	FreeAttrPoints   uint32 `json:"free_attr_points,omitempty"`
	ExpToNext        uint64 `json:"exp_to_next,omitempty"`
}

type BagListReq struct {
	ContainerType string `json:"container_type,omitempty"`
	Page          uint32 `json:"page,omitempty"`
	PageSize      uint32 `json:"page_size,omitempty"`
	Category      string `json:"category,omitempty"`
}

type UseItemReq struct {
	ContainerType  string         `json:"container_type"`
	SlotIndex      uint32         `json:"slot_index"`
	Quantity       uint64         `json:"quantity"`
	TargetPetUID   uint64         `json:"target_pet_uid,omitempty"`
	TargetPlayerID uint64         `json:"target_player_id,omitempty"`
	TargetItemUID  string         `json:"target_item_uid,omitempty"`
	ExtraArgs      map[string]any `json:"extra_args,omitempty"`
}

type UseItemAppliedEffect struct {
	Category  string `json:"category"`
	FieldKey  string `json:"field_key"`
	Operation string `json:"operation"`
	Value     int64  `json:"value,omitempty"`
	BoolValue bool   `json:"bool_value,omitempty"`
}

type UseItemResult struct {
	EffectType           string                 `json:"effect_type"`
	ExpandTarget         string                 `json:"expand_target,omitempty"`
	ExpandSlots          uint32                 `json:"expand_slots,omitempty"`
	NewCapacity          uint32                 `json:"new_capacity,omitempty"`
	TargetPetUID         uint64                 `json:"target_pet_uid,omitempty"`
	RestoredHP           uint32                 `json:"restored_hp,omitempty"`
	NewPetHP             uint32                 `json:"new_pet_hp,omitempty"`
	UnlockedTalismanSlot string                 `json:"unlocked_talisman_slot,omitempty"`
	Rewards              []QuestReward          `json:"rewards,omitempty"`
	AppliedEffects       []UseItemAppliedEffect `json:"applied_effects,omitempty"`
	NeedsWalletPush      bool                   `json:"needs_wallet_push,omitempty"`
}

type UseItemResp struct {
	ContainerType string        `json:"container_type"`
	SlotIndex     uint32        `json:"slot_index"`
	ItemID        uint64        `json:"item_id"`
	UsedQuantity  uint64        `json:"used_quantity"`
	Result        UseItemResult `json:"result"`
}

// DropItemReq 描述玩家主动丢弃容器格子物品的请求。
// 实例化物品（item_uid 非空）应优先传 item_uid 定位唯一背包条目；可堆叠物品按 slot_index + quantity 部分丢弃。
type DropItemReq struct {
	ContainerType string `json:"container_type"`
	SlotIndex     uint32 `json:"slot_index"`
	ItemUID       string `json:"item_uid,omitempty"`
	Quantity      uint64 `json:"quantity"`
}

// DropItemResp 描述丢弃成功后客户端用于提示与刷新格子的摘要。
type DropItemResp struct {
	ContainerType string `json:"container_type"`
	SlotIndex     uint32 `json:"slot_index"`
	ItemUID       string `json:"item_uid,omitempty"`
	ItemID        uint64 `json:"item_id"`
	ItemName      string `json:"item_name"`
	DroppedQty    uint64 `json:"dropped_quantity"`
}

type ContainerListReq struct {
	ContainerType string `json:"container_type"`
}

type WalletQueryReq struct{}

type BagToWarehouseReq struct {
	EntityID      uint64 `json:"entity_id"`
	FromSlotIndex uint32 `json:"from_slot_index"`
	Quantity      uint64 `json:"quantity"`
}

type BagToWarehouseResp struct {
	Accepted          bool   `json:"accepted"`
	Reason            string `json:"reason"`
	MovedItemID       uint64 `json:"moved_item_id"`
	MovedItemUID      string `json:"moved_item_uid"`
	MovedQuantity     uint64 `json:"moved_quantity"`
	FromContainerType string `json:"from_container_type"`
	ToContainerType   string `json:"to_container_type"`
	FromSlotIndex     uint32 `json:"from_slot_index"`
	ToSlotIndex       uint32 `json:"to_slot_index"`
}

type WarehouseToBagReq struct {
	EntityID      uint64 `json:"entity_id"`
	FromSlotIndex uint32 `json:"from_slot_index"`
	Quantity      uint64 `json:"quantity"`
}

type WarehouseToBagResp struct {
	Accepted          bool   `json:"accepted"`
	Reason            string `json:"reason"`
	MovedItemID       uint64 `json:"moved_item_id"`
	MovedItemUID      string `json:"moved_item_uid"`
	MovedQuantity     uint64 `json:"moved_quantity"`
	FromContainerType string `json:"from_container_type"`
	ToContainerType   string `json:"to_container_type"`
	FromSlotIndex     uint32 `json:"from_slot_index"`
	ToSlotIndex       uint32 `json:"to_slot_index"`
}

type ContainerSortReq struct {
	ContainerType string `json:"container_type"`
}

type ContainerSortResp struct {
	ContainerType string `json:"container_type"`
	Sorted        bool   `json:"sorted"`
}

type ContainerMoveReq struct {
	ContainerType string `json:"container_type"`
	FromSlotIndex uint32 `json:"from_slot_index"`
	ToSlotIndex   uint32 `json:"to_slot_index"`
	Quantity      uint64 `json:"quantity"`
}

type ContainerMoveResp struct {
	ContainerType string `json:"container_type"`
	FromSlotIndex uint32 `json:"from_slot_index"`
	ToSlotIndex   uint32 `json:"to_slot_index"`
	Moved         bool   `json:"moved"`
	Reason        string `json:"reason"`
}

type WalletSnapshot struct {
	TotalCopper uint64 `json:"total_copper"`
	Gold        uint64 `json:"gold"`
	Silver      uint64 `json:"silver"`
	Copper      uint64 `json:"copper"`
}

type ContainerItemSnapshot struct {
	SlotIndex           uint32                       `json:"slot_index"`
	ItemID              uint64                       `json:"item_id"`
	ItemUID             string                       `json:"item_uid"`
	Quantity            uint64                       `json:"quantity"`
	IsBound             bool                         `json:"is_bound"`
	ItemName            string                       `json:"item_name"`
	ItemType            string                       `json:"item_type"`
	ItemSubType         string                       `json:"item_sub_type"`
	Quality             uint32                       `json:"quality"`
	Icon                string                       `json:"icon,omitempty"`
	RequiredLevel       uint32                       `json:"required_level"`
	EnhanceLevel        uint32                       `json:"enhance_level"`
	IsDamaged           bool                         `json:"is_damaged,omitempty"`
	Usable              bool                         `json:"usable"`
	CanDrop             bool                         `json:"can_drop"`
	TargetType          string                       `json:"target_type"`
	EffectType          string                       `json:"effect_type"`
	EquipSlot           string                       `json:"equip_slot,omitempty"`
	Description         string                       `json:"description,omitempty"`
	DescriptionMentions []ItemDescriptionMention     `json:"description_mentions,omitempty"`
	Bonus               PlayerEquipmentBonusSnapshot `json:"bonus,omitempty"`
	EnhancePreview      *EnhancePreviewSnapshot      `json:"enhance_preview,omitempty"`
	RepairPreview       *RepairPreviewSnapshot       `json:"repair_preview,omitempty"`
}

// EnhancePreviewRowSnapshot 描述强化预览表中的一行属性对比。
type EnhancePreviewRowSnapshot struct {
	Label   string `json:"label"`
	Current string `json:"current"`
	NextMin string `json:"next_min"`
	NextMax string `json:"next_max"`
}

// EnhanceMaterialOptionSnapshot 描述背包内可选强化材料。
type EnhanceMaterialOptionSnapshot struct {
	ItemID                  uint64 `json:"item_id"`
	ItemName                string `json:"item_name"`
	OwnedQuantity           uint64 `json:"owned_quantity"`
	EffectiveSuccessRatePct uint32 `json:"effective_success_rate_pct"`
	FailurePenalty          string `json:"failure_penalty"`
	FailurePenaltyLabel     string `json:"failure_penalty_label"`
	Description             string `json:"description,omitempty"`
}

// EnhancePreviewSnapshot 描述客户端强化弹窗所需的权威预览数据。
type EnhancePreviewSnapshot struct {
	CanEnhance              bool                            `json:"can_enhance"`
	MaxEnhanceLevel         uint32                          `json:"max_enhance_level"`
	SuccessRatePct          uint32                          `json:"success_rate_pct"`
	RequiredLevel           uint32                          `json:"required_level"`
	RequiredLevelBandMin    uint32                          `json:"required_level_band_min"`
	RequiredLevelBandLabel  string                          `json:"required_level_band_label"`
	CostGoldCopper          uint64                          `json:"cost_gold_copper"`
	CostItemID              uint64                          `json:"cost_item_id"`
	CostItemName            string                          `json:"cost_item_name"`
	CostQuantity            uint64                          `json:"cost_quantity"`
	OwnedCostQuantity       uint64                          `json:"owned_cost_quantity"`
	EnhanceMaterialCategory string                          `json:"enhance_material_category"`
	Materials               []EnhanceMaterialOptionSnapshot `json:"materials"`
	Rows                    []EnhancePreviewRowSnapshot     `json:"rows"`
}

// RepairPreviewSnapshot 描述客户端修复弹窗所需的权威预览数据。
type RepairPreviewSnapshot struct {
	CanRepair         bool   `json:"can_repair"`
	CostItemID        uint64 `json:"cost_item_id"`
	CostItemName      string `json:"cost_item_name"`
	CostQuantity      uint64 `json:"cost_quantity"`
	OwnedCostQuantity uint64 `json:"owned_cost_quantity"`
}

type ContainerSnapshot struct {
	ContainerType string                  `json:"container_type"`
	Capacity      uint32                  `json:"capacity"`
	MaxCapacity   uint32                  `json:"max_capacity"`
	UsedSlots     uint32                  `json:"used_slots"`
	Page          uint32                  `json:"page,omitempty"`
	PageSize      uint32                  `json:"page_size,omitempty"`
	TotalItems    uint32                  `json:"total_items,omitempty"`
	Category      string                  `json:"category,omitempty"`
	Items         []ContainerItemSnapshot `json:"items"`
}

type BagListResp struct {
	Container     ContainerSnapshot            `json:"container"`
	Wallet        WalletSnapshot               `json:"wallet"`
	EquippedItems []PlayerEquippedItemSnapshot `json:"equipped_items"`
}

type ContainerListResp struct {
	Container ContainerSnapshot `json:"container"`
	Wallet    WalletSnapshot    `json:"wallet"`
}

type WalletQueryResp struct {
	Wallet WalletSnapshot `json:"wallet"`
}

type BagSlotUpdate struct {
	SlotIndex uint32                 `json:"slot_index"`
	Deleted   bool                   `json:"deleted"`
	Item      *ContainerItemSnapshot `json:"item,omitempty"`
}

type BagUpdatePush struct {
	ContainerType string          `json:"container_type"`
	Capacity      uint32          `json:"capacity"`
	MaxCapacity   uint32          `json:"max_capacity"`
	UsedSlots     uint32          `json:"used_slots"`
	Updates       []BagSlotUpdate `json:"updates"`
}

type WalletUpdatePush struct {
	Wallet      WalletSnapshot `json:"wallet"`
	ReasonType  string         `json:"reason_type"`
	ReasonRefID uint64         `json:"reason_ref_id"`
}

type CurrencyCost struct {
	CurrencyType string `json:"currency_type"`
	TotalCopper  uint64 `json:"total_copper"`
	Gold         uint64 `json:"gold"`
	Silver       uint64 `json:"silver"`
	Copper       uint64 `json:"copper"`
}

type BuyItemReq struct {
	ShopID   uint64 `json:"shop_id"`
	GoodsID  uint64 `json:"goods_id"`
	ItemID   uint64 `json:"item_id"`
	Quantity uint64 `json:"quantity"`
}

type BuyItemResp struct {
	ShopID   uint64         `json:"shop_id"`
	GoodsID  uint64         `json:"goods_id"`
	ItemID   uint64         `json:"item_id"`
	Quantity uint64         `json:"quantity"`
	Cost     CurrencyCost   `json:"cost"`
	Wallet   WalletSnapshot `json:"wallet"`
}

type QuestListReq struct{}

type QuestObjectiveGuide struct {
	SceneID         uint32 `json:"scene_id,omitempty"`
	NPCID           uint64 `json:"npc_id,omitempty"`
	NPCName         string `json:"npc_name,omitempty"`
	Text            string `json:"text,omitempty"`
	MenuEntryID     uint64 `json:"menu_entry_id,omitempty"`
	DialogueEntryID uint64 `json:"dialogue_entry_id,omitempty"`
}

type QuestObjectiveState struct {
	ObjectiveID    uint64               `json:"objective_id"`
	EventType      string               `json:"event_type,omitempty"`
	Description    string               `json:"description"`
	Current        uint32               `json:"current"`
	Target         uint32               `json:"target"`
	Completed      bool                 `json:"completed"`
	TargetSelector map[string]any       `json:"target_selector,omitempty"`
	Guide          *QuestObjectiveGuide `json:"guide,omitempty"`
}

type QuestReward struct {
	Type     string `json:"type"`
	Value    uint64 `json:"value"`
	ItemID   uint64 `json:"item_id"`
	ItemName string `json:"item_name,omitempty"`
	Count    uint64 `json:"count"`
	PetID    uint64 `json:"pet_id"`
	AttrKey  string `json:"attr_key,omitempty"`
}

type QuestSummary struct {
	QuestID              uint64                `json:"quest_id"`
	QuestType            string                `json:"quest_type"`
	ClientIconID         uint64                `json:"client_icon_id"`
	State                string                `json:"state"`
	Tracked              bool                  `json:"tracked"`
	StartNPCID           uint64                `json:"start_npc_id"`
	SubmitNPCID          uint64                `json:"submit_npc_id"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	CompletionPromptText string                `json:"completion_prompt_text,omitempty"`
	Objectives           []QuestObjectiveState `json:"objectives"`
	Rewards              []QuestReward         `json:"rewards,omitempty"`
}

type QuestListResp struct {
	Quests         []QuestSummary `json:"quests"`
	TrackedQuestID uint64         `json:"tracked_quest_id"`
	ServerTimeMS   int64          `json:"server_time_ms"`
}

type QuestUpdatePush struct {
	Quest QuestSummary `json:"quest"`
}

type QuestRemovePush struct {
	QuestID uint64 `json:"quest_id"`
}

type QuestAcceptReq struct {
	QuestID uint64 `json:"quest_id"`
	NPCID   uint64 `json:"npc_id"`
}

type QuestAcceptResp struct {
	Accepted           bool         `json:"accepted"`
	Reason             string       `json:"reason"`
	Quest              QuestSummary `json:"quest"`
	ClientAnimationKey string       `json:"client_animation_key,omitempty"`
}

type QuestSubmitReq struct {
	QuestID uint64 `json:"quest_id"`
	NPCID   uint64 `json:"npc_id"`
}

type QuestSubmitResp struct {
	Accepted             bool          `json:"accepted"`
	Reason               string        `json:"reason"`
	Quest                QuestSummary  `json:"quest"`
	Rewards              []QuestReward `json:"rewards"`
	PlayerLevel          uint32        `json:"player_level,omitempty"`
	LevelUpCount         uint32        `json:"level_up_count,omitempty"`
	AttrPointsGained     uint32        `json:"attr_points_gained,omitempty"`
	LevelUpBonus         *LevelUpBonus `json:"level_up_bonus,omitempty"`
	ClientAnimationKey   string        `json:"client_animation_key,omitempty"`
	CompletionPromptText string        `json:"completion_prompt_text,omitempty"`
}

type QuestTrackReq struct {
	QuestID uint64 `json:"quest_id"`
}

type QuestTrackResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
	QuestID  uint64 `json:"quest_id"`
}
