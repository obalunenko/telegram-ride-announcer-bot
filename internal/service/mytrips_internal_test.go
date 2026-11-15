package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
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
