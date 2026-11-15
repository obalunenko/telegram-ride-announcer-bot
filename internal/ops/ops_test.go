package ops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/backends"
)

const (
	userID  int64 = 42
	chatID  int64 = 4242
	otherID int64 = 4343
)

// newBackends returns in-memory repositories.
func newBackends(t *testing.T) *backends.Backends {
	t.Helper()

	b, err := backends.New(backends.NewParams{
		Users:    users.NewInMemory(),
		States:   states.NewInMemory(),
		Sessions: sessions.NewInMemory(),
		Trips:    trips.NewInMemory(),
	})
	require.NoError(t, err)

	return b
}

// createUser stores the test user.
func createUser(t *testing.T, b *backends.Backends, id int64) *models.User {
	t.Helper()

	user, err := ops.CreateUser(context.Background(), b, ops.CreateUserParams{
		UserID:    id,
		Username:  "john",
		Firstname: "John",
		Lastname:  "Doe",
	})
	require.NoError(t, err)

	return user
}
