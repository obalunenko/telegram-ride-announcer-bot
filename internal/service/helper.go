package service

import (
	"context"
	"fmt"

	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
)

func (s *Service) sendMessage(ctx context.Context, text string) {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return
	}

	msg := tu.Message(tu.ID(sess.ChatID), text)

	_, err := s.bot.Client().SendMessage(msg)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send message")
	}
}

func (s *Service) renderTrip(trip *models.Trip) (string, error) {
	r, err := s.templates.Trip(renderer.TripParams{
		Title:       trip.Name,
		Description: trip.Description,
		Date:        trip.Date,
		CreatedBy:   fmt.Sprintf("@%s", trip.CreatedBy.Username),
	})
	if err != nil {
		return "", err
	}

	return r, nil
}
