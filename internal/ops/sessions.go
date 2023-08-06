package ops

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid/v5"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
)

func GetSession(ctx context.Context, sessRepo sessions.Repository, statesRepo states.Repository, tripsRepo trips.Repository, user *models.User) (*models.Session, error) {
	// Check if user exists.
	sess, err := sessRepo.GetSessionByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session by user id[%d]: %w", user.ID, err)
	}

	state, err := statesRepo.GetStateByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state by user id[%d]: %w", user.ID, err)
	}

	var trip *models.Trip

	if state.TripID != nil {
		trip, err = GetTrip(ctx, tripsRepo, *state.TripID)
		if err != nil {
			return nil, fmt.Errorf("failed to get trip by id[%s]: %w", state.TripID, err)
		}
	}

	var resp *models.Session

	resp = &models.Session{
		ID:     sess.ID,
		User:   user,
		ChatID: sess.ChatID,
		UserState: models.UserState{
			ID:    state.ID,
			State: models.State(state.Step),
			Trip:  trip,
		},
	}

	return resp, nil
}

type CreateSessionParams struct {
	User   *models.User
	ChatID int64
}

func CreateSession(ctx context.Context, sessRepo sessions.Repository, statesRepo states.Repository, tripsRepo trips.Repository, p CreateSessionParams) (*models.Session, error) {
	// check if state exists
	state, err := statesRepo.GetStateByUserID(ctx, p.User.ID)
	if err != nil && !errors.Is(err, states.ErrNotFound) {
		return nil, fmt.Errorf("failed to get state by user id[%d]: %w", p.User.ID, err)
	}

	if state == nil {
		// create session and state
		// Create state
		state, err = statesRepo.CreateState(ctx, states.CreateParams{
			UserID:  p.User.ID,
			Command: "",
			TripID:  nil,
			Step:    uint(models.StateStart),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create state: %w", err)
		}
	}

	// Create session
	err = sessRepo.CreateSession(ctx, p.User.ID, p.ChatID, &state.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"user_id": p.User.ID,
		"chat_id": p.ChatID,
	}).Debug("New session created")

	return GetSession(ctx, sessRepo, statesRepo, tripsRepo, p.User)
}

func UpdateSession(ctx context.Context, sessRepo sessions.Repository, statesRepo states.Repository, sess *models.Session) error {
	uid := sess.User.ID
	var tid *uuid.UUID

	if sess.UserState.Trip != nil {
		tid = &sess.UserState.Trip.ID
	}

	// update state
	err := statesRepo.UpdateState(ctx, &states.State{
		ID:      sess.UserState.ID,
		UserID:  uid,
		TripID:  tid,
		Command: "",
		Step:    uint(sess.UserState.State),
	})
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// update session
	err = sessRepo.UpdateSession(ctx, &sessions.Session{
		ID:      sess.ID,
		UserID:  uid,
		ChatID:  sess.ChatID,
		StateID: &sess.UserState.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"user_id":          uid,
		"chat_id":          sess.ChatID,
		"session_id":       sess.ID,
		"current_state":    sess.UserState.State.String(),
		"trip_in_progress": sess.UserState.Trip != nil,
	}).Debug("Session updated")

	return nil
}
