package session

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/platform/idgen"
	"pocket-pet-remake/server/internal/protocol"
)

var ErrSessionNotFound = errors.New("session not found")

type Service struct {
	mu                sync.RWMutex
	sessionsByID      map[string]*Session
	sessionIDByPlayer map[uint64]string
	sessionIDByConn   map[string]string
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	now               func() time.Time
	logger            *log.Logger
}

func NewService(logger *log.Logger, heartbeatInterval, heartbeatTimeout time.Duration) *Service {
	return &Service{
		sessionsByID:      make(map[string]*Session),
		sessionIDByPlayer: make(map[uint64]string),
		sessionIDByConn:   make(map[string]string),
		heartbeatInterval: heartbeatInterval,
		heartbeatTimeout:  heartbeatTimeout,
		now:               time.Now,
		logger:            logger,
	}
}

func (s *Service) HeartbeatInterval() time.Duration {
	return s.heartbeatInterval
}

func (s *Service) IsAuthenticated(connID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.sessionIDByConn[connID]
	return ok
}

func (s *Service) GetByConnID(connID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID, ok := s.sessionIDByConn[connID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	current, ok := s.sessionsByID[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	copy := *current
	return &copy, nil
}

func (s *Service) Bind(playerID uint64, conn Conn) (*Session, error) {
	now := s.now()
	sessionID, err := idgen.RandomHex(16)
	if err != nil {
		return nil, err
	}
	reconnectToken, err := idgen.RandomHex(16)
	if err != nil {
		return nil, err
	}

	newSession := &Session{
		ID:             sessionID,
		PlayerID:       playerID,
		ConnID:         conn.ID(),
		Conn:           conn,
		ReconnectToken: reconnectToken,
		CreatedAt:      now,
		LastHeartbeat:  now,
	}

	var kicked Conn
	s.mu.Lock()
	if oldSessionID, ok := s.sessionIDByPlayer[playerID]; ok {
		if oldSession, exists := s.sessionsByID[oldSessionID]; exists {
			kicked = oldSession.Conn
			delete(s.sessionIDByConn, oldSession.ConnID)
			delete(s.sessionsByID, oldSessionID)
		}
	}
	s.sessionIDByPlayer[playerID] = newSession.ID
	s.sessionIDByConn[newSession.ConnID] = newSession.ID
	s.sessionsByID[newSession.ID] = newSession
	s.mu.Unlock()

	if kicked != nil {
		packet, packetErr := protocol.NewJSONPacket(protocol.CmdForceOfflinePush, 0, errcode.WSCodeSuccess, protocol.ForceOfflinePush{
			Reason: "account logged in elsewhere",
		})
		if packetErr == nil {
			_ = kicked.SendPacket(packet)
		}
		_ = kicked.Close()
	}

	return newSession, nil
}

func (s *Service) Touch(connID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, ok := s.sessionIDByConn[connID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	session, ok := s.sessionsByID[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	session.LastHeartbeat = s.now()
	return session, nil
}

func (s *Service) Disconnect(connID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeByConnIDLocked(connID)
}

func (s *Service) StartSweeper(ctx context.Context) {
	interval := s.heartbeatInterval / 2
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpired()
		}
	}
}

func (s *Service) cleanupExpired() {
	now := s.now()
	stale := make([]Conn, 0)

	s.mu.Lock()
	for sessionID, session := range s.sessionsByID {
		if now.Sub(session.LastHeartbeat) <= s.heartbeatTimeout {
			continue
		}
		stale = append(stale, session.Conn)
		delete(s.sessionIDByConn, session.ConnID)
		delete(s.sessionIDByPlayer, session.PlayerID)
		delete(s.sessionsByID, sessionID)
	}
	s.mu.Unlock()

	for _, conn := range stale {
		s.logger.Printf("close stale session conn_id=%s", conn.ID())
		_ = conn.Close()
	}
}

func (s *Service) removeByConnIDLocked(connID string) {
	sessionID, ok := s.sessionIDByConn[connID]
	if !ok {
		return
	}
	session, ok := s.sessionsByID[sessionID]
	if !ok {
		delete(s.sessionIDByConn, connID)
		return
	}
	delete(s.sessionIDByConn, connID)
	delete(s.sessionIDByPlayer, session.PlayerID)
	delete(s.sessionsByID, sessionID)
}
