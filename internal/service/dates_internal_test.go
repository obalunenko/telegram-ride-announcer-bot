package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTripDate(t *testing.T) {
	tbilisi, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	tests := []struct {
		name    string
		text    string
		now     time.Time
		loc     *time.Location
		want    time.Time
		wantErr error
	}{
		{
			name: "today",
			text: "today",
			now:  testNow,
			loc:  time.UTC,
			want: time.Date(2026, time.November, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "tomorrow in another case",
			text: " Tomorrow ",
			now:  testNow,
			loc:  time.UTC,
			want: time.Date(2026, time.November, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "date",
			text: "01.12.2026",
			now:  testNow,
			loc:  time.UTC,
			want: time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "today in the bot time zone",
			text: "today",
			// 22:00 UTC is already the next day in Tbilisi (UTC+4).
			now:  time.Date(2026, time.November, 15, 22, 0, 0, 0, time.UTC),
			loc:  tbilisi,
			want: time.Date(2026, time.November, 16, 0, 0, 0, 0, tbilisi),
		},
		{
			name:    "past date",
			text:    "14.11.2026",
			now:     testNow,
			loc:     time.UTC,
			wantErr: errDateInPast,
		},
		{
			name:    "not a date",
			text:    "next week",
			now:     testNow,
			loc:     time.UTC,
			wantErr: errInvalidDate,
		},
		{
			name:    "impossible date",
			text:    "32.13.2026",
			now:     testNow,
			loc:     time.UTC,
			wantErr: errInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTripDate(tt.text, tt.now, tt.loc)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "got %s, want %s", got, tt.want)
		})
	}
}

func TestParseTripTime(t *testing.T) {
	today := time.Date(2026, time.November, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		text    string
		day     time.Time
		want    time.Time
		wantErr error
	}{
		{
			name: "later today",
			text: "09:30",
			day:  today,
			want: time.Date(2026, time.November, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name: "one-digit hour",
			text: "9:05",
			day:  today,
			want: time.Date(2026, time.November, 15, 9, 5, 0, 0, time.UTC),
		},
		{
			name: "earlier tomorrow",
			text: "07:00",
			day:  today.AddDate(0, 0, 1),
			want: time.Date(2026, time.November, 16, 7, 0, 0, 0, time.UTC),
		},
		{
			name:    "earlier today",
			text:    "07:00",
			day:     today,
			wantErr: errTimeInPast,
		},
		{
			name:    "now",
			text:    "08:00",
			day:     today,
			wantErr: errTimeInPast,
		},
		{
			name:    "not a time",
			text:    "7 pm",
			day:     today,
			wantErr: errInvalidTime,
		},
		{
			name:    "impossible time",
			text:    "25:00",
			day:     today,
			wantErr: errInvalidTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTripTime(tt.text, tt.day, testNow, time.UTC)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "got %s, want %s", got, tt.want)
		})
	}
}

func TestFormatTripDate(t *testing.T) {
	tbilisi, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	start := time.Date(2026, time.November, 15, 9, 30, 0, 0, time.UTC)

	assert.Equal(t, "Sun, 15 Nov 2026 09:30 UTC", formatTripDate(start, time.UTC))
	assert.Equal(t, "Sun, 15 Nov 2026 13:30 +04", formatTripDate(start, tbilisi))
	assert.Equal(t, "not set", formatTripDate(time.Time{}, time.UTC))
}
