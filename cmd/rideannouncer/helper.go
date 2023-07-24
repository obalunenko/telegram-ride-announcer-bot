package main

import (
	"context"

	tgbotapi "github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"
)

func sendMessage(ctx context.Context, bot *tgbotapi.Bot, text string) {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return
	}

	msg := tu.Message(tu.ID(sess.chatID), text)

	_, err := bot.SendMessage(msg)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send message")
	}
}
