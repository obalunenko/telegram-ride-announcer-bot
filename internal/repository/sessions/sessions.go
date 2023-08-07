// Package sessions provide a repository for sessions.
package sessions

import (
	"context"
	"errors"
	"sync"

	"github.com/gofrs/uuid/v5"
)

var (
	// ErrAlreadyExists is returned when a session already exists.
	ErrAlreadyExists = errors.New("session already exists")
	// ErrNotFound is returned when a session is not found.
	ErrNotFound = errors.New("session not found")
)

// Repository provides access to the session storage.
type Repository interface {
	// CreateSession creates a new session.
	CreateSession(ctx context.Context, userID, chatID int64, stateID *uuid.UUID) error
	// ListSessions returns all sessions.
	ListSessions(ctx context.Context) ([]*Session, error)
	// GetSessionByUserID returns a session by user ID.
	GetSessionByUserID(ctx context.Context, userID int64) (*Session, error)
	// UpdateSession updates a session.
	UpdateSession(ctx context.Context, sess *Session) error
	// DeleteSession deletes a session.
	DeleteSession(ctx context.Context, userID int64) error
}

// Session represents a session.
type Session struct {
	ID      uuid.UUID  `db:"id"`
	UserID  int64      `db:"user_id"`
	ChatID  int64      `db:"chat_id"`
	StateID *uuid.UUID `db:"state_id"`
}

func newSession(userID, chatID int64, stateID *uuid.UUID) *Session {
	return &Session{
		ID:      uuid.Must(uuid.NewV4()),
		UserID:  userID,
		ChatID:  chatID,
		StateID: stateID,
	}
}

// NewInMemory creates a new in-memory repository.
func NewInMemory() Repository {
	return &inMemoryRepository{
		RWMutex:  sync.RWMutex{},
		sessions: make(map[int64]*Session),
	}
}

// inMemoryRepository is an in-memory repository for sessions.
type inMemoryRepository struct {
	sync.RWMutex

	sessions map[int64]*Session
}

func (i *inMemoryRepository) ListSessions(_ context.Context) ([]*Session, error) {
	i.RLock()
	defer i.RUnlock()

	sessions := make([]*Session, 0, len(i.sessions))

	for _, s := range i.sessions {
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (i *inMemoryRepository) CreateSession(_ context.Context, userID, chatID int64, stateID *uuid.UUID) error {
	s := newSession(userID, chatID, stateID)

	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[userID]; ok {
		return ErrAlreadyExists
	}

	i.sessions[userID] = s

	return nil
}

func (i *inMemoryRepository) GetSessionByUserID(_ context.Context, userID int64) (*Session, error) {
	i.RLock()
	defer i.RUnlock()

	s, ok := i.sessions[userID]
	if !ok {
		return nil, ErrNotFound
	}

	return s, nil
}

func (i *inMemoryRepository) UpdateSession(_ context.Context, sess *Session) error {
	if sess == nil {
		return errors.New("session is nil")
	}

	userID := sess.UserID

	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[userID]; !ok {
		return ErrNotFound
	}

	i.sessions[userID] = sess

	return nil
}

func (i *inMemoryRepository) DeleteSession(_ context.Context, userID int64) error {
	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[userID]; !ok {
		return ErrNotFound
	}

	delete(i.sessions, userID)

	return nil
}
