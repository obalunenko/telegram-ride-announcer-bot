package service

import (
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

// backends is a set of repositories.
type backends interface {
	UsersRepository() users.Repository
	SessionsRepository() sessions.Repository
	TripsRepository() trips.Repository
	StatesRepository() states.Repository
}
