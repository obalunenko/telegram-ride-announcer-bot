package service

import (
	"context"

	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"
)

func (s *Service) sendMessage(ctx context.Context, text string) {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return
	}

	msg := tu.Message(tu.ID(sess.ChatID), text)

	_, err := s.bot.SendMessage(msg)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send message")
	}
}
