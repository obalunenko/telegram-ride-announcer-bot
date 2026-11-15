package ops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
)

func TestCreateSession(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	sess, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)
	assert.Equal(t, user, sess.User)
	assert.Equal(t, chatID, sess.ChatID)
	assert.Equal(t, models.StateStart, sess.UserState.State)
	assert.Nil(t, sess.UserState.Trip)

	_, err = ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.ErrorIs(t, err, sessions.ErrAlreadyExists)
}

func TestGetSession_NotFound(t *testing.T) {
	b := newBackends(t)
	user := createUser(t, b, userID)

	_, err := ops.GetSession(context.Background(), b, user, chatID)
	require.ErrorIs(t, err, sessions.ErrNotFound)
}

func TestGetSession_WithoutState(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	require.NoError(t, b.SessionsRepository().CreateSession(ctx, userID, chatID, nil))

	_, err := ops.GetSession(ctx, b, user, chatID)
	require.ErrorContains(t, err, "has no state")
}

func TestUpdateSession(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	sess, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)

	trip, err := ops.CreateTrip(ctx, b, ops.CreateTripParams{Name: "Morning ride", CreatedBy: userID})
	require.NoError(t, err)

	sess.UserState.State = models.StateNewTripName
	sess.UserState.Trip = trip

	require.NoError(t, ops.UpdateSession(ctx, b, sess))

	got, err := ops.GetSession(ctx, b, user, chatID)
	require.NoError(t, err)
	assert.Equal(t, models.StateNewTripName, got.UserState.State)
	require.NotNil(t, got.UserState.Trip)
	assert.Equal(t, trip.ID, got.UserState.Trip.ID)
}

func TestSessionsPerChat(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	first, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)

	first.UserState.State = models.StateNewTripDate
	require.NoError(t, ops.UpdateSession(ctx, b, first))

	second, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: otherID})
	require.NoError(t, err)
	assert.Equal(t, models.StateStart, second.UserState.State, "a new chat starts its own dialog")
	assert.NotEqual(t, first.UserState.ID, second.UserState.ID)

	list, err := ops.ListSessions(ctx, b)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	require.NoError(t, ops.DeleteSession(ctx, b, second))

	_, err = ops.GetSession(ctx, b, user, otherID)
	require.ErrorIs(t, err, sessions.ErrNotFound)

	got, err := ops.GetSession(ctx, b, user, chatID)
	require.NoError(t, err)
	assert.Equal(t, models.StateNewTripDate, got.UserState.State)
}

func TestCreateSession_KeepsStateOfChat(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	sess, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)

	sess.UserState.State = models.StateNewTripName
	require.NoError(t, ops.UpdateSession(ctx, b, sess))
	require.NoError(t, ops.DeleteSession(ctx, b, sess))

	again, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)
	assert.Equal(t, sess.UserState.ID, again.UserState.ID)
	assert.Equal(t, models.StateNewTripName, again.UserState.State)
}

func TestListSessions_SkipsUnknownUser(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	_, err := ops.CreateSession(ctx, b, ops.CreateSessionParams{User: user, ChatID: chatID})
	require.NoError(t, err)

	// A session record of a user that is not stored.
	require.NoError(t, b.SessionsRepository().CreateSession(ctx, otherID, chatID, nil))

	list, err := ops.ListSessions(ctx, b)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, userID, list[0].User.ID)
}

func TestDeleteSession_NotFound(t *testing.T) {
	err := ops.DeleteSession(context.Background(), newBackends(t), &models.Session{
		User:   &models.User{ID: userID},
		ChatID: chatID,
	})
	require.ErrorIs(t, err, sessions.ErrNotFound)
}
