package service

import (
	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

func (s *Service) newTripHandler() th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdNewTrip))

		log.Debug(ctx, "Called new_trip handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		tripStates := []models.State{
			models.StateNewTrip,
			models.StateNewTripName,
			models.StateNewTripDate,
			models.StateNewTripTime,
			models.StateNewTripDescription,
			models.StateNewTripConfirm,
		}

		if !sess.UserState.State.IsAny(tripStates...) {
			sess.UserState.State = models.StateNewTrip
			sess.UserState.Trip = nil

			if err := s.saveSession(ctx, sess); err != nil {
				log.WithError(ctx, err).Error("Failed to save session")

				return
			}
		}

		if err := s.createTrip(ctx, update); err != nil {
			log.WithError(ctx, err).Error("Failed to create trip")
		}
	}
}
