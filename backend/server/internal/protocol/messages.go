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
	OpID      uint32 `json:"op_id"`
	MoveSeq   uint32 `json:"move_seq"`
	SceneID   uint32 `json:"scene_id"`
	TargetPos Vec2i  `json:"target_pos"`
}

type MoveIntentResp struct {
	Accepted     bool   `json:"accepted"`
	MoveSeq      uint32 `json:"move_seq"`
	CorrectedPos Vec2i  `json:"corrected_pos"`
	Reason       string `json:"reason"`
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
