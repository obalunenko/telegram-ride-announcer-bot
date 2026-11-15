package service

import (
	"testing"

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
