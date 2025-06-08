package service

import (
	"fmt"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

func (s *Service) newTripHandler() th.Handler {
	return func(tctx *th.Context, update tgbotapi.Update) error {
		ctx := tctx.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdNewTrip))

		log.Debug(ctx, "Called new_trip handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			return fmt.Errorf("session is nil")
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
				return fmt.Errorf("failed to save session: %w", err)
			}
		}

		if err := s.createTrip(ctx, update); err != nil {
			return fmt.Errorf("failed to create trip: %w", err)
		}

		return nil
	}
}
