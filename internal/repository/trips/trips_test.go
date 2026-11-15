package trips_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
)

const (
	userID  int64 = 42
	otherID int64 = 43
)

// start is the start of the test trips.
var start = time.Date(2026, time.November, 16, 9, 30, 0, 0, time.UTC)

func ptr[T any](v T) *T {
	return &v
}

func TestInMemory_CreateTrip(t *testing.T) {
	ctx := context.Background()
	repo := trips.NewInMemory()

	trip, err := repo.CreateTrip(ctx, "Morning ride", start, "Easy pace", userID)
	require.NoError(t, err)
	assert.Equal(t, "Morning ride", trip.Name)
	assert.Equal(t, start, trip.Date)
	assert.Equal(t, "Easy pace", trip.Description)
	assert.Equal(t, userID, trip.CreatedBy)
	assert.False(t, trip.Completed)

	got, err := repo.GetTripByID(ctx, trip.ID)
	require.NoError(t, err)
	assert.Equal(t, trip, got)
}

func TestInMemory_UpdateTrip(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		params trips.UpdateTripParams
		check  func(t *testing.T, trip *trips.Trip)
	}{
		{
			name:   "name",
			params: trips.UpdateTripParams{Name: ptr("Evening ride")},
			check: func(t *testing.T, trip *trips.Trip) {
				assert.Equal(t, "Evening ride", trip.Name)
				assert.Equal(t, "Easy pace", trip.Description, "other fields are kept")
			},
		},
		{
			name:   "date",
			params: trips.UpdateTripParams{Date: ptr(start.AddDate(0, 0, 1))},
			check: func(t *testing.T, trip *trips.Trip) {
				assert.Equal(t, start.AddDate(0, 0, 1), trip.Date)
			},
		},
		{
			name:   "description",
			params: trips.UpdateTripParams{Description: ptr("Fast pace")},
			check: func(t *testing.T, trip *trips.Trip) {
				assert.Equal(t, "Fast pace", trip.Description)
			},
		},
		{
			name:   "completed",
			params: trips.UpdateTripParams{Completed: ptr(true)},
			check: func(t *testing.T, trip *trips.Trip) {
				assert.True(t, trip.Completed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := trips.NewInMemory()

			trip, err := repo.CreateTrip(ctx, "Morning ride", start, "Easy pace", userID)
			require.NoError(t, err)

			require.NoError(t, repo.UpdateTrip(ctx, trip.ID, tt.params))

			got, err := repo.GetTripByID(ctx, trip.ID)
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}

func TestInMemory_DeleteTrip(t *testing.T) {
	ctx := context.Background()
	repo := trips.NewInMemory()

	trip, err := repo.CreateTrip(ctx, "Morning ride", start, "Easy pace", userID)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteTrip(ctx, trip.ID))

	_, err = repo.GetTripByID(ctx, trip.ID)
	require.ErrorIs(t, err, trips.ErrNotFound)
	require.ErrorIs(t, repo.DeleteTrip(ctx, trip.ID), trips.ErrNotFound, "a deleted trip can't be deleted again")
	require.ErrorIs(t, repo.UpdateTrip(ctx, trip.ID, trips.UpdateTripParams{}), trips.ErrNotFound)
}

func TestInMemory_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := trips.NewInMemory()
	id := uuid.Must(uuid.NewV4())

	_, err := repo.GetTripByID(ctx, id)
	require.ErrorIs(t, err, trips.ErrNotFound)
	require.ErrorIs(t, repo.UpdateTrip(ctx, id, trips.UpdateTripParams{}), trips.ErrNotFound)
	require.ErrorIs(t, repo.DeleteTrip(ctx, id), trips.ErrNotFound)
}

func TestInMemory_ListTrips(t *testing.T) {
	ctx := context.Background()
	repo := trips.NewInMemory()

	mine, err := repo.CreateTrip(ctx, "Morning ride", start, "", userID)
	require.NoError(t, err)

	deleted, err := repo.CreateTrip(ctx, "Canceled ride", start, "", userID)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteTrip(ctx, deleted.ID))

	other, err := repo.CreateTrip(ctx, "Other ride", start, "", otherID)
	require.NoError(t, err)

	all, err := repo.ListTrips(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*trips.Trip{mine, other}, all)

	byUser, err := repo.ListTripsByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []*trips.Trip{mine}, byUser)
}
