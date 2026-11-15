package service

import (
	"slices"
	"time"
	"unicode/utf16"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

// upcomingTrips keeps the published trips that start after now. Drafts of unfinished
// dialogs and past trips are not listed.
func upcomingTrips(list []*models.Trip, now time.Time) []*models.Trip {
	upcoming := make([]*models.Trip, 0, len(list))

	for _, trip := range list {
		if trip.Completed && trip.Date.After(now) {
			upcoming = append(upcoming, trip)
		}
	}

	return upcoming
}

const (
	// msgNoUpcomingTrips is the reply to /mytrips without upcoming trips.
	msgNoUpcomingTrips = "You have no upcoming trips. Use /" + CmdNewTrip + " command to create a new trip."
	// myTripsHeader starts the /mytrips list.
	myTripsHeader = "Your upcoming trips:\n\n"
)

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

// maxMessageLen is the Telegram limit of a message text, in UTF-16 code units.
const maxMessageLen = 4096

// splitMessages joins the parts into messages that fit the limit. The first message starts
// with the header, parts are separated by a newline, and a part is never split.
func splitMessages(header string, parts []string, limit int) []string {
	var msgs []string

	cur, hasPart := header, false

	for _, part := range parts {
		sep := ""
		if hasPart {
			sep = "\n"
		}

		if hasPart && utf16Len(cur+sep+part) > limit {
			msgs = append(msgs, cur)
			cur, sep = "", ""
		}

		cur += sep + part
		hasPart = true
	}

	if cur != "" {
		msgs = append(msgs, cur)
	}

	return msgs
}

// utf16Len returns the length of the text in UTF-16 code units, as Telegram counts it.
func utf16Len(s string) int {
	var n int

	for _, r := range s {
		n += max(utf16.RuneLen(r), 1)
	}

	return n
}
