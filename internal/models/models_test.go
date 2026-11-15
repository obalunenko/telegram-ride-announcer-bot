package models_test

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

func TestState_Valid(t *testing.T) {
	tests := []struct {
		name  string
		state models.State
		want  bool
	}{
		{name: "unknown", state: 0, want: false},
		{name: "start", state: models.StateStart, want: true},
		{name: "confirm", state: models.StateNewTripConfirm, want: true},
		{name: "out of range", state: models.State(1000), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.Valid())
		})
	}
}

func TestState_IsAny(t *testing.T) {
	assert.True(t, models.StateNewTripName.IsAny(models.StateStart, models.StateNewTripName))
	assert.False(t, models.StateNewTripName.IsAny(models.StateStart))
	assert.False(t, models.StateNewTripName.IsAny())
}

func TestState_String(t *testing.T) {
	assert.Equal(t, "NewTripName", models.StateNewTripName.String())
}

func TestNewSession(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	user := &models.User{ID: 42}
	state := models.UserState{State: models.StateStart}

	assert.Equal(t, &models.Session{ID: id, User: user, ChatID: 4242, UserState: state},
		models.NewSession(id, user, 4242, state))
}
