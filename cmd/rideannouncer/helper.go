package main

import (
	"context"

	tgbotapi "github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"
)

func sendMessage(ctx context.Context, bot *tgbotapi.Bot, chatID int64, text string) {
	msg := tu.Message(tu.ID(chatID), text)

	_, err := bot.SendMessage(msg)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send message")
	}
}
