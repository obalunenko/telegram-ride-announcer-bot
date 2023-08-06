package models

import (
	"encoding/json"
	"time"

	uuid "github.com/gofrs/uuid/v5"
)

type (
	ChatID = int64
	UserID = int64
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

func NewUser(id int64, username, firstname, lastname string) *User {
	return &User{
		ID:        id,
		Username:  username,
		Firstname: firstname,
		Lastname:  lastname,
	}
}

// Session represents a session.
type Session struct {
	ID        uuid.UUID `json:"id,omitempty"`
	User      *User     `json:"user,omitempty"`
	ChatID    ChatID    `json:"chat_id,omitempty"`
	UserState UserState `json:"state,omitempty"`
}

type UserState struct {
	ID    uuid.UUID
	State State
	Trip  *Trip
}

func NewUserState(id uuid.UUID, state State, trip *Trip) UserState {
	return UserState{
		ID:    id,
		State: state,
		Trip:  trip,
	}
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
	StateNewTripPublication // Waiting for publication

	stateSentinel // Sentinel value.
)

func (s State) Valid() bool {
	return s > stateUnknown && s < stateSentinel
}

func NewSession(id uuid.UUID, user *User, chatID ChatID, state UserState) *Session {
	return &Session{
		ID:        id,
		User:      user,
		ChatID:    chatID,
		UserState: state,
	}
}

func (s State) IsAny(states ...State) bool {
	for _, state := range states {
		if s == state {
			return true
		}
	}

	return false
}
