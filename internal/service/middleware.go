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

func (s *Service) setContextMiddleware(ctx context.Context) th.Handler {
	return func(c *th.Context, update tgbotapi.Update) error {
		update = update.WithContext(ctx)

		return c.Next(update)
	}
}

func (s *Service) loggerMiddleware() th.Handler {
	return func(c *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")
		} else {
			ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "user_id", sess.User.ID))
			ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", sess.ChatID))
		}

		update = update.WithContext(ctx)

		return c.Next(update)
	}
}

func (s *Service) setSessionMiddleware() th.Handler {
	return func(c *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		// All the handlers serve messages from users, other updates have no one to create a session for.
		if update.Message == nil || update.Message.From == nil {
			log.Debug(ctx, "Skipping update without a message from a user")

			return nil
		}

		uid := update.Message.From.ID
		cid := update.Message.Chat.ID

		if uid == s.bot.ID() {
			// Don't create a session for bot.
			log.WithField(ctx, "user_id", uid).Debug("Bot is trying to create a session for itself")

			return nil
		}

		user, err := ops.GetUser(ctx, s.backends, uid)
		if err != nil {
			if !errors.Is(err, users.ErrNotFound) {
				log.WithError(ctx, err).Error("Failed to get user")

				return nil
			}

			p := ops.CreateUserParams{
				UserID:    uid,
				Username:  update.Message.From.Username,
				Firstname: update.Message.From.FirstName,
				Lastname:  update.Message.From.LastName,
			}

			user, err = ops.CreateUser(ctx, s.backends, p)
			if err != nil {
				log.WithError(ctx, err).Error("Failed to create user")

				return nil
			}
		}

		session, err := ops.GetSession(ctx, s.backends, user)
		if err != nil && !errors.Is(err, sessions.ErrNotFound) {
			log.WithError(ctx, err).Error("Failed to get session")

			return nil
		}

		if session == nil {
			session, err = ops.CreateSession(ctx, s.backends, ops.CreateSessionParams{
				User:   user,
				ChatID: cid,
			})
			if err != nil {
				log.WithError(ctx, err).Error("Failed to create session")

				return nil
			}
		}

		ctx = contextWithSession(ctx, session)

		update = update.WithContext(ctx)

		return c.Next(update)
	}
}

// PanicRecovery is a middleware that will recover handler from panic
func (s *Service) panicRecovery() th.Handler {
	return func(c *th.Context, update tgbotapi.Update) error {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(string(debug.Stack()))

				log.WithField(update.Context(), "error", err).Error("Panic recovered")
			}
		}()

		return c.Next(update)
	}
}
