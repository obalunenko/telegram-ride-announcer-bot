// Package backends provide a set of repositories.
package backends

import (
	"errors"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

// Backends is a set of repositories.
type Backends struct {
	users    users.Repository
	states   states.Repository
	sessions sessions.Repository
	trips    trips.Repository
}

// UsersRepository returns users repository.
func (b *Backends) UsersRepository() users.Repository {
	return b.users
}

// StatesRepository returns states repository.
func (b *Backends) StatesRepository() states.Repository {
	return b.states
}

// SessionsRepository returns sessions repository.
func (b *Backends) SessionsRepository() sessions.Repository {
	return b.sessions
}

// TripsRepository returns trips repository.
func (b *Backends) TripsRepository() trips.Repository {
	return b.trips
}

// NewParams is a params for New function.
type NewParams struct {
	Users    users.Repository
	States   states.Repository
	Sessions sessions.Repository
	Trips    trips.Repository
}

// New creates a new Backends.
func New(p NewParams) (*Backends, error) {
	if p.Users == nil {
		return nil, errors.New("users repository is required")
	}

	if p.States == nil {
		return nil, errors.New("states repository is required")
	}

	if p.Sessions == nil {
		return nil, errors.New("sessions repository is required")
	}

	if p.Trips == nil {
		return nil, errors.New("trips repository is required")
	}

	return &Backends{
		users:    p.Users,
		states:   p.States,
		sessions: p.Sessions,
		trips:    p.Trips,
	}, nil
}
