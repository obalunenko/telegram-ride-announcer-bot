package service

import (
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

// Backends is a set of repositories.
type Backends interface {
	UsersRepository() users.Repository
	SessionsRepository() sessions.Repository
	TripsRepository() trips.Repository
	StatesRepository() states.Repository
}
