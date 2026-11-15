package service

import (
	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

// msgNothingToCancel is the reply to /cancel outside of a trip dialog.
const msgNothingToCancel = "There is nothing to cancel."

// newTripHandler starts the trip dialog. It discards the draft of an unfinished dialog first.
func (s *Service) newTripHandler() th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdNewTrip))

		log.Debug(ctx, "Called new_trip handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		if _, err := s.resetDialog(ctx, sess, models.StateNewTrip); err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")

			return nil
		}

		if err := s.createTrip(ctx, update); err != nil {
			log.WithError(ctx, err).Error("Failed to create trip")
		}

		return nil
	}
}

// tripAnswerHandler handles a text answer to the current step of the trip dialog.
func (s *Service) tripAnswerHandler() th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdNewTrip))

		if err := s.createTrip(ctx, update); err != nil {
			log.WithError(ctx, err).Error("Failed to create trip")
		}

		return nil
	}
}

// cancelHandler ends the trip dialog and discards its draft.
func (s *Service) cancelHandler() th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdCancel))

		log.Debug(ctx, "Called cancel handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		discarded, err := s.resetDialog(ctx, sess, models.StateStart)
		if err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")

			return nil
		}

		if !discarded {
			s.sendMessage(ctx, msgNothingToCancel)
		}

		return nil
	}
}
