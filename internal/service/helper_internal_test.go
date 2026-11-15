package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

func TestCreatorName(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
		want string
	}{
		{
			name: "username",
			user: &models.User{Username: "john", Firstname: "John", Lastname: "Doe"},
			want: "@john",
		},
		{
			name: "no username",
			user: &models.User{Firstname: "John", Lastname: "Doe"},
			want: "John Doe",
		},
		{
			name: "first name only",
			user: &models.User{Firstname: "John"},
			want: "John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, creatorName(tt.user))
		})
	}
}

func TestTripText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
		wantOK bool
	}{
		{name: "trimmed", text: "  Morning ride  ", maxLen: 100, want: "Morning ride", wantOK: true},
		{name: "spaces only", text: "   ", maxLen: 100, want: "", wantOK: false},
		{name: "at the limit in characters", text: "привет", maxLen: 6, want: "привет", wantOK: true},
		{name: "over the limit", text: "abcdefg", maxLen: 6, want: "abcdefg", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tripText(tt.text, tt.maxLen)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
