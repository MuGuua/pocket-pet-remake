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
var ErrReconnectTokenInvalid = errors.New("reconnect token invalid")

type Service struct {
	mu                sync.RWMutex
	sessionsByID      map[string]*Session
	sessionIDByPlayer map[uint64]string
	sessionIDByConn   map[string]string
	sessionIDByToken  map[string]string
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	now               func() time.Time
	logger            *log.Logger
	onDisconnect      func(playerID uint64)
}

func NewService(logger *log.Logger, heartbeatInterval, heartbeatTimeout time.Duration) *Service {
	return &Service{
		sessionsByID:      make(map[string]*Session),
		sessionIDByPlayer: make(map[uint64]string),
		sessionIDByConn:   make(map[string]string),
		sessionIDByToken:  make(map[string]string),
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

func (s *Service) GetByPlayerID(playerID uint64) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID, ok := s.sessionIDByPlayer[playerID]
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

func (s *Service) SetDisconnectHandler(handler func(playerID uint64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onDisconnect = handler
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
		kicked = s.removeSessionLocked(oldSessionID)
	}
	s.sessionIDByPlayer[playerID] = newSession.ID
	s.sessionIDByConn[newSession.ConnID] = newSession.ID
	s.sessionIDByToken[newSession.ReconnectToken] = newSession.ID
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

// Reconnect rebinds a previously disconnected session to a fresh socket and
// rotates the reconnect token so old reconnect packets cannot be replayed.
func (s *Service) Reconnect(reconnectToken string, conn Conn) (*Session, error) {
	now := s.now()
	if reconnectToken == "" {
		return nil, ErrReconnectTokenInvalid
	}
	newToken, err := idgen.RandomHex(16)
	if err != nil {
		return nil, err
	}

	var kicked Conn
	s.mu.Lock()
	sessionID, ok := s.sessionIDByToken[reconnectToken]
	if !ok {
		s.mu.Unlock()
		return nil, ErrReconnectTokenInvalid
	}
	current, ok := s.sessionsByID[sessionID]
	if !ok {
		delete(s.sessionIDByToken, reconnectToken)
		s.mu.Unlock()
		return nil, ErrReconnectTokenInvalid
	}
	if current.ConnID != "" && current.ConnID != conn.ID() {
		kicked = current.Conn
		delete(s.sessionIDByConn, current.ConnID)
	}
	delete(s.sessionIDByToken, current.ReconnectToken)
	current.ConnID = conn.ID()
	current.Conn = conn
	current.ReconnectToken = newToken
	current.LastHeartbeat = now
	current.DisconnectedAt = time.Time{}
	s.sessionIDByConn[current.ConnID] = current.ID
	s.sessionIDByToken[current.ReconnectToken] = current.ID
	copy := *current
	s.mu.Unlock()

	if kicked != nil {
		packet, packetErr := protocol.NewJSONPacket(protocol.CmdForceOfflinePush, 0, errcode.WSCodeSuccess, protocol.ForceOfflinePush{
			Reason: "account reconnected elsewhere",
		})
		if packetErr == nil {
			_ = kicked.SendPacket(packet)
		}
		_ = kicked.Close()
	}
	return &copy, nil
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
	playerID, removed := s.disconnectConnLocked(connID)
	handler := s.onDisconnect
	s.mu.Unlock()

	if removed && handler != nil {
		handler(playerID)
	}
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
	stalePlayerIDs := make([]uint64, 0)
	handler := s.onDisconnect

	s.mu.Lock()
	for sessionID, session := range s.sessionsByID {
		if session.ConnID == "" || session.Conn == nil {
			if session.DisconnectedAt.IsZero() || now.Sub(session.DisconnectedAt) <= s.heartbeatTimeout {
				continue
			}
			s.removeSessionLocked(sessionID)
			continue
		}
		if now.Sub(session.LastHeartbeat) <= s.heartbeatTimeout {
			continue
		}
		stale = append(stale, session.Conn)
		stalePlayerIDs = append(stalePlayerIDs, session.PlayerID)
		s.removeSessionLocked(sessionID)
	}
	s.mu.Unlock()

	for _, conn := range stale {
		s.logger.Printf("close stale session conn_id=%s", conn.ID())
		_ = conn.Close()
	}
	if handler != nil {
		for _, playerID := range stalePlayerIDs {
			handler(playerID)
		}
	}
}

func (s *Service) disconnectConnLocked(connID string) (uint64, bool) {
	sessionID, ok := s.sessionIDByConn[connID]
	if !ok {
		return 0, false
	}
	session, ok := s.sessionsByID[sessionID]
	if !ok {
		delete(s.sessionIDByConn, connID)
		return 0, false
	}
	delete(s.sessionIDByConn, connID)
	session.ConnID = ""
	session.Conn = nil
	session.LastHeartbeat = s.now()
	session.DisconnectedAt = session.LastHeartbeat
	return session.PlayerID, true
}

func (s *Service) removeSessionLocked(sessionID string) Conn {
	current, ok := s.sessionsByID[sessionID]
	if !ok {
		return nil
	}
	delete(s.sessionIDByConn, current.ConnID)
	delete(s.sessionIDByPlayer, current.PlayerID)
	delete(s.sessionIDByToken, current.ReconnectToken)
	delete(s.sessionsByID, sessionID)
	return current.Conn
}
