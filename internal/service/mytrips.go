package service

import (
	"slices"

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

// sortByStart orders the trips by start, the nearest first.
// Trips with the same start keep the order in which they were created.
func sortByStart(list []*models.Trip) {
	slices.SortStableFunc(list, func(a, b *models.Trip) int {
		if c := a.Date.Compare(b.Date); c != 0 {
			return c
		}

		return a.CreatedAt.Compare(b.CreatedAt)
	})
}
