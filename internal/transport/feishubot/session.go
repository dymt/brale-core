package feishubot

import (
	"sync"
	"time"
)

type sessionStep string

const (
	stepAwaitObserveSymbol sessionStep = "await_observe_symbol"
	stepAwaitLatestSymbol  sessionStep = "await_latest_symbol"
)

type session struct {
	SenderID  string
	ChatID    string
	Step      sessionStep
	UpdatedAt time.Time
}

type sessionStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	data map[string]*session
}

func newSessionStore(ttl time.Duration) *sessionStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &sessionStore{ttl: ttl, data: make(map[string]*session)}
}

func (s *sessionStore) get(senderID string) (*session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.data[senderID]
	if !ok {
		return nil, false
	}
	if time.Since(current.UpdatedAt) > s.ttl {
		delete(s.data, senderID)
		return nil, false
	}
	return current, true
}

func (s *sessionStore) save(value *session) {
	value.UpdatedAt = time.Now()
	s.mu.Lock()
	s.data[value.SenderID] = value
	s.mu.Unlock()
}

func (s *sessionStore) delete(senderID string) {
	s.mu.Lock()
	delete(s.data, senderID)
	s.mu.Unlock()
}
