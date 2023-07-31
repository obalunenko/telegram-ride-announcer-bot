package service

import (
	"context"
	"errors"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

func (s *Service) setContextMiddleware(ctx context.Context) th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}

func (s *Service) loggerMiddleware() th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			ctx := update.Context()

			sess := sessionFromContext(ctx)
			if sess == nil {
				log.Error(ctx, "Session is nil")
			} else {
				ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "user_id", sess.User.ID))
				ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", sess.ChatID))
			}

			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}

func (s *Service) setSessionMiddleware() th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			ctx := update.Context()

			uid := update.Message.From.ID
			cid := update.Message.Chat.ID

			user, err := s.users.GetBuID(ctx, uid)
			if err != nil {
				if !errors.Is(err, users.ErrNotFound) {
					log.WithError(ctx, err).Error("Failed to get user by id")

					return
				}
			}

			if user == nil {
				err = s.users.Create(ctx, &users.User{
					ID:        uid,
					Username:  update.Message.From.Username,
					Firstname: update.Message.From.FirstName,
					Lastname:  update.Message.From.LastName,
				})
				if err != nil {
					log.WithError(ctx, err).Error("Failed to create user")

					return
				}
			}

			user, err = s.users.GetBuID(ctx, uid)
			if err != nil {
				log.WithError(ctx, err).Error("Failed to get user by id")

				return
			}

			sess, err := s.sessions.GetSessionByUserID(ctx, uid)
			if err != nil {
				if !errors.Is(err, sessions.ErrNotFound) {
					log.WithError(ctx, err).Error("Failed to get session by user id")

					return
				}
			}

			if sess == nil {
				err = s.sessions.CreateSession(ctx, uid, cid, sessions.State(models.StateStart))
				if err != nil {
					log.WithError(ctx, err).Error("Failed to create session")

					return
				}

				sess, err = s.sessions.GetSessionByUserID(ctx, uid)
				if err != nil {
					log.WithError(ctx, err).Error("Failed to get session by user id")

					return
				}

				log.WithField(ctx, "user_id", uid).Debug("New Session created")
			}

			u := models.NewUser(user.ID, user.Username, user.Firstname, user.Lastname)

			ctx = contextWithSession(ctx, models.NewSession(sess.ID, u, sess.ChatID, models.State(sess.State)))

			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}
