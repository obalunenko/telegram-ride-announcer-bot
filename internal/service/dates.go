package service

import (
	"errors"
	"strings"
	"time"
)

const (
	// tripDateLayout is the layout of a date that the user enters.
	tripDateLayout = "02.01.2006"
	// tripTimeLayout is the layout of a start time that the user enters.
	tripTimeLayout = "15:04"
	// tripCardLayout shows the trip start in the trip card.
	tripCardLayout = "Mon, 02 Jan 2006 15:04 MST"

	dateToday    = "today"
	dateTomorrow = "tomorrow"
)

var (
	errInvalidDate = errors.New("invalid date")
	errDateInPast  = errors.New("date in the past")
	errInvalidTime = errors.New("invalid time")
	errTimeInPast  = errors.New("time in the past")
)

// parseTripDate returns the start of the day that the user entered: today, tomorrow, or DD.MM.YYYY.
// The day must not be in the past.
func parseTripDate(text string, now time.Time, loc *time.Location) (time.Time, error) {
	now = now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var day time.Time

	switch v := strings.ToLower(strings.TrimSpace(text)); v {
	case dateToday:
		day = today
	case dateTomorrow:
		day = today.AddDate(0, 0, 1)
	default:
		d, err := time.ParseInLocation(tripDateLayout, v, loc)
		if err != nil {
			return time.Time{}, errInvalidDate
		}

		day = d
	}

	if day.Before(today) {
		return time.Time{}, errDateInPast
	}

	return day, nil
}

// parseTripTime returns the trip start: the day at the HH:MM time that the user entered.
// The start must be in the future.
func parseTripTime(text string, day, now time.Time, loc *time.Location) (time.Time, error) {
	t, err := time.Parse(tripTimeLayout, strings.TrimSpace(text))
	if err != nil {
		return time.Time{}, errInvalidTime
	}

	day = day.In(loc)

	start := time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	if !start.After(now) {
		return time.Time{}, errTimeInPast
	}

	return start, nil
}

// formatTripDate shows the trip start in the time zone of the bot.
func formatTripDate(date time.Time, loc *time.Location) string {
	if date.IsZero() {
		return "not set"
	}

	return date.In(loc).Format(tripCardLayout)
}
