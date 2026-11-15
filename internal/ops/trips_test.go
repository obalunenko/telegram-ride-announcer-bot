package ops_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

func TestCreateTrip(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)

	trip, err := ops.CreateTrip(ctx, b, ops.CreateTripParams{
		Name:        "Morning ride",
		Date:        "today",
		Description: "Easy pace",
		CreatedBy:   userID,
	})
	require.NoError(t, err)
	assert.Equal(t, "Morning ride", trip.Name)
	assert.Equal(t, "today", trip.Date)
	assert.Equal(t, "Easy pace", trip.Description)
	assert.Equal(t, user, trip.CreatedBy)
	assert.False(t, trip.Completed)
}

func TestGetTrip_Errors(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)

	_, err := ops.GetTrip(ctx, b, uuid.Must(uuid.NewV4()))
	require.ErrorIs(t, err, trips.ErrNotFound)

	// The creator of the trip is not stored.
	trip, err := b.TripsRepository().CreateTrip(ctx, "Morning ride", "today", "", userID)
	require.NoError(t, err)

	_, err = ops.GetTrip(ctx, b, trip.ID)
	require.ErrorIs(t, err, users.ErrNotFound)
}

func TestUpdateTrip(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	createUser(t, b, userID)

	trip, err := ops.CreateTrip(ctx, b, ops.CreateTripParams{Name: "Morning ride", CreatedBy: userID})
	require.NoError(t, err)

	name, completed := "Evening ride", true

	updated, err := ops.UpdateTrip(ctx, b, trip.ID, ops.UpdateTripParams{Name: &name, Completed: &completed})
	require.NoError(t, err)
	assert.Equal(t, "Evening ride", updated.Name)
	assert.True(t, updated.Completed, "publishing marks the trip completed")

	_, err = ops.UpdateTrip(ctx, b, uuid.Must(uuid.NewV4()), ops.UpdateTripParams{Name: &name})
	require.ErrorIs(t, err, trips.ErrNotFound)
}

func TestDeleteTrip(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	createUser(t, b, userID)

	trip, err := ops.CreateTrip(ctx, b, ops.CreateTripParams{Name: "Morning ride", CreatedBy: userID})
	require.NoError(t, err)

	require.NoError(t, ops.DeleteTrip(ctx, b, trip.ID))

	_, err = ops.GetTrip(ctx, b, trip.ID)
	require.ErrorIs(t, err, trips.ErrNotFound)
	require.ErrorIs(t, ops.DeleteTrip(ctx, b, trip.ID), trips.ErrNotFound)
}

func TestListTripsByUser(t *testing.T) {
	ctx := context.Background()
	b := newBackends(t)
	user := createUser(t, b, userID)
	createUser(t, b, otherID)

	trip, err := ops.CreateTrip(ctx, b, ops.CreateTripParams{Name: "Morning ride", CreatedBy: userID})
	require.NoError(t, err)

	completed := true

	_, err = ops.UpdateTrip(ctx, b, trip.ID, ops.UpdateTripParams{Completed: &completed})
	require.NoError(t, err)

	_, err = ops.CreateTrip(ctx, b, ops.CreateTripParams{Name: "Other ride", CreatedBy: otherID})
	require.NoError(t, err)

	list, err := ops.ListTripsByUser(ctx, b, user)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Morning ride", list[0].Name)
	assert.Equal(t, user, list[0].CreatedBy)
	assert.True(t, list[0].Completed)
}
