package main

import (
	"context"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"
)

func setContextMiddleware(ctx context.Context) th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}

func getChatIDMiddleware() th.Middleware {
	mu.Lock()
	if chatIDs == nil {
		chatIDs = make(map[int64]struct{})
	}
	mu.Unlock()

	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			ctx := update.Context()

			defer next(bot, update)

			sess := sessionFromContext(ctx)
			if sess == nil {
				log.Error(ctx, "Session is nil")

				return
			}

			mu.RLock()
			cid := sess.chatID
			_, exist := chatIDs[cid]
			mu.RUnlock()

			if !exist {
				mu.Lock()
				chatIDs[cid] = struct{}{}
				mu.Unlock()

				log.Info(ctx, "New chat added")
			}
		}
	}
}

func loggerMiddleware() th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			ctx := update.Context()

			sess := sessionFromContext(ctx)
			if sess == nil {
				log.Error(ctx, "Session is nil")
			} else {
				ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "user_id", sess.user.id))
				ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", sess.chatID))
			}

			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}

func setSessionMiddleware() th.Middleware {
	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			ctx := update.Context()

			uid := update.Message.From.ID
			cid := update.Message.Chat.ID

			sess := getSession(uid)
			if sess == nil {
				sess = newSession(
					newUser(uid, update.Message.From.Username, update.Message.From.FirstName, update.Message.From.LastName),
					cid,
				)
			}
			setSession(sess, uid)
			ctx = contextWithSession(ctx, sess)

			update = update.WithContext(ctx)

			next(bot, update)
		}
	}
}
