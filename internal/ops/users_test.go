package ops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)

	user := createUser(t, b, userID)
	assert.Equal(t, &models.User{ID: userID, Username: "john", Firstname: "John", Lastname: "Doe"}, user)

	got, err := ops.GetUser(ctx, b, userID)
	require.NoError(t, err)
	assert.Equal(t, user, got)

	_, err = ops.CreateUser(ctx, b, ops.CreateUserParams{UserID: userID})
	require.ErrorIs(t, err, users.ErrAlreadyExists)
}

func TestGetUser_NotFound(t *testing.T) {
	_, err := ops.GetUser(context.Background(), newBackends(t), userID)
	require.ErrorIs(t, err, users.ErrNotFound)
}

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		exists  bool
		wantErr error
	}{
		{
			name:   "existing user",
			exists: true,
		},
		{
			name:    "unknown user",
			wantErr: users.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBackends(t)

			if tt.exists {
				createUser(t, b, userID)
			}

			user, err := ops.UpdateUser(ctx, b, ops.UpdateUserParams{
				UserID:    userID,
				Username:  "johnny",
				Firstname: "Johnny",
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, &models.User{ID: userID, Username: "johnny", Firstname: "Johnny"}, user)
		})
	}
}
