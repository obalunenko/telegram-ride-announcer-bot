package states_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
)

const (
	userID  int64 = 42
	chatID  int64 = 4242
	otherID int64 = 4343
)

func TestInMemory_CreateState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   bool
		chatID  int64
		wantErr error
	}{
		{
			name:   "new state",
			chatID: chatID,
		},
		{
			name:   "same user in another chat",
			setup:  true,
			chatID: otherID,
		},
		{
			name:    "same user in the same chat",
			setup:   true,
			chatID:  chatID,
			wantErr: states.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := states.NewInMemory()

			var existing *states.State

			if tt.setup {
				var err error

				existing, err = repo.CreateState(ctx, states.CreateParams{UserID: userID, ChatID: chatID, Step: 1})
				require.NoError(t, err)
			}

			state, err := repo.CreateState(ctx, states.CreateParams{UserID: userID, ChatID: tt.chatID, Step: 2})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, existing, state, "the existing state is returned")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.chatID, state.ChatID)

			got, err := repo.GetStateByID(ctx, state.ID)
			require.NoError(t, err)
			assert.Equal(t, state, got)
		})
	}
}

func TestInMemory_UpdateState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		update  func(st states.State) *states.State
		wantErr string
	}{
		{
			name: "step changed",
			update: func(st states.State) *states.State {
				st.Step = 3

				return &st
			},
		},
		{
			name: "unknown state",
			update: func(st states.State) *states.State {
				st.ID = [16]byte{1}

				return &st
			},
			wantErr: states.ErrNotFound.Error(),
		},
		{
			name: "another user",
			update: func(st states.State) *states.State {
				st.UserID = otherID

				return &st
			},
			wantErr: "state user id mismatch",
		},
		{
			name: "another chat",
			update: func(st states.State) *states.State {
				st.ChatID = otherID

				return &st
			},
			wantErr: "state chat id mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := states.NewInMemory()

			state, err := repo.CreateState(ctx, states.CreateParams{UserID: userID, ChatID: chatID, Step: 1})
			require.NoError(t, err)

			updated := tt.update(*state)

			err = repo.UpdateState(ctx, updated)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)

			got, err := repo.GetStateByID(ctx, state.ID)
			require.NoError(t, err)
			assert.Equal(t, updated, got)
		})
	}
}

func TestInMemory_UpdateState_NoChanges(t *testing.T) {
	ctx := context.Background()
	repo := states.NewInMemory()

	state, err := repo.CreateState(ctx, states.CreateParams{UserID: userID, ChatID: chatID})
	require.NoError(t, err)

	require.ErrorIs(t, repo.UpdateState(ctx, state), states.ErrNoChanges)
}

func TestInMemory_GetStateByID_NotFound(t *testing.T) {
	_, err := states.NewInMemory().GetStateByID(context.Background(), [16]byte{1})
	require.ErrorIs(t, err, states.ErrNotFound)
}
