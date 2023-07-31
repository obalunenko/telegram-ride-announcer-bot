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

type Repository interface {
	// CreateSession creates a new session.
	CreateSession(ctx context.Context, userID, chatID int64, state State) error
	// ListSessions returns all sessions.
	ListSessions(ctx context.Context) ([]*Session, error)
	// GetSessionByUserID returns a session by user ID.
	GetSessionByUserID(ctx context.Context, userID int64) (*Session, error)
	// UpdateSession updates a session.
	UpdateSession(ctx context.Context, sess *Session) error
	// DeleteSession deletes a session.
	DeleteSession(ctx context.Context, userID int64) error
}

//go:generate stringer -type=State -output=state_string.go -trimprefix=State
type State uint

const (
	stateUnknown State = iota

	StateStart
	StateNewTrip            // Just started creating a new trip
	StateNewTripName        // Waiting for trip name
	StateNewTripDate        // Waiting for trip date
	StateNewTripTime        // Waiting for trip time
	StateNewTripDescription // Waiting for trip description
	StateNewTripConfirm     // Waiting for confirmation

	stateSentinel // Sentinel value.
)

func (s State) Valid() bool {
	return s > stateUnknown && s < stateSentinel
}

type Session struct {
	ID     uuid.UUID
	UserID int64
	ChatID int64
	State  State
}

func newSession(userID, chatID int64, state State) *Session {
	return &Session{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: userID,
		ChatID: chatID,
		State:  state,
	}
}

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

func (i *inMemoryRepository) ListSessions(ctx context.Context) ([]*Session, error) {
	i.RLock()
	defer i.RUnlock()

	sessions := make([]*Session, 0, len(i.sessions))

	for _, s := range i.sessions {
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (i *inMemoryRepository) CreateSession(ctx context.Context, userID, chatID int64, state State) error {
	s := newSession(userID, chatID, state)

	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[userID]; ok {
		return ErrAlreadyExists
	}

	i.sessions[userID] = s

	return nil
}

func (i *inMemoryRepository) GetSessionByUserID(ctx context.Context, userID int64) (*Session, error) {
	i.RLock()
	defer i.RUnlock()

	s, ok := i.sessions[userID]
	if !ok {
		return nil, nil
	}

	return s, nil
}

func (i *inMemoryRepository) UpdateSession(ctx context.Context, sess *Session) error {
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

func (i *inMemoryRepository) DeleteSession(ctx context.Context, userID int64) error {
	i.Lock()
	defer i.Unlock()

	if _, ok := i.sessions[userID]; !ok {
		return ErrNotFound
	}

	delete(i.sessions, userID)

	return nil
}
