package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
)

func TestUpcomingTrips(t *testing.T) {
	upcoming := &models.Trip{Name: "upcoming", Date: testNow.Add(time.Hour), Completed: true}
	past := &models.Trip{Name: "past", Date: testNow.Add(-time.Hour), Completed: true}
	startsNow := &models.Trip{Name: "starts now", Date: testNow, Completed: true}
	draft := &models.Trip{Name: "draft", Date: testNow.Add(time.Hour)}

	assert.Equal(t, []*models.Trip{upcoming}, upcomingTrips([]*models.Trip{draft, past, startsNow, upcoming}, testNow))
	assert.Empty(t, upcomingTrips([]*models.Trip{draft, past}, testNow))
	assert.Empty(t, upcomingTrips(nil, testNow))
}

func TestSortByStart(t *testing.T) {
	created := testNow.Add(-time.Hour)

	later := &models.Trip{Name: "later", Date: testNow.Add(48 * time.Hour), CreatedAt: created}
	sooner := &models.Trip{Name: "sooner", Date: testNow.Add(24 * time.Hour), CreatedAt: created.Add(time.Minute)}
	sameStartFirst := &models.Trip{Name: "same start, created first", Date: testNow.Add(48 * time.Hour), CreatedAt: created.Add(-time.Minute)}

	list := []*models.Trip{later, sooner, sameStartFirst}
	sortByStart(list)

	assert.Equal(t, []*models.Trip{sooner, sameStartFirst, later}, list)
}

func TestSplitMessages(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		limit int
		want  []string
	}{
		{
			name:  "fits in one message",
			parts: []string{"aa", "bb"},
			limit: 10,
			want:  []string{"H:aa\nbb"},
		},
		{
			name:  "split between parts",
			parts: []string{"aaaa", "bbbb", "cc"},
			limit: 10,
			want:  []string{"H:aaaa", "bbbb\ncc"},
		},
		{
			name:  "a long part is not split",
			parts: []string{"aaaaaaaaaaaa"},
			limit: 10,
			want:  []string{"H:aaaaaaaaaaaa"},
		},
		{
			name:  "no parts",
			limit: 10,
			want:  []string{"H:"},
		},
		{
			name: "counted in UTF-16 code units",
			// The emoji takes two code units: "H:🚴" is 4 units, "H:🚴\nb" is 6.
			parts: []string{"🚴", "b"},
			limit: 5,
			want:  []string{"H:🚴", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitMessages("H:", tt.parts, tt.limit))
		})
	}
}

func TestMyTripsHandler_LongList(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)
	sess := newSession(t, repos, testUserID, models.StateStart)

	description := strings.Repeat("a", maxTripDescriptionLen)

	for range 5 {
		trip := newPublishedTrip(t, repos, testUserID)

		_, err := ops.UpdateTrip(ctx, repos, trip.ID, ops.UpdateTripParams{Description: &description})
		require.NoError(t, err)
	}

	require.NoError(t, svc.myTripsHandler()(nil, sessionUpdate(sess, "/mytrips")))

	var cards int

	for range 2 {
		msg := nextMessage(t, api)
		assert.LessOrEqual(t, utf16Len(msg), maxMessageLen)

		cards += strings.Count(msg, "Title: Morning ride")
	}

	assert.Equal(t, 5, cards, "every trip is listed once")
	api.AssertNoRequest(t)
}

func TestMyTripsHandler_UpcomingOnly(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)
	sess := newSession(t, repos, testUserID, models.StateStart)

	newTripAt := func(name string, start time.Time, published bool) {
		t.Helper()

		trip, err := ops.CreateTrip(ctx, repos, ops.CreateTripParams{Name: name, Date: start, CreatedBy: testUserID})
		require.NoError(t, err)

		if published {
			_, err = ops.UpdateTrip(ctx, repos, trip.ID, ops.UpdateTripParams{Completed: &published})
			require.NoError(t, err)
		}
	}

	newTripAt("Later ride", testNow.Add(48*time.Hour), true)
	newTripAt("Past ride", testNow.Add(-time.Hour), true)
	newTripAt("Draft ride", testNow.Add(time.Hour), false)
	newTripAt("Next ride", testNow.Add(2*time.Hour), true)

	require.NoError(t, svc.myTripsHandler()(nil, sessionUpdate(sess, "/mytrips")))

	msg := nextMessage(t, api)
	assert.True(t, strings.HasPrefix(msg, myTripsHeader), msg)
	assert.NotContains(t, msg, "Past ride")
	assert.NotContains(t, msg, "Draft ride")

	next, later := strings.Index(msg, "Title: Next ride"), strings.Index(msg, "Title: Later ride")
	require.NotEqual(t, -1, next)
	require.NotEqual(t, -1, later)
	assert.Less(t, next, later, "the nearest trip comes first")

	api.AssertNoRequest(t)

	// After every trip has started, nothing is listed.
	svc.now = func() time.Time { return testNow.Add(72 * time.Hour) }

	require.NoError(t, svc.myTripsHandler()(nil, sessionUpdate(sess, "/mytrips")))
	assert.Equal(t, msgNoUpcomingTrips, nextMessage(t, api))
}
