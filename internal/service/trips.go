package service

import (
	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
)

func (s *Service) newTripHandler() th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdNewTrip))

		log.Debug(ctx, "Called new_trip handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		defer func() {
			err := s.sessions.UpdateSession(ctx, &sessions.Session{
				ID:     sess.ID,
				UserID: sess.User.ID,
				ChatID: sess.ChatID,
				State:  sessions.State(sess.State),
			})
			if err != nil {
				log.WithError(ctx, err).Error("Failed to update session")

				return
			}
		}()

		if sess.State.IsAny(models.StateStart) {
			sess.State = models.StateNewTrip

			err := s.sessions.UpdateSession(ctx, &sessions.Session{
				ID:     sess.ID,
				UserID: sess.User.ID,
				ChatID: sess.ChatID,
				State:  sessions.State(sess.State),
			})
			if err != nil {
				log.WithError(ctx, err).Error("Failed to update session")

				return
			}
		}

		if err := s.createTrip(ctx, update); err != nil {
			log.WithError(ctx, err).Error("Failed to create trip")
		}
	}
}
