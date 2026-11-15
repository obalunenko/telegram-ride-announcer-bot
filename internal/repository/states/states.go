// Package states provides a repository for states.
package states

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gofrs/uuid/v5"
)

var (
	// ErrNotFound is returned when a state is not found.
	ErrNotFound = errors.New("state not found")
	// ErrNoChanges is returned when no changes were applied during an update.
	ErrNoChanges = errors.New("no changes were applied")
	// ErrAlreadyExists is returned when a state already exists.
	ErrAlreadyExists = errors.New("state already exists")
)

// Repository provides access to the state storage.
// A user has one state per chat.
type Repository interface {
	// CreateState creates a new state. If the user already has a state in the chat,
	// it returns that state with ErrAlreadyExists.
	CreateState(ctx context.Context, params CreateParams) (*State, error)
	// ListStates returns all states.
	ListStates(ctx context.Context) ([]*State, error)
	// GetStateByID returns a state by ID.
	GetStateByID(ctx context.Context, id uuid.UUID) (*State, error)
	// UpdateState updates a state.
	UpdateState(ctx context.Context, state *State) error
}

// State represents a state.
type State struct {
	ID      uuid.UUID  `db:"id"`
	UserID  int64      `db:"user_id"`
	ChatID  int64      `db:"chat_id"`
	TripID  *uuid.UUID `db:"trip_id"`
	Command string     `db:"command"`
	Step    uint       `db:"step"`
}

// CreateParams contains the parameters for CreateState.
type CreateParams struct {
	UserID  int64
	ChatID  int64
	Command string
	TripID  *uuid.UUID
	Step    uint
}

// NewInMemory creates a new in-memory repository.
func NewInMemory() Repository {
	return &inMemoryRepository{
		RWMutex: sync.RWMutex{},
		states:  make(map[uuid.UUID]*State),
	}
}

type inMemoryRepository struct {
	sync.RWMutex

	states map[uuid.UUID]*State
}

func (i *inMemoryRepository) CreateState(_ context.Context, params CreateParams) (*State, error) {
	i.Lock()
	defer i.Unlock()

	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate id: %w", err)
	}

	newState := &State{
		ID:      id,
		UserID:  params.UserID,
		ChatID:  params.ChatID,
		TripID:  params.TripID,
		Command: params.Command,
		Step:    params.Step,
	}

	// Each user can only have one state per chat.
	for _, state := range i.states {
		if state.UserID == newState.UserID && state.ChatID == newState.ChatID {
			return state, ErrAlreadyExists
		}
	}

	i.states[newState.ID] = newState

	return newState, nil
}

func (i *inMemoryRepository) ListStates(_ context.Context) ([]*State, error) {
	i.RLock()
	defer i.RUnlock()

	states := make([]*State, 0, len(i.states))

	for _, state := range i.states {
		states = append(states, state)
	}

	return states, nil
}

func (i *inMemoryRepository) GetStateByID(_ context.Context, id uuid.UUID) (*State, error) {
	i.RLock()
	defer i.RUnlock()

	state, ok := i.states[id]
	if !ok {
		return nil, ErrNotFound
	}

	return state, nil
}

func (i *inMemoryRepository) UpdateState(_ context.Context, newState *State) error {
	i.Lock()
	defer i.Unlock()

	st, ok := i.states[newState.ID]
	if !ok {
		return ErrNotFound
	}

	if st.ID != newState.ID {
		return fmt.Errorf("state id mismatch: %s != %s", st.ID, newState.ID)
	}

	if st.UserID != newState.UserID {
		return fmt.Errorf("state user id mismatch: %d != %d", st.UserID, newState.UserID)
	}

	if st.ChatID != newState.ChatID {
		return fmt.Errorf("state chat id mismatch: %d != %d", st.ChatID, newState.ChatID)
	}

	if st == newState {
		return ErrNoChanges
	}

	i.states[st.ID] = newState

	return nil
}
