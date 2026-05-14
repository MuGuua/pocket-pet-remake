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

type InteractResp struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
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
	ActorID   uint64   `json:"actor_id"`
	ActorType uint32   `json:"actor_type"`
	PetUID    uint64   `json:"pet_uid"`
	PetID     uint32   `json:"pet_id"`
	Name      string   `json:"name"`
	HP        uint32   `json:"hp"`
	HPMax     uint32   `json:"hp_max"`
	SkillIDs  []uint32 `json:"skill_ids"`
}

type BattleStartPush struct {
	BattleID      uint64                `json:"battle_id"`
	BattleType    uint32                `json:"battle_type"`
	BattleVersion uint32                `json:"battle_version"`
	Allies        []BattleActorSnapshot `json:"allies"`
	Enemies       []BattleActorSnapshot `json:"enemies"`
	Round         uint32                `json:"round"`
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
}

type BattleActorState struct {
	ActorID uint64 `json:"actor_id"`
	HP      uint32 `json:"hp"`
	HPMax   uint32 `json:"hp_max"`
	Dead    bool   `json:"dead"`
}

type BattleStatePush struct {
	BattleID      uint64             `json:"battle_id"`
	BattleVersion uint32             `json:"battle_version"`
	Round         uint32             `json:"round"`
	Events        []BattleEvent      `json:"events"`
	Actors        []BattleActorState `json:"actors"`
}

type BattleResultPush struct {
	BattleID      uint64 `json:"battle_id"`
	Win           bool   `json:"win"`
	ReturnSceneID uint32 `json:"return_scene_id"`
	ReturnPos     Vec2i  `json:"return_pos"`
	Reason        string `json:"reason"`
}
