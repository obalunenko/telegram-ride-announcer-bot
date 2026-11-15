package service

import (
	"context"
	"strings"

	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
)

// msgSomethingWentWrong is the reply to an update that the bot failed to handle.
const msgSomethingWentWrong = "Sorry, something went wrong. Please try again."

// replyError tells the chat that the bot failed to handle the update.
// It does not need a session, so the middleware can use it too.
func (s *Service) replyError(ctx context.Context, chatID int64) {
	_, err := s.bot.Client().SendMessage(ctx, tu.Message(tu.ID(chatID), msgSomethingWentWrong))
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send error message")
	}
}

func (s *Service) sendMessage(ctx context.Context, text string) {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return
	}

	msg := tu.Message(tu.ID(sess.ChatID), text)

	_, err := s.bot.Client().SendMessage(ctx, msg)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to send message")
	}
}

// creatorName shows the trip creator: the username, or the name when the user has no username.
func creatorName(user *models.User) string {
	if user.Username != "" {
		return "@" + user.Username
	}

	return strings.TrimSpace(user.Firstname + " " + user.Lastname)
}

func (s *Service) renderTrip(trip *models.Trip) (string, error) {
	r, err := s.templates.Trip(renderer.TripParams{
		Title:       trip.Name,
		Description: trip.Description,
		Date:        formatTripDate(trip.Date, s.location),
		CreatedBy:   creatorName(trip.CreatedBy),
	})
	if err != nil {
		return "", err
	}

	return r, nil
}
