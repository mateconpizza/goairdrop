package webui

import (
	"errors"
	"sync"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	UserID     string
	Expiry     time.Time // idle expiration
	LastActive time.Time
}

type SessionStore interface {
	Create(userID, token string, expiry time.Time) error
	Get(token string) (*Session, error)
	Delete(token string) error
	Update(token string, sess *Session) error
}

type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]Session),
	}
}

func (s *MemoryStore) Create(userID, token string, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[token] = Session{
		UserID: userID,
		Expiry: expiry,
	}
	return nil
}

func (s *MemoryStore) Get(token string) (*Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	// expiración
	if time.Now().After(session.Expiry) {
		_ = s.Delete(token)
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

func (s *MemoryStore) Update(token string, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[token]
	if !ok {
		return ErrSessionNotFound
	}

	s.sessions[token] = *sess
	return nil
}

func (s *MemoryStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
	return nil
}

func (s *MemoryStore) Cleanup(interval time.Duration) {
	// go store.Cleanup(5 * time.Minute)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		s.mu.Lock()
		for token, sess := range s.sessions {
			if now.After(sess.Expiry) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}
