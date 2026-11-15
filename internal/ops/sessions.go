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
)

// GetSession returns the session of the user in the chat.
func GetSession(ctx context.Context, b backends, user *models.User, chatID int64) (*models.Session, error) {
	sess, err := b.SessionsRepository().GetSession(ctx, user.ID, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session of user id[%d] in chat id[%d]: %w", user.ID, chatID, err)
	}

	return toSession(ctx, b, user, sess)
}

// toSession loads the state and the trip of the session record.
func toSession(ctx context.Context, b backends, user *models.User, sess *sessions.Session) (*models.Session, error) {
	if sess.StateID == nil {
		return nil, fmt.Errorf("session id[%s] has no state", sess.ID)
	}

	state, err := b.StatesRepository().GetStateByID(ctx, *sess.StateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state by id[%s]: %w", sess.StateID, err)
	}

	var trip *models.Trip

	if state.TripID != nil {
		trip, err = GetTrip(ctx, b, *state.TripID)
		if err != nil {
			return nil, fmt.Errorf("failed to get trip by id[%s]: %w", state.TripID, err)
		}
	}

	return &models.Session{
		ID:     sess.ID,
		User:   user,
		ChatID: sess.ChatID,
		UserState: models.UserState{
			ID:    state.ID,
			State: models.State(state.Step),
			Trip:  trip,
		},
	}, nil
}

// CreateSessionParams is a params for CreateSession function.
type CreateSessionParams struct {
	User   *models.User
	ChatID int64
}

// CreateSession creates a new session of the user in the chat.
func CreateSession(ctx context.Context, b backends, p CreateSessionParams) (*models.Session, error) {
	// The user keeps the state of the chat when the session is created again.
	state, err := b.StatesRepository().CreateState(ctx, states.CreateParams{
		UserID: p.User.ID,
		ChatID: p.ChatID,
		TripID: nil,
		Step:   uint(models.StateStart),
	})
	if err != nil && !errors.Is(err, states.ErrAlreadyExists) {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	// Create session
	err = b.SessionsRepository().CreateSession(ctx, p.User.ID, p.ChatID, &state.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"user_id": p.User.ID,
		"chat_id": p.ChatID,
	}).Debug("New session created")

	return GetSession(ctx, b, p.User, p.ChatID)
}

// UpdateSession updates session.
func UpdateSession(ctx context.Context, b backends, sess *models.Session) error {
	uid := sess.User.ID

	var tid *uuid.UUID

	if sess.UserState.Trip != nil {
		tid = &sess.UserState.Trip.ID
	}

	// update state
	err := b.StatesRepository().UpdateState(ctx, &states.State{
		ID:     sess.UserState.ID,
		UserID: uid,
		ChatID: sess.ChatID,
		TripID: tid,
		Step:   uint(sess.UserState.State),
	})
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// update session
	err = b.SessionsRepository().UpdateSession(ctx, &sessions.Session{
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

// ListSessions returns list of sessions.
func ListSessions(ctx context.Context, b backends) ([]*models.Session, error) {
	list, err := b.SessionsRepository().ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var resp []*models.Session

	for _, sess := range list {
		user, err := GetUser(ctx, b, sess.UserID)
		if err != nil {
			log.WithError(ctx, err).WithFields(log.Fields{
				"user_id": sess.UserID,
			}).Warn("Failed to get user by id")

			continue
		}

		s, err := toSession(ctx, b, user, sess)
		if err != nil {
			return nil, err
		}

		resp = append(resp, s)
	}

	return resp, nil
}

// DeleteSession deletes session.
func DeleteSession(ctx context.Context, b backends, sess *models.Session) error {
	err := b.SessionsRepository().DeleteSession(ctx, sess.User.ID, sess.ChatID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"user_id": sess.User.ID,
		"chat_id": sess.ChatID,
	}).Debug("Session deleted")

	return nil
}
