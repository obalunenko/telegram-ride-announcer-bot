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

func TestPublishedTrips(t *testing.T) {
	published := &models.Trip{Name: "Morning ride", Completed: true}
	draft := &models.Trip{Name: "Evening ride"}

	assert.Equal(t, []*models.Trip{published}, publishedTrips([]*models.Trip{draft, published}))
	assert.Empty(t, publishedTrips([]*models.Trip{draft}))
	assert.Empty(t, publishedTrips(nil))
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
