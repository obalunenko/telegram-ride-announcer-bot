// Package models contains models.
package models

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
)

type (
	// ChatID is a chat ID.
	ChatID = int64
	// UserID is a user ID.
	UserID = int64
	// TripID is a trip ID.
	TripID = uuid.UUID
)

// Trip represents a trip.
type Trip struct {
	ID          TripID    `json:"ID,omitempty"`
	Name        string    `json:"name,omitempty"`
	Date        string    `json:"date,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	CreatedBy   UserID    `json:"created_by,omitempty"`
}

func (t Trip) String() string {
	var s string

	v, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		panic(err)
	}

	s = string(v)

	return s
}

// User represents a user.
type User struct {
	ID        UserID `json:"id,omitempty"`
	Username  string `json:"username,omitempty"`
	Firstname string `json:"firstname,omitempty"`
	Lastname  string `json:"lastname,omitempty"`
}

// Session represents a session.
type Session struct {
	ID        uuid.UUID `json:"id,omitempty"`
	User      *User     `json:"user,omitempty"`
	ChatID    ChatID    `json:"chat_id,omitempty"`
	UserState UserState `json:"state,omitempty"`
}

// UserState represents a user state.
type UserState struct {
	ID    uuid.UUID
	State State
	Trip  *Trip
}

//go:generate stringer -type=State -output=state_string.go -trimprefix=State

// State represents a state.
type State uint

const (
	stateUnknown State = iota

	// StateStart is the start state.
	StateStart
	// StateNewTrip means that a new trip is just started.
	StateNewTrip
	// StateNewTripName means that trip on progress and waiting for trip name
	StateNewTripName
	// StateNewTripDate means that trip on progress and waiting for trip date
	StateNewTripDate
	// StateNewTripTime means that trip on progress and waiting for trip time
	StateNewTripTime
	// StateNewTripDescription means that trip on progress and waiting for trip description
	StateNewTripDescription
	// StateNewTripConfirm means that trip on progress and waiting for trip confirmation
	StateNewTripConfirm
	// StateNewTripPublish means that trip on progress and waiting for trip publish
	StateNewTripPublish

	stateSentinel // Sentinel value.
)

// Valid checks if state is valid.
func (s State) Valid() bool {
	return s > stateUnknown && s < stateSentinel
}

// NewSession creates a new session.
func NewSession(id uuid.UUID, user *User, chatID ChatID, state UserState) *Session {
	return &Session{
		ID:        id,
		User:      user,
		ChatID:    chatID,
		UserState: state,
	}
}

// IsAny checks if state is any of given states.
func (s State) IsAny(states ...State) bool {
	for _, state := range states {
		if s == state {
			return true
		}
	}

	return false
}
