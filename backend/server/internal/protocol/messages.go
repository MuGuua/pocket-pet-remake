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

type EntityBrief struct {
	EntityID   uint64 `json:"entity_id"`
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
	Self           PlayerBrief   `json:"self"`
	SceneID        uint32        `json:"scene_id"`
	SelfPos        Vec2i         `json:"self_pos"`
	SceneVersion   uint32        `json:"scene_version"`
	NearbyEntities []EntityBrief `json:"nearby_entities"`
	Lineup         []PetBrief    `json:"lineup"`
	Gold           uint32        `json:"gold"`
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

type EntityMovePush struct {
	SceneVersion uint32 `json:"scene_version"`
	EntityID     uint64 `json:"entity_id"`
	MoveSeq      uint32 `json:"move_seq"`
	FromPos      Vec2i  `json:"from_pos"`
	ToPos        Vec2i  `json:"to_pos"`
	Speed        uint32 `json:"speed"`
}

type WorldResyncPush struct {
	SceneID        uint32        `json:"scene_id"`
	SelfPos        Vec2i         `json:"self_pos"`
	SceneVersion   uint32        `json:"scene_version"`
	NearbyEntities []EntityBrief `json:"nearby_entities"`
}

type BattleActorSnapshot struct {
	ActorID     uint64                `json:"actor_id"`
	ActorType   uint32                `json:"actor_type"`
	PetUID      uint64                `json:"pet_uid"`
	PetID       uint32                `json:"pet_id"`
	Name        string                `json:"name"`
	HP          uint32                `json:"hp"`
	HPMax       uint32                `json:"hp_max"`
	ATK         uint32                `json:"atk"`
	DEF         uint32                `json:"def"`
	SPD         uint32                `json:"spd"`
	Skills      []BattleSkillSnapshot `json:"skills"`
	SkillIDs    []uint32              `json:"skill_ids"`
	StatusIDs   []uint32              `json:"status_ids"`
	LineupIndex uint32                `json:"lineup_index"`
}

type BattleSkillSnapshot struct {
	SkillID    uint32 `json:"skill_id"`
	Name       string `json:"name"`
	TargetType string `json:"target_type"`
}

type BattleStartPush struct {
	BattleID             uint64                `json:"battle_id"`
	BattleType           uint32                `json:"battle_type"`
	BattleVersion        uint32                `json:"battle_version"`
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
	OpID       uint32 `json:"op_id"`
	BattleID   uint64 `json:"battle_id"`
	Round      uint32 `json:"round"`
	ActionType uint32 `json:"action_type"`
	ActorID    uint64 `json:"actor_id"`
	SkillID    uint32 `json:"skill_id"`
	TargetID   uint64 `json:"target_id"`
	ItemUID    uint64 `json:"item_uid"`
	SwitchPet  uint64 `json:"switch_pet_uid"`
	AutoBattleEnabled bool `json:"auto_battle_enabled"`
}

type BattleActionResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
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
	BattleID      uint64 `json:"battle_id"`
	Win           bool   `json:"win"`
	ReturnSceneID uint32 `json:"return_scene_id"`
	ReturnPos     Vec2i  `json:"return_pos"`
	Reason        string `json:"reason"`
}

type QuestListReq struct{}

type QuestObjectiveState struct {
	ObjectiveID uint64 `json:"objective_id"`
	Description string `json:"description"`
	Current     uint32 `json:"current"`
	Target      uint32 `json:"target"`
	Completed   bool   `json:"completed"`
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
	Accepted bool         `json:"accepted"`
	Reason   string       `json:"reason"`
	Quest    QuestSummary `json:"quest"`
}

type QuestTrackReq struct {
	QuestID uint64 `json:"quest_id"`
}

type QuestTrackResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
	QuestID  uint64 `json:"quest_id"`
}
