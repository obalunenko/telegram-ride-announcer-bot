package service

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

func (s *Service) setContextMiddleware(ctx context.Context) th.Middleware {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update, next th.Handler) {
		update = update.WithContext(ctx)

		next(bot, update)
	}
}

func (s *Service) loggerMiddleware() th.Middleware {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update, next th.Handler) {
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

func (s *Service) setSessionMiddleware() th.Middleware {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update, next th.Handler) {
		ctx := update.Context()

		uid := update.Message.From.ID
		cid := update.Message.Chat.ID

		if uid == s.bot.ID() {
			// Don't create a session for bot.
			log.WithField(ctx, "user_id", uid).Debug("Bot is trying to create a session for itself")

			return
		}

		user, err := ops.GetUser(ctx, s.users, uid)
		if err != nil {
			if errors.Is(err, users.ErrNotFound) {
				user, err = ops.CreateUser(ctx, s.users, uid, update.Message.From.Username, update.Message.From.FirstName, update.Message.From.LastName)
				if err != nil {
					log.WithError(ctx, err).Error("Failed to create user")

					return
				}
			}
		}

		session, err := ops.GetSession(ctx, s.sessions, s.states, s.trips, user)
		if err != nil && !errors.Is(err, sessions.ErrNotFound) {
			log.WithError(ctx, err).Error("Failed to get session")

			return
		}

		if session == nil {
			session, err = ops.CreateSession(ctx, s.sessions, s.states, s.trips, ops.CreateSessionParams{
				User:   user,
				ChatID: cid,
			})
			if err != nil {
				log.WithError(ctx, err).Error("Failed to create session")

				return
			}
		}

		ctx = contextWithSession(ctx, session)

		update = update.WithContext(ctx)

		next(bot, update)
	}
}

// PanicRecovery is a middleware that will recover handler from panic
func (s *Service) panicRecovery() th.Middleware {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update, next th.Handler) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(string(debug.Stack()))

				log.WithError(update.Context(), err.(error)).Error("Panic recovered")
			}
		}()

		next(bot, update)
	}
}
