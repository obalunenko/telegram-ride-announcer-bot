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

	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "unsupported"))

		log.Debug(ctx, "Called unsupported handler")

		s.sendMessage(ctx, msg)

		return nil
	}
}

func (s *Service) startHandler(ctx context.Context) th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdStart))

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		if err := s.resetDialog(ctx, sess); err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		log.Debug(ctx, "Called start handler")

		msg, err := s.templates.Welcome(renderer.WelcomeParams{
			Firstname:   sess.User.Firstname,
			BotUsername: s.bot.Username(),
			HelpCmd:     fmt.Sprintf("/%s", CmdHelp),
		})
		if err != nil {
			log.WithError(ctx, err).Error("Failed to render template")
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		s.sendMessage(ctx, msg)

		return nil
	}
}

func (s *Service) helpHandler() th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdHelp))

		log.Debug(ctx, "Called help handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		if err := s.resetDialog(ctx, sess); err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")
			s.replyError(ctx, sess.ChatID)

			return nil
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
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		s.sendMessage(ctx, msg)

		return nil
	}
}

func (s *Service) saveSession(ctx context.Context, sess *models.Session) error {
	return ops.UpdateSession(ctx, s.backends, sess)
}

// msgDraftDiscarded tells the user that a command interrupted their trip dialog.
const msgDraftDiscarded = "Your unfinished trip is discarded."

// discardDraft deletes the draft trip of an unfinished trip dialog.
// It reports whether there was a draft; the caller saves the session.
func (s *Service) discardDraft(ctx context.Context, sess *models.Session) (bool, error) {
	if sess.UserState.Trip == nil {
		return false, nil
	}

	if err := ops.DeleteTrip(ctx, s.backends, sess.UserState.Trip.ID); err != nil {
		return false, fmt.Errorf("failed to delete trip: %w", err)
	}

	sess.UserState.Trip = nil

	return true, nil
}

// resetDialog ends any dialog of the session: it discards the draft trip, tells the user
// about it, and saves the session in StateStart.
func (s *Service) resetDialog(ctx context.Context, sess *models.Session) error {
	discarded, err := s.discardDraft(ctx, sess)
	if err != nil {
		return err
	}

	sess.UserState.State = models.StateStart

	if err = s.saveSession(ctx, sess); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	if discarded {
		s.sendMessage(ctx, msgDraftDiscarded)
	}

	return nil
}

func (s *Service) createTrip(ctx context.Context, update tgbotapi.Update) error {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return fmt.Errorf("session is nil")
	}

	log.WithField(ctx, "UserState", sess.UserState.State.String()).Debug("Current Session UserState")

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
		return s.startTrip(ctx, sess)
	case models.StateNewTripName:
		return s.setTripName(ctx, sess, update.Message.Text)
	case models.StateNewTripDate:
		return s.setTripDate(ctx, sess, update.Message.Text)
	case models.StateNewTripDescription:
		return s.setTripDescription(ctx, sess, update.Message.Text)
	case models.StateNewTripConfirm:
		return s.confirmTrip(ctx, sess, update.Message.Text)
	default:
		log.WithField(ctx, "UserState", sess.UserState.State.String()).Error("Unexpected UserState")

		return nil
	}
}

// replyInTrip saves the session and only then sends the reply of a trip step:
// the user answers the reply, and the answer must find the session in the new state.
func (s *Service) replyInTrip(
	ctx context.Context,
	sess *models.Session,
	msg *tgbotapi.SendMessageParams,
) (*tgbotapi.Message, error) {
	log.WithField(ctx, "UserState", sess.UserState.State.String()).Debug("Saving Session")

	if err := s.saveSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	resp, err := s.bot.Client().SendMessage(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return resp, nil
}

// startTrip creates a draft trip, if the session has none yet, and asks for the trip name.
func (s *Service) startTrip(ctx context.Context, sess *models.Session) error {
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

	msg := tu.Message(tu.ID(sess.ChatID), "Please enter trip name")

	_, err := s.replyInTrip(ctx, sess, msg)

	return err
}

// setTripName saves the trip name and asks for the trip date.
func (s *Service) setTripName(ctx context.Context, sess *models.Session, name string) error {
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

	_, err = s.replyInTrip(ctx, sess, msg)

	return err
}

// setTripDate saves the trip date and asks for the trip description.
func (s *Service) setTripDate(ctx context.Context, sess *models.Session, date string) error {
	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Date: &date,
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	sess.UserState.Trip = trip

	sess.UserState.State = models.StateNewTripDescription

	msg := tu.Message(tu.ID(sess.ChatID), "Please enter trip description")

	_, err = s.replyInTrip(ctx, sess, msg)

	return err
}

// setTripDescription saves the trip description and asks to confirm the trip.
func (s *Service) setTripDescription(ctx context.Context, sess *models.Session, description string) error {
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

	tripfmt, err := s.renderTrip(trip)
	if err != nil {
		return fmt.Errorf("failed to render trip: %w", err)
	}

	msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf("%s\n\nPlease confirm", tripfmt))

	msg.WithReplyMarkup(keyboard)

	_, err = s.replyInTrip(ctx, sess, msg)

	return err
}

// confirmTrip cancels the trip on "no" and publishes it on any other answer.
func (s *Service) confirmTrip(ctx context.Context, sess *models.Session, confirm string) error {
	if confirm == "no" {
		return s.cancelTrip(ctx, sess)
	}

	return s.publishTrip(ctx, sess)
}

// cancelTrip deletes the draft trip and lets the user start a new one.
func (s *Service) cancelTrip(ctx context.Context, sess *models.Session) error {
	if err := ops.DeleteTrip(ctx, s.backends, sess.UserState.Trip.ID); err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}

	sess.UserState.State = models.StateNewTrip
	sess.UserState.Trip = nil

	msg := tu.Message(tu.ID(sess.ChatID), "Your trip is canceled. Thank you!")

	_, err := s.replyInTrip(ctx, sess, msg)

	return err
}

// publishTrip completes the trip, announces it to the chat and pins the announcement.
func (s *Service) publishTrip(ctx context.Context, sess *models.Session) error {
	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Completed: boolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	if sess.User.ID != trip.CreatedBy.ID {
		return fmt.Errorf("user %d is not the creator of the trip %d", sess.User.ID, trip.ID)
	}

	tripfmt, err := s.renderTrip(trip)
	if err != nil {
		return fmt.Errorf("failed to render trip: %w", err)
	}

	// The trip is complete, the user can start a new one.
	sess.UserState.State = models.StateStart
	sess.UserState.Trip = nil

	msgtxt := fmt.Sprintf("Trip is published!\n\n%s", tripfmt)

	msg := tu.Message(tu.ID(sess.ChatID), msgtxt)

	resp, err := s.replyInTrip(ctx, sess, msg)
	if err != nil {
		return err
	}

	// TODO: Check if message really pinned
	err = s.bot.Client().PinChatMessage(ctx, &tgbotapi.PinChatMessageParams{
		ChatID:              tu.ID(resp.Chat.ID),
		MessageID:           resp.MessageID,
		DisableNotification: false,
	})
	if err != nil {
		return fmt.Errorf("failed to pin message: %w", err)
	}

	return nil
}

func (s *Service) textHandler() th.Handler {
	return func(c *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "text"))

		log.Debug(ctx, "Called text handler")

		// Check Session UserState.
		// If the Session UserState is not empty, then we are in the middle of creating a trip.
		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		if update.Message != nil {
			if strings.HasPrefix(update.Message.Text, "/") {
				// TODO: Handle commands.
				log.WithField(ctx, "command", update.Message.Text).Debug("Command detected")
			}
		}

		st := sess.UserState.State

		log.WithField(ctx, "UserState", st.String()).Debug("Current Session UserState")

		switch st {
		case models.StateStart:
			// If Session UserState is empty, then we are not in the middle of creating a trip.
			return s.notFoundHandler(ctx)(c, update)
		case models.StateNewTrip,
			models.StateNewTripDate,
			models.StateNewTripName,
			models.StateNewTripTime,
			models.StateNewTripDescription,
			models.StateNewTripConfirm,
			models.StateNewTripPublish:
			return s.newTripHandler()(c, update)
		default:
			return s.notFoundHandler(ctx)(c, update)
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
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", CmdMyTrips))

		log.Debug(ctx, "Called my_trips handler")

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return nil
		}

		list, err := ops.ListTripsByUser(ctx, s.backends, sess.User)
		if err != nil {
			log.WithError(ctx, err).WithField("user_id", sess.User.ID).Error("Failed to list trips")

			return nil
		}

		if len(list) == 0 {
			msg := fmt.Sprintf("You have no trips yet. Use /%s command to create a new trip.", CmdNewTrip)

			s.sendMessage(ctx, msg)

			return nil
		}

		msgtxt := "Your trips:\n\n"

		for i := range list {
			if i > 0 {
				msgtxt += "\n"
			}

			trip := list[i]

			tripfmt, err := s.renderTrip(trip)
			if err != nil {
				log.WithError(ctx, err).WithField("trip_id", trip.ID).Error("Failed to render trip")

				return nil
			}

			msgtxt += tripfmt
		}

		s.sendMessage(ctx, msgtxt)

		return nil
	}
}

func (s *Service) subscribedHandler() th.Handler {
	return s.notImplementedHandler(CmdSubscribed)
}

func (s *Service) notImplementedHandler(cmd string) th.Handler {
	return func(_ *th.Context, update tgbotapi.Update) error {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmd))

		log.Debug(ctx, "Called not_implemented handler")

		msg := fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", CmdHelp)

		s.sendMessage(ctx, msg)

		return nil
	}
}

func boolPtr(b bool) *bool {
	return &b
}
