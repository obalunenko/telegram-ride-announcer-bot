package service

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
)

func (s *Service) notFoundHandler(ctx context.Context) th.Handler {
	return s.unsupportedHandler(ctx, "Command not found.")
}

func (s *Service) unsupportedHandler(ctx context.Context, text string) th.Handler {
	msg := fmt.Sprintf("%s Use /%s command to see all available commands.", text, CmdHelp)

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "unsupported"))

		log.Debug(ctx, "Called unsupported handler")

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) startHandler(ctx context.Context) th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdStart))

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		// Reset Session UserState.
		sess.UserState.State = models.StateStart

		if err := s.saveSession(ctx, sess); err != nil {
			log.WithError(ctx, err).Error("Failed to update session")

			return
		}

		log.Debug(ctx, "Called start handler")

		msg, err := s.templates.Welcome(renderer.WelcomeParams{
			Firstname:   sess.User.Firstname,
			BotUsername: s.bot.Username(),
			HelpCmd:     fmt.Sprintf("/%s", CmdHelp),
		})
		if err != nil {
			log.WithError(ctx, err).Error("Failed to render template")

			return
		}

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) helpHandler() th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdHelp))

		log.Debug(ctx, "Called help handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		// Reset Session UserState.
		sess.UserState.State = models.StateStart

		if err := s.saveSession(ctx, sess); err != nil {
			log.WithError(ctx, err).Error("Failed to update session")

			return
		}

		var cmdsStr string

		for _, cmd := range s.bot.Commands() {
			cmdsStr += fmt.Sprintf("\t/%s - %s\n", cmd.Command, cmd.Description)
		}

		msg, err := s.templates.Help(renderer.HelpParams{
			BotUsername: s.bot.Username(),
			Commands:    cmdsStr,
			HelpCmd:     fmt.Sprintf("/%s", CmdHelp),
		})
		if err != nil {
			log.WithError(ctx, err).Error("Failed to render template")

			return
		}

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) saveSession(ctx context.Context, sess *models.Session) error {
	return ops.UpdateSession(ctx, s.backends, sess)
}

func (s *Service) createTrip(ctx context.Context, update tgbotapi.Update) error {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return fmt.Errorf("session is nil")
	}

	log.WithField(ctx, "UserState", sess.UserState.State.String()).Debug("Current Session UserState")

	defer func() {
		log.WithField(ctx, "UserState", sess.UserState.State.String()).Debug("Saving Session")

		if err := s.saveSession(ctx, sess); err != nil {
			log.WithError(ctx, err).Error("Failed to update session")
		}
	}()

	// 1. Ask for trip name.

	// 2. Ask for date and time

	// 3. Ask for description

	// 4. Ask for a track link. If not provided, ask for track file

	// 5. Ask for a photo (optional)

	// 6. Ask for confirmation

	// 7. Subscribe the creator to the trip (maybe implement it later)

	// 8. Announce the trip.

	// 9. Pin the trip announcement to the chat.

	switch sess.UserState.State {
	case models.StateNewTrip:
		if sess.UserState.Trip == nil {
			t, err := ops.CreateTrip(ctx, s.backends, ops.CreateTripParams{
				Name:        "",
				Date:        "",
				Description: "",
				CreatedBy:   sess.User.ID,
			})
			if err != nil {
				log.WithError(ctx, err).Error("Failed to create trip")

				return err
			}

			sess.UserState.Trip = t
		}

		sess.UserState.State = models.StateNewTripName

		msg := "Please enter trip name"

		s.sendMessage(ctx, msg)

		return nil
	case models.StateNewTripName:
		name := update.Message.Text
		sess.UserState.Trip.Name = name

		trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Name: &name,
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripDate

		// 1. Ask for date and time
		keyboard := tu.Keyboard(
			tu.KeyboardRow(
				tu.KeyboardButton("today"),
			),
			tu.KeyboardRow(
				tu.KeyboardButton("tomorrow"),
			),
		).WithResizeKeyboard().WithInputFieldPlaceholder("Enter date").WithOneTimeKeyboard()

		msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf("Your trip name %q. Please select date and time", name))

		msg.WithReplyMarkup(keyboard)

		_, err = s.bot.Client().SendMessage(msg)
		if err != nil {
			log.WithError(ctx, err).Error("Failed to send message")
		}

		return nil

	case models.StateNewTripDate:
		date := update.Message.Text

		trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Date: &date,
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripDescription

		msg := "Please enter trip description"

		s.sendMessage(ctx, msg)

		return nil

	case models.StateNewTripDescription:
		description := update.Message.Text

		trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Description: &description,
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripConfirm

		keyboard := tu.Keyboard(
			tu.KeyboardRow(
				tu.KeyboardButton("yes"),
			),
			tu.KeyboardRow(
				tu.KeyboardButton("no"),
			),
		).WithResizeKeyboard().WithInputFieldPlaceholder("Confirm").WithOneTimeKeyboard()

		if sess.User.ID != trip.CreatedBy.ID {
			return fmt.Errorf("user %d is not the creator of the trip %d", sess.User.ID, trip.ID)
		}

		tripfmt, err := s.templates.Trip(renderer.TripParams{
			Title:       trip.Name,
			Description: trip.Description,
			Date:        trip.Date,
			CreatedBy:   fmt.Sprintf("@%s", sess.User.Username),
		})
		if err != nil {
			return fmt.Errorf("failed to render trip: %w", err)
		}

		msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf("%s\n\nPlease confirm", tripfmt))

		msg.WithReplyMarkup(keyboard)

		_, err = s.bot.Client().SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil

	case models.StateNewTripConfirm:
		confirm := update.Message.Text

		if confirm == "no" {
			sess.UserState.State = models.StateNewTrip

			if err := ops.DeleteTrip(ctx, s.backends, sess.UserState.Trip.ID); err != nil {
				return fmt.Errorf("failed to delete trip: %w", err)
			}

			sess.UserState.Trip = nil

			msg := "Your trip is canceled. Thank you!"

			s.sendMessage(ctx, msg)

			return nil
		}

		trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Completed: boolPtr(true),
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripPublish

		if sess.User.ID != trip.CreatedBy.ID {
			return fmt.Errorf("user %d is not the creator of the trip %d", sess.User.ID, trip.ID)
		}

		tripfmt, err := s.templates.Trip(renderer.TripParams{
			Title:       trip.Name,
			Description: trip.Description,
			Date:        trip.Date,
			CreatedBy:   fmt.Sprintf("@%s", sess.User.Username),
		})

		msgtxt := fmt.Sprintf("Trip is published!\n\n%s", tripfmt)

		msg := tu.Message(tu.ID(sess.ChatID), msgtxt)

		resp, err := s.bot.Client().SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// TODO: Check if message really pinned
		err = s.bot.Client().PinChatMessage(&tgbotapi.PinChatMessageParams{
			ChatID:              tu.ID(resp.Chat.ID),
			MessageID:           resp.MessageID,
			DisableNotification: false,
		})
		if err != nil {
			return fmt.Errorf("failed to pin message: %w", err)
		}

		sess.UserState.State = models.StateStart
		sess.UserState.Trip = nil

		return nil

	default:
		log.WithField(ctx, "UserState", sess.UserState.State.String()).Error("Unexpected UserState")

		return nil
	}
}

func (s *Service) textHandler() th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "text"))

		log.Debug(ctx, "Called text handler")

		// Check Session UserState.
		// If the Session UserState is not empty, then we are in the middle of creating a trip.
		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		if update.Message != nil {
			if strings.HasPrefix(update.Message.Text, "/") {
				// TODO: Handle commands.
			}
		}

		st := sess.UserState.State

		log.WithField(ctx, "UserState", st.String()).Debug("Current Session UserState")

		switch st {
		case models.StateStart:
			// If Session UserState is empty, then we are not in the middle of creating a trip.
			s.notFoundHandler(ctx)(bot, update)

			return
		case models.StateNewTrip,
			models.StateNewTripDate,
			models.StateNewTripName,
			models.StateNewTripTime,
			models.StateNewTripDescription,
			models.StateNewTripConfirm,
			models.StateNewTripPublish:
			s.newTripHandler()(bot, update)

			return
		default:
			s.notFoundHandler(ctx)(bot, update)
		}
	}
}

func (s *Service) tripsHandler() th.Handler {
	return s.notImplementedHandler(CmdTrips)
}

func (s *Service) subscribeHandler() th.Handler {
	return s.notImplementedHandler(CmdSubscribe)
}

func (s *Service) unsubscribeHandler() th.Handler {
	return s.notImplementedHandler(CmdUnsubscribe)
}

func (s *Service) myTripsHandler() th.Handler {
	return s.notImplementedHandler(CmdMyTrips)
}

func (s *Service) subscribedHandler() th.Handler {
	return s.notImplementedHandler(CmdSubscribed)
}

func (s *Service) notImplementedHandler(cmd string) th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmd))

		log.Debug(ctx, "Called not_implemented handler")

		msg := fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", CmdHelp)

		s.sendMessage(ctx, msg)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
