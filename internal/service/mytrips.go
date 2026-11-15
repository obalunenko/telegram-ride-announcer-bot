package service

import (
	"slices"
	"unicode/utf16"

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
