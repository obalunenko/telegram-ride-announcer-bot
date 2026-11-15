package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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

		if _, err := s.resetDialog(ctx, sess, models.StateStart); err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		log.Debug(ctx, "Called start handler")

		msg, err := s.templates.Welcome(renderer.WelcomeParams{
			Firstname:   sess.User.Firstname,
			BotUsername: "@" + s.bot.Username(),
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

		if _, err := s.resetDialog(ctx, sess, models.StateStart); err != nil {
			log.WithError(ctx, err).Error("Failed to reset dialog")
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		var cmdsStr string

		for _, cmd := range s.bot.Menu() {
			cmdsStr += fmt.Sprintf("\t/%s - %s\n", cmd.Command, cmd.Description)
		}

		msg, err := s.templates.Help(renderer.HelpParams{
			BotUsername: "@" + s.bot.Username(),
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

// resetDialog ends any dialog of the session: it discards the draft trip, saves the session
// in the state, and tells the user when a draft was discarded. It reports whether it was.
func (s *Service) resetDialog(ctx context.Context, sess *models.Session, state models.State) (bool, error) {
	discarded, err := s.discardDraft(ctx, sess)
	if err != nil {
		return false, err
	}

	sess.UserState.State = state

	if err = s.saveSession(ctx, sess); err != nil {
		return false, fmt.Errorf("failed to update session: %w", err)
	}

	if discarded {
		s.sendMessage(ctx, msgDraftDiscarded)
	}

	return discarded, nil
}

func (s *Service) createTrip(ctx context.Context, update tgbotapi.Update) error {
	sess := sessionFromContext(ctx)
	if sess == nil {
		log.Error(ctx, "Session is nil")

		return fmt.Errorf("session is nil")
	}

	log.WithField(ctx, "UserState", sess.UserState.State.String()).Debug("Current Session UserState")

	// Steps: name, date, time, description, confirmation; then the announcement is pinned.
	switch sess.UserState.State {
	case models.StateNewTrip:
		return s.startTrip(ctx, sess)
	case models.StateNewTripName:
		return s.setTripName(ctx, sess, update.Message.Text)
	case models.StateNewTripDate:
		return s.setTripDate(ctx, sess, update.Message.Text)
	case models.StateNewTripTime:
		return s.setTripTime(ctx, sess, update.Message.Text)
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

// msgEnterTripName starts the trip dialog and tells how to stop it.
const msgEnterTripName = "Please enter trip name. Send /cancel to stop."

// startTrip creates a draft trip, if the session has none yet, and asks for the trip name.
func (s *Service) startTrip(ctx context.Context, sess *models.Session) error {
	if sess.UserState.Trip == nil {
		t, err := ops.CreateTrip(ctx, s.backends, ops.CreateTripParams{
			Name:        "",
			Date:        time.Time{},
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

	msg := tu.Message(tu.ID(sess.ChatID), msgEnterTripName)

	_, err := s.replyInTrip(ctx, sess, msg)

	return err
}

const (
	// maxTripNameLen and maxTripDescriptionLen limit the trip texts, in characters.
	maxTripNameLen        = 100
	maxTripDescriptionLen = 1000

	msgInvalidTripName = "Please enter a trip name of 1 to 100 characters."
	msgInvalidTripDesc = "Please enter a trip description of 1 to 1000 characters."
)

// tripText trims the text and reports whether it has 1 to maxLen characters.
func tripText(text string, maxLen int) (string, bool) {
	text = strings.TrimSpace(text)
	n := utf8.RuneCountInString(text)

	return text, n > 0 && n <= maxLen
}

// setTripName saves the trip name and asks for the trip date.
// It asks again when the name is empty or too long.
func (s *Service) setTripName(ctx context.Context, sess *models.Session, text string) error {
	name, ok := tripText(text, maxTripNameLen)
	if !ok {
		_, err := s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), msgInvalidTripName))

		return err
	}

	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Name: &name,
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	sess.UserState.Trip = trip

	sess.UserState.State = models.StateNewTripDate

	msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf(msgEnterTripDate, name)).WithReplyMarkup(dateKeyboard())

	_, err = s.replyInTrip(ctx, sess, msg)

	return err
}

const (
	// msgEnterTripDate asks for the trip date after the name.
	msgEnterTripDate = "Your trip name %q. Please select the date or enter it as DD.MM.YYYY."
	msgInvalidDate   = "I can't read this date. Please select today or tomorrow, or enter the date as DD.MM.YYYY."
	msgDateInPast    = "This date has already passed. Please enter today or a later date."
	msgEnterTripTime = "Please enter the start time as HH:MM, for example 09:30."
	msgInvalidTime   = "I can't read this time. Please enter it as HH:MM, for example 09:30."
	msgTimeInPast    = "This time has already passed. Please enter a later time."
	msgEnterTripDesc = "Please enter trip description"
)

// dateKeyboard offers the nearest trip dates.
func dateKeyboard() *tgbotapi.ReplyKeyboardMarkup {
	return tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton(dateToday),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(dateTomorrow),
		),
	).WithResizeKeyboard().WithInputFieldPlaceholder("DD.MM.YYYY").WithOneTimeKeyboard()
}

// setTripDate saves the trip day and asks for the start time.
// It asks again when the date can't be read or is in the past.
func (s *Service) setTripDate(ctx context.Context, sess *models.Session, text string) error {
	day, err := parseTripDate(text, s.now(), s.location)
	if err != nil {
		reply := msgInvalidDate
		if errors.Is(err, errDateInPast) {
			reply = msgDateInPast
		}

		_, err = s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), reply).WithReplyMarkup(dateKeyboard()))

		return err
	}

	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Date: &day,
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	sess.UserState.Trip = trip

	sess.UserState.State = models.StateNewTripTime

	_, err = s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), msgEnterTripTime))

	return err
}

// setTripTime saves the trip start and asks for the trip description.
// It asks again when the time can't be read or has passed.
func (s *Service) setTripTime(ctx context.Context, sess *models.Session, text string) error {
	start, err := parseTripTime(text, sess.UserState.Trip.Date, s.now(), s.location)
	if err != nil {
		reply := msgInvalidTime
		if errors.Is(err, errTimeInPast) {
			reply = msgTimeInPast
		}

		_, err = s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), reply))

		return err
	}

	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Date: &start,
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	sess.UserState.Trip = trip

	sess.UserState.State = models.StateNewTripDescription

	_, err = s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), msgEnterTripDesc))

	return err
}

// setTripDescription saves the trip description and asks to confirm the trip.
// It asks again when the description is empty or too long.
func (s *Service) setTripDescription(ctx context.Context, sess *models.Session, text string) error {
	description, ok := tripText(text, maxTripDescriptionLen)
	if !ok {
		_, err := s.replyInTrip(ctx, sess, tu.Message(tu.ID(sess.ChatID), msgInvalidTripDesc))

		return err
	}

	trip, err := ops.UpdateTrip(ctx, s.backends, sess.UserState.Trip.ID, ops.UpdateTripParams{
		Description: &description,
	})
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	sess.UserState.Trip = trip

	sess.UserState.State = models.StateNewTripConfirm

	if sess.User.ID != trip.CreatedBy.ID {
		return fmt.Errorf("user %d is not the creator of the trip %d", sess.User.ID, trip.ID)
	}

	tripfmt, err := s.renderTrip(trip)
	if err != nil {
		return fmt.Errorf("failed to render trip: %w", err)
	}

	msg := tu.Message(tu.ID(sess.ChatID), fmt.Sprintf("%s\n\nPlease confirm", tripfmt))

	msg.WithReplyMarkup(confirmKeyboard())

	_, err = s.replyInTrip(ctx, sess, msg)

	return err
}

const (
	answerYes = "yes"
	answerNo  = "no"

	// msgConfirmAgain asks again when the answer to the confirmation is neither yes nor no.
	msgConfirmAgain = `Please answer "yes" to publish the trip or "no" to cancel it.`
)

// confirmKeyboard offers the answers to the trip confirmation.
func confirmKeyboard() *tgbotapi.ReplyKeyboardMarkup {
	return tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton(answerYes),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(answerNo),
		),
	).WithResizeKeyboard().WithInputFieldPlaceholder("Confirm").WithOneTimeKeyboard()
}

// confirmTrip publishes the trip on "yes", cancels it on "no", and asks again on any other answer.
// Answers are matched ignoring case and surrounding spaces.
func (s *Service) confirmTrip(ctx context.Context, sess *models.Session, confirm string) error {
	switch strings.ToLower(strings.TrimSpace(confirm)) {
	case answerYes:
		return s.publishTrip(ctx, sess)
	case answerNo:
		return s.cancelTrip(ctx, sess)
	default:
		msg := tu.Message(tu.ID(sess.ChatID), msgConfirmAgain).WithReplyMarkup(confirmKeyboard())

		_, err := s.replyInTrip(ctx, sess, msg)

		return err
	}
}

// cancelTrip deletes the draft trip and ends the dialog; the user starts a new one with /newtrip.
func (s *Service) cancelTrip(ctx context.Context, sess *models.Session) error {
	if err := ops.DeleteTrip(ctx, s.backends, sess.UserState.Trip.ID); err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}

	sess.UserState.State = models.StateStart
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

	err = s.bot.Client().PinChatMessage(ctx, &tgbotapi.PinChatMessageParams{
		ChatID:              tu.ID(resp.Chat.ID),
		MessageID:           resp.MessageID,
		DisableNotification: false,
	})
	if err != nil {
		// The trip is published already; tell the user why the announcement is not pinned.
		log.WithError(ctx, err).Warn("Failed to pin the trip announcement")

		s.sendMessage(ctx, msgPinFailed)
	}

	return nil
}

// msgPinFailed tells the user that the announcement is published but not pinned.
const msgPinFailed = "The trip is published, but I couldn't pin it. " +
	"To pin announcements, make me an admin who can pin messages."

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
			models.StateNewTripConfirm:
			return s.tripAnswerHandler()(c, update)
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
			s.replyError(ctx, sess.ChatID)

			return nil
		}

		list = upcomingTrips(list, s.now())
		sortByStart(list)

		if len(list) == 0 {
			s.sendMessage(ctx, msgNoUpcomingTrips)

			return nil
		}

		cards := make([]string, 0, len(list))

		for _, trip := range list {
			tripfmt, err := s.renderTrip(trip)
			if err != nil {
				log.WithError(ctx, err).WithField("trip_id", trip.ID).Error("Failed to render trip")
				s.replyError(ctx, sess.ChatID)

				return nil
			}

			cards = append(cards, tripfmt)
		}

		// A long list is sent in several messages, so it fits the Telegram limit.
		for _, msg := range splitMessages(myTripsHeader, cards, maxMessageLen) {
			s.sendMessage(ctx, msg)
		}

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
