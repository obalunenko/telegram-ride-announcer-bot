package users_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

const userID int64 = 42

func TestInMemory_Create(t *testing.T) {
	ctx := context.Background()
	repo := users.NewInMemory()
	user := &users.User{ID: userID, Username: "john", Firstname: "John", Lastname: "Doe"}

	require.NoError(t, repo.Create(ctx, user))
	require.ErrorIs(t, repo.Create(ctx, user), users.ErrAlreadyExists)

	got, err := repo.GetBuID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestInMemory_GetBuID_NotFound(t *testing.T) {
	_, err := users.NewInMemory().GetBuID(context.Background(), userID)
	require.ErrorIs(t, err, users.ErrNotFound)
}

func TestInMemory_Update(t *testing.T) {
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
			repo := users.NewInMemory()

			if tt.exists {
				require.NoError(t, repo.Create(ctx, &users.User{ID: userID, Username: "old_john"}))
			}

			updated := &users.User{ID: userID, Username: "john", Firstname: "John"}

			err := repo.Update(ctx, updated)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)

			got, err := repo.GetBuID(ctx, userID)
			require.NoError(t, err)
			assert.Equal(t, updated, got)
		})
	}
}
