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
	PlayerID           uint64   `json:"player_id"`
	Name               string   `json:"name"`
	Level              uint32   `json:"level"`
	Exp                uint64   `json:"exp"`
	Gold               uint32   `json:"gold"`
	HP                 uint32   `json:"hp"`
	HPMax              uint32   `json:"hp_max"`
	Energy             uint32   `json:"energy"`
	EnergyMax          uint32   `json:"energy_max"`
	ATK                uint32   `json:"atk"`
	DEF                uint32   `json:"def"`
	SPD                uint32   `json:"spd"`
	MANA               uint32   `json:"mana"`
	HitPct             uint32   `json:"hit_pct"`
	DodgePct           uint32   `json:"dodge_pct"`
	CritRatePct        uint32   `json:"crit_rate_pct"`
	CritDmgPct         uint32   `json:"crit_dmg_pct"`
	PhysicalResistPct  uint32   `json:"physical_resist_pct"`
	SkillResistPct     uint32   `json:"skill_resist_pct"`
	ConfusionResistPct uint32   `json:"confusion_resist_pct"`
	SleepResistPct     uint32   `json:"sleep_resist_pct"`
	ParalysisResistPct uint32   `json:"paralysis_resist_pct"`
	SealResistPct      uint32   `json:"seal_resist_pct"`
	CurseResistPct     uint32   `json:"curse_resist_pct"`
	CritResistPct      uint32   `json:"crit_resist_pct"`
	CritDmgResistPct   uint32   `json:"crit_dmg_resist_pct"`
	CharacterResistPct uint32   `json:"character_resist_pct"`
	PetResistPct       uint32   `json:"pet_resist_pct"`
	MercenaryResistPct uint32   `json:"mercenary_resist_pct"`
	GenericShieldPct   uint32   `json:"generic_shield_pct"`
	SkillIDs           []uint32 `json:"skill_ids"`
	SkinID             string   `json:"skin_id"`
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
}

type PetBrief struct {
	PetUID uint64 `json:"pet_uid"`
	PetID  uint32 `json:"pet_id"`
	Level  uint32 `json:"level"`
	HP     uint32 `json:"hp"`
	HPMax  uint32 `json:"hp_max"`
}

type PetDetail struct {
	PetUID   uint64   `json:"pet_uid"`
	PetID    uint32   `json:"pet_id"`
	Level    uint32   `json:"level"`
	Exp      uint64   `json:"exp"`
	Quality  uint32   `json:"quality"`
	HP       uint32   `json:"hp"`
	HPMax    uint32   `json:"hp_max"`
	ATK      uint32   `json:"atk"`
	DEF      uint32   `json:"def"`
	SPD      uint32   `json:"spd"`
	SkillIDs []uint32 `json:"skill_ids"`
	InLineup bool     `json:"in_lineup"`
	// IsUsable 表示该实例对应的 pet_id 是否存在于启用中的系统宠物模板列表。
	IsUsable bool `json:"is_usable"`
}

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
}

type MoveIntentResp struct {
	Accepted     bool   `json:"accepted"`
	MoveSeq      uint32 `json:"move_seq"`
	SceneID      uint32 `json:"scene_id"`
	CorrectedPos Vec2i  `json:"corrected_pos"`
	Reason       string `json:"reason"`
}

type InteractReq struct {
	EntityID uint64 `json:"entity_id"`
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

type NPCActionReq struct {
	EntityID uint64 `json:"entity_id"`
	EntryID  string `json:"entry_id"`
}

type NPCActionResp struct {
	Accepted    bool           `json:"accepted"`
	Reason      string         `json:"reason"`
	EntityID    uint64         `json:"entity_id"`
	EntryID     string         `json:"entry_id"`
	ResultType  string         `json:"result_type"`
	Notice      string         `json:"notice"`
	NPCName     string         `json:"npc_name"`
	MenuEntries []NpcMenuEntry `json:"menu_entries"`
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
	SceneVersion uint32 `json:"scene_version"`
	EntityID     uint64 `json:"entity_id"`
	MoveSeq      uint32 `json:"move_seq"`
	FromPos      Vec2i  `json:"from_pos"`
	ToPos        Vec2i  `json:"to_pos"`
	Speed        uint32 `json:"speed"`
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
}

type WildEncounterResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
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
	SkillID      uint32 `json:"skill_id"`
	Name         string `json:"name"`
	TargetType   string `json:"target_type"`
	TargetCount  uint32 `json:"target_count"`
	AnimationKey  string `json:"animation_key"`
	SkillVisualID string `json:"skill_visual_id"`
	CastColor     string `json:"cast_color"`
	ImpactColor  string `json:"impact_color"`
	Projectile   bool   `json:"projectile"`
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

type BattleResultPush struct {
	BattleID         uint64            `json:"battle_id"`
	Win              bool              `json:"win"`
	ReturnSceneID    uint32            `json:"return_scene_id"`
	ReturnPos        Vec2i             `json:"return_pos"`
	Reason           string            `json:"reason"`
	RewardGold       uint32            `json:"reward_gold"`
	RewardPlayerExp  uint64            `json:"reward_player_exp"`
	PlayerGold       uint32            `json:"player_gold"`
	PlayerExp        uint64            `json:"player_exp"`
	PetRewards       []BattlePetReward `json:"pet_rewards"`
	DropTexts        []string          `json:"drop_texts"`
	CaptureSuccess   bool              `json:"capture_success,omitempty"`
	CaptureMonsterID uint32            `json:"capture_monster_id,omitempty"`
	CapturedPetID    uint32            `json:"captured_pet_id,omitempty"`
	CapturedPetUID   uint64            `json:"captured_pet_uid,omitempty"`
}

type BattlePetReward struct {
	PetUID uint64 `json:"pet_uid"`
	Exp    uint64 `json:"exp"`
}

type BagListReq struct {
	ContainerType string `json:"container_type,omitempty"`
}

type UseItemReq struct {
	ContainerType  string         `json:"container_type"`
	SlotIndex      uint32         `json:"slot_index"`
	Quantity       uint64         `json:"quantity"`
	TargetPetUID   uint64         `json:"target_pet_uid,omitempty"`
	TargetPlayerID uint64         `json:"target_player_id,omitempty"`
	ExtraArgs      map[string]any `json:"extra_args,omitempty"`
}

type UseItemResult struct {
	EffectType   string        `json:"effect_type"`
	ExpandTarget string        `json:"expand_target,omitempty"`
	ExpandSlots  uint32        `json:"expand_slots,omitempty"`
	NewCapacity  uint32        `json:"new_capacity,omitempty"`
	TargetPetUID uint64        `json:"target_pet_uid,omitempty"`
	RestoredHP   uint32        `json:"restored_hp,omitempty"`
	NewPetHP     uint32        `json:"new_pet_hp,omitempty"`
	Rewards      []QuestReward `json:"rewards,omitempty"`
}

type UseItemResp struct {
	ContainerType string        `json:"container_type"`
	SlotIndex     uint32        `json:"slot_index"`
	ItemID        uint64        `json:"item_id"`
	UsedQuantity  uint64        `json:"used_quantity"`
	Result        UseItemResult `json:"result"`
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
	SlotIndex    uint32 `json:"slot_index"`
	ItemID       uint64 `json:"item_id"`
	ItemUID      string `json:"item_uid"`
	Quantity     uint64 `json:"quantity"`
	IsBound      bool   `json:"is_bound"`
	ItemName     string `json:"item_name"`
	ItemType     string `json:"item_type"`
	ItemSubType  string `json:"item_sub_type"`
	Quality      uint32 `json:"quality"`
	Icon         string `json:"icon"`
	EnhanceLevel uint32 `json:"enhance_level"`
}

type ContainerSnapshot struct {
	ContainerType string                  `json:"container_type"`
	Capacity      uint32                  `json:"capacity"`
	MaxCapacity   uint32                  `json:"max_capacity"`
	UsedSlots     uint32                  `json:"used_slots"`
	Items         []ContainerItemSnapshot `json:"items"`
}

type BagListResp struct {
	Container ContainerSnapshot `json:"container"`
	Wallet    WalletSnapshot    `json:"wallet"`
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

type QuestObjectiveState struct {
	ObjectiveID uint64 `json:"objective_id"`
	Description string `json:"description"`
	Current     uint32 `json:"current"`
	Target      uint32 `json:"target"`
	Completed   bool   `json:"completed"`
}

type QuestReward struct {
	Type   string `json:"type"`
	Value  uint64 `json:"value"`
	ItemID uint64 `json:"item_id"`
	Count  uint64 `json:"count"`
	PetID  uint64 `json:"pet_id"`
}

type QuestSummary struct {
	QuestID     uint64                `json:"quest_id"`
	QuestType   string                `json:"quest_type"`
	State       string                `json:"state"`
	Tracked     bool                  `json:"tracked"`
	StartNPCID  uint64                `json:"start_npc_id"`
	SubmitNPCID uint64                `json:"submit_npc_id"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Objectives  []QuestObjectiveState `json:"objectives"`
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
	Accepted bool         `json:"accepted"`
	Reason   string       `json:"reason"`
	Quest    QuestSummary `json:"quest"`
}

type QuestSubmitReq struct {
	QuestID uint64 `json:"quest_id"`
	NPCID   uint64 `json:"npc_id"`
}

type QuestSubmitResp struct {
	Accepted bool          `json:"accepted"`
	Reason   string        `json:"reason"`
	Quest    QuestSummary  `json:"quest"`
	Rewards  []QuestReward `json:"rewards"`
}

type QuestTrackReq struct {
	QuestID uint64 `json:"quest_id"`
}

type QuestTrackResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
	QuestID  uint64 `json:"quest_id"`
}
