package sessions_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
)

const (
	userID  int64 = 42
	chatID  int64 = 4242
	otherID int64 = 4343
)

func TestInMemory_CreateSession(t *testing.T) {
	ctx := context.Background()
	stateID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		setup   func(t *testing.T, repo sessions.Repository)
		chatID  int64
		wantErr error
	}{
		{
			name:   "new session",
			chatID: chatID,
		},
		{
			name: "same user in another chat",
			setup: func(t *testing.T, repo sessions.Repository) {
				require.NoError(t, repo.CreateSession(ctx, userID, chatID, &stateID))
			},
			chatID: otherID,
		},
		{
			name: "same user in the same chat",
			setup: func(t *testing.T, repo sessions.Repository) {
				require.NoError(t, repo.CreateSession(ctx, userID, chatID, &stateID))
			},
			chatID:  chatID,
			wantErr: sessions.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := sessions.NewInMemory()

			if tt.setup != nil {
				tt.setup(t, repo)
			}

			err := repo.CreateSession(ctx, userID, tt.chatID, &stateID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)

			sess, err := repo.GetSession(ctx, userID, tt.chatID)
			require.NoError(t, err)
			assert.Equal(t, userID, sess.UserID)
			assert.Equal(t, tt.chatID, sess.ChatID)
			assert.Equal(t, &stateID, sess.StateID)
		})
	}
}

func TestInMemory_SessionsPerChat(t *testing.T) {
	ctx := context.Background()
	repo := sessions.NewInMemory()
	stateID := uuid.Must(uuid.NewV4())

	require.NoError(t, repo.CreateSession(ctx, userID, chatID, &stateID))
	require.NoError(t, repo.CreateSession(ctx, userID, otherID, &stateID))

	list, err := repo.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	_, err = repo.GetSession(ctx, otherID, chatID)
	require.ErrorIs(t, err, sessions.ErrNotFound)

	sess, err := repo.GetSession(ctx, userID, otherID)
	require.NoError(t, err)

	newStateID := uuid.Must(uuid.NewV4())
	sess.StateID = &newStateID

	require.NoError(t, repo.UpdateSession(ctx, sess))

	got, err := repo.GetSession(ctx, userID, otherID)
	require.NoError(t, err)
	assert.Equal(t, &newStateID, got.StateID)

	first, err := repo.GetSession(ctx, userID, chatID)
	require.NoError(t, err)
	assert.Equal(t, &stateID, first.StateID)

	require.NoError(t, repo.DeleteSession(ctx, userID, otherID))

	_, err = repo.GetSession(ctx, userID, otherID)
	require.ErrorIs(t, err, sessions.ErrNotFound)

	_, err = repo.GetSession(ctx, userID, chatID)
	require.NoError(t, err)
}

func TestInMemory_UpdateAndDeleteErrors(t *testing.T) {
	ctx := context.Background()
	repo := sessions.NewInMemory()

	require.Error(t, repo.UpdateSession(ctx, nil))
	require.ErrorIs(t, repo.UpdateSession(ctx, &sessions.Session{UserID: userID, ChatID: chatID}), sessions.ErrNotFound)
	require.ErrorIs(t, repo.DeleteSession(ctx, userID, chatID), sessions.ErrNotFound)
}
