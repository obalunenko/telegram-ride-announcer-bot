package main

import (
	"context"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"
)

func getChatIDMiddleware(ctx context.Context) th.Middleware {
	mu.Lock()
	if chatIDs == nil {
		chatIDs = make(map[int64]struct{})
	}
	mu.Unlock()

	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			chatID := update.Message.Chat.ID

			mu.RLock()
			_, exist := chatIDs[chatID]
			mu.RUnlock()

			if !exist {
				mu.Lock()
				chatIDs[chatID] = struct{}{}
				mu.Unlock()

				log.WithField(ctx, "chat_id", chatID).Info("New chat added")
			}

			next(bot, update)
		}
	}
}
