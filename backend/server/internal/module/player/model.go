package player

import "errors"

var ErrPlayerNotFound = errors.New("player not found")

type Profile struct {
	PlayerID uint64
	Name     string
	Level    uint32
	Gold     uint32
	SceneID  uint32
	PosX     int32
	PosY     int32
}
