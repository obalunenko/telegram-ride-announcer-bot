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
// A user has one session per chat.
type Repository interface {
	// CreateSession creates a new session of the user in the chat.
	CreateSession(ctx context.Context, userID, chatID int64, stateID *uuid.UUID) error
	// ListSessions returns all sessions.
	ListSessions(ctx context.Context) ([]*Session, error)
	// GetSession returns the session of the user in the chat.
	GetSession(ctx context.Context, userID, chatID int64) (*Session, error)
	// UpdateSession updates a session.
	UpdateSession(ctx context.Context, sess *Session) error
	// DeleteSession deletes the session of the user in the chat.
	DeleteSession(ctx context.Context, userID, chatID int64) error
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

// key identifies the session of a user in a chat.
type key struct {
	userID int64
	chatID int64
}

// NewInMemory creates a new in-memory repository.
func NewInMemory() Repository {
	return &inMemoryRepository{
		RWMutex:  sync.RWMutex{},
		sessions: make(map[key]*Session),
	}
}

// inMemoryRepository is an in-memory repository for sessions.
type inMemoryRepository struct {
	sync.RWMutex

	sessions map[key]*Session
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

	k := key{userID: userID, chatID: chatID}

	if _, ok := i.sessions[k]; ok {
		return ErrAlreadyExists
	}

	i.sessions[k] = s

	return nil
}

func (i *inMemoryRepository) GetSession(_ context.Context, userID, chatID int64) (*Session, error) {
	i.RLock()
	defer i.RUnlock()

	s, ok := i.sessions[key{userID: userID, chatID: chatID}]
	if !ok {
		return nil, ErrNotFound
	}

	return s, nil
}

func (i *inMemoryRepository) UpdateSession(_ context.Context, sess *Session) error {
	if sess == nil {
		return errors.New("session is nil")
	}

	k := key{userID: sess.UserID, chatID: sess.ChatID}

	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[k]; !ok {
		return ErrNotFound
	}

	i.sessions[k] = sess

	return nil
}

func (i *inMemoryRepository) DeleteSession(_ context.Context, userID, chatID int64) error {
	k := key{userID: userID, chatID: chatID}

	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[k]; !ok {
		return ErrNotFound
	}

	delete(i.sessions, k)

	return nil
}
