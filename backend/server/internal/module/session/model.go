package session

import (
	"time"

	"pocket-pet-remake/server/internal/protocol"
)

type Conn interface {
	ID() string
	SendPacket(packet *protocol.Packet) error
	Close() error
}

type Session struct {
	ID             string
	PlayerID       uint64
	ConnID         string
	Conn           Conn
	ReconnectToken string
	CreatedAt      time.Time
	LastHeartbeat  time.Time
}
