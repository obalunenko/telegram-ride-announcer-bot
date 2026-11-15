package service

import (
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

// publishedTrips keeps the published trips; drafts of unfinished dialogs are not listed.
func publishedTrips(list []*models.Trip) []*models.Trip {
	published := make([]*models.Trip, 0, len(list))

	for _, trip := range list {
		if trip.Completed {
			published = append(published, trip)
		}
	}

	return published
}
