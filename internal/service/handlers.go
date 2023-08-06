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
)

func (s *Service) notFoundHandler(ctx context.Context) th.Handler {
	return s.unsupportedHandler(ctx, "Command not found.")
}

func (s *Service) unsupportedHandler(ctx context.Context, text string) th.Handler {
	msg := fmt.Sprintf("%s Use /%s command to see all available commands.", text, cmdHelp)

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "unsupported"))

		log.Debug(ctx, "Called unsupported handler")

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) startHandler(ctx context.Context) th.Handler {
	msgFormat := `Hello, %s! 👋 Welcome to %s. 

I'm here to assist you in scheduling, announcing, and joining exciting bike trips with your community. 

Here are some of the things I can do:

🔹 CreateState new bike trip announcements.
🔹 Display a list of upcoming trips.
🔹 Subscribe you to a bike trip.
🔹 Provide outfit and SPF recommendations based on the weather.
🔹 Show a list of bike trips you've subscribed to.


To get started and see more detailed instructions, use /%s command.

Happy cycling and let's embark on this journey together, %s! 🚴‍♂️
`

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdStart))

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

		msg := fmt.Sprintf(msgFormat, sess.User.Firstname, botName, cmdHelp, sess.User.Firstname)

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) helpHandler() th.Handler {
	msgFormat := `Welcome to %s Help!

Here's a list of commands that you can use: 
%s

Remember, you can always type /%s to see this list of commands again. 

Enjoy planning and going on your bike trips with %s!
`

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdHelp))

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

		cmds, err := bot.GetMyCommands(&tgbotapi.GetMyCommandsParams{})
		if err != nil {
			log.WithError(ctx, err).Error("Failed to get bot commands")
		}

		var cmdsStr string

		for _, cmd := range cmds {
			cmdsStr += fmt.Sprintf("\t/%s - %s\n", cmd.Command, cmd.Description)
		}

		msg := fmt.Sprintf(msgFormat, botName, cmdsStr, cmdHelp, botName)

		s.sendMessage(ctx, msg)
	}
}

func (s *Service) saveSession(ctx context.Context, sess *models.Session) error {
	return ops.UpdateSession(ctx, s.sessions, s.states, sess)
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
	case models.StateNewTrip: // 1. Ask for trip name.
		if sess.UserState.Trip == nil {
			t, err := ops.CreateTrip(ctx, s.trips, ops.CreateTripParams{
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

		// Trip creation is started.
		// Ask for trip name.
		sess.UserState.State = models.StateNewTripName

		msg := "Please enter trip name"

		s.sendMessage(ctx, msg)

		return nil
	case models.StateNewTripName: // 2. Ask for date and time
		// Waiting for trip name.

		name := update.Message.Text
		sess.UserState.Trip.Name = name

		trip, err := ops.UpdateTrip(ctx, s.trips, sess.UserState.Trip.ID, ops.UpdateTripParams{
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

		_, err = s.bot.SendMessage(msg)
		if err != nil {
			log.WithError(ctx, err).Error("Failed to send message")
		}

		return nil

	case models.StateNewTripDate: // 3. Ask for description
		// Waiting for trip date.

		date := update.Message.Text

		trip, err := ops.UpdateTrip(ctx, s.trips, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Date: &date,
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripDescription

		// 2. Ask for description
		msg := "Please enter trip description"

		s.sendMessage(ctx, msg)

		return nil

	case models.StateNewTripDescription: // 4. Ask for a track link. If not provided, ask for track file
		// Waiting for trip description.

		description := update.Message.Text

		trip, err := ops.UpdateTrip(ctx, s.trips, sess.UserState.Trip.ID, ops.UpdateTripParams{
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

		msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf("Your trip %s. Please confirm", trip.String()))

		msg.WithReplyMarkup(keyboard)

		_, err = s.bot.SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil

	case models.StateNewTripConfirm: // 5. Ask for a photo (optional)
		// Waiting for trip confirmation.

		confirm := update.Message.Text

		if confirm == "no" {
			sess.UserState.State = models.StateNewTrip
			err := ops.DeleteTrip(ctx, s.trips, sess.UserState.Trip.ID)
			if err != nil {
				return fmt.Errorf("failed to delete trip: %w", err)
			}

			sess.UserState.Trip = nil

			msg := "Your trip is cancelled. Thank you!"

			s.sendMessage(ctx, msg)

			return nil
		}

		trip, err := ops.UpdateTrip(ctx, s.trips, sess.UserState.Trip.ID, ops.UpdateTripParams{
			Completed: boolPtr(true),
		})
		if err != nil {
			return fmt.Errorf("failed to update trip: %w", err)
		}

		sess.UserState.Trip = trip

		sess.UserState.State = models.StateNewTripPublication

		// Get trip string representation.
		t := sess.UserState.Trip.String()

		// Format message.

		msgtxt := fmt.Sprintf("Your trip %s published. Thank you!", t)

		msg := tu.Message(tu.ID(sess.ChatID), msgtxt)

		resp, err := s.bot.SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// TODO: Check if message really pinned
		err = s.bot.PinChatMessage(&tgbotapi.PinChatMessageParams{
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
			}
		}

		st := sess.UserState.State

		log.WithField(ctx, "UserState", st.String()).Debug("Current Session UserState")

		switch st {
		case models.StateStart:
			// If Session UserState is empty, then we are not in the middle of creating a trip.
			s.notFoundHandler(ctx)(bot, update)

			return
		case models.StateNewTrip, models.StateNewTripName, models.StateNewTripDate, models.StateNewTripTime, models.StateNewTripDescription, models.StateNewTripConfirm, models.StateNewTripPublication:
			s.newTripHandler()(bot, update)

			return
		default:
			s.notFoundHandler(ctx)(bot, update)
		}
	}
}

func (s *Service) tripsHandler() th.Handler {
	return s.notImplementedHandler(cmdTrips)
}

func (s *Service) subscribeHandler() th.Handler {
	return s.notImplementedHandler(cmdSubscribe)
}

func (s *Service) unsubscribeHandler() th.Handler {
	return s.notImplementedHandler(cmdUnsubscribe)
}

func (s *Service) myTripsHandler() th.Handler {
	return s.notImplementedHandler(cmdMyTrips)
}

func (s *Service) subscribedHandler() th.Handler {
	return s.notImplementedHandler(cmdSubscribed)
}

func (s *Service) notImplementedHandler(cmd string) th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmd))

		log.Debug(ctx, "Called not_implemented handler")

		msg := fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", cmdHelp)

		s.sendMessage(ctx, msg)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
