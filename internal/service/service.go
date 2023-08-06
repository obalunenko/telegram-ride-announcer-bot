// Package service provides Telegram bot service.
package service

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

const (
	// BotName is a Command of the bot.
	botName        = "Ride Announcer Bot"
	botDescription = "Bot for scheduling and announcing planned bicycle trips in chat groups."

	cmdHelp        = "help"
	cmdStart       = "start"
	cmdNewTrip     = "newtrip"
	cmdTrips       = "trips"
	cmdSubscribe   = "subscribe"
	cmdUnsubscribe = "unsubscribe"
	cmdMyTrips     = "mytrips"
	cmdSubscribed  = "subscribed"
)

var (
	// Disabled commands,
	// because they are not implemented yet.
	disabledCmds = []string{
		cmdTrips, cmdSubscribe, cmdUnsubscribe, cmdMyTrips, cmdSubscribed,
	}

	commands = []tgbotapi.BotCommand{
		{
			Command:     cmdHelp,
			Description: "show this help message",
		},
		{
			Command:     cmdStart,
			Description: "start using the bot",
		},
		{
			Command:     cmdTrips,
			Description: "show all trips",
		},
		{
			Command:     cmdNewTrip,
			Description: "create new trip",
		},
		{
			Command:     cmdSubscribe,
			Description: "subscribe to a trip",
		},
		{
			Command:     cmdUnsubscribe,
			Description: "unsubscribe from a trip",
		},
		{
			Command:     cmdMyTrips,
			Description: "show trips you've created",
		},
		{
			Command:     cmdSubscribed,
			Description: "show trips you've subscribed to",
		},
	}
)

// Service is a Telegram bot service.
type Service struct {
	bot      *tgbotapi.Bot
	sessions sessions.Repository
	users    users.Repository
	states   states.Repository
	trips    trips.Repository

	stopFns []stopFunc
}

// NewParams is a set of parameters for creating a new Service.
type NewParams struct {
	SessionsRepo sessions.Repository
	UsersRepo    users.Repository
	StatesRepo   states.Repository
	TripsRepo    trips.Repository
}

// New creates a new Service.
func New(bot *tgbotapi.Bot, p NewParams) (*Service, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}

	if p.StatesRepo == nil {
		return nil, errors.New("states repository is nil")
	}

	if p.TripsRepo == nil {
		return nil, errors.New("trips repository is nil")
	}

	if p.UsersRepo == nil {
		return nil, errors.New("users repository is nil")
	}

	if p.SessionsRepo == nil {
		return nil, errors.New("sessions repository is nil")
	}

	return &Service{
		bot:      bot,
		sessions: p.SessionsRepo,
		users:    p.UsersRepo,
		states:   p.StatesRepo,
		trips:    p.TripsRepo,
		stopFns:  nil,
	}, nil
}

// Start is a helper function that will be called when the program starts.
func (s *Service) Start(ctx context.Context) {
	s.updateOnStart(ctx)

	self, err := s.bot.GetMe()
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to get bot info")
	}

	log.WithField(ctx, "Username", self.Username).Info("Authorized on account")

	s.stopFns = append(s.stopFns, s.initHandlers(ctx))
}

// Shutdown is a helper function that will be called when the program receives an interrupt signal.
// It will gracefully shut down the bot by waiting for all requests to be processed before shutting down.
func (s *Service) Shutdown(ctx context.Context) {
	list, err := ops.ListSessions(ctx, s.sessions, s.states, s.trips, s.users)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to get sessions")

		return
	}

	for _, sess := range list {
		msg := fmt.Sprintf("I'm going to sleep. Bye, %s!", sess.User.Username)

		s.sendMessage(contextWithSession(ctx, sess), msg)

		if err = ops.DeleteSession(ctx, s.sessions, sess); err != nil {
			log.WithError(ctx, err).WithField("user_id", sess.User.ID).Warn("Failed to delete session")

			continue
		}
	}

	log.Info(ctx, "Stop receiving updates")
	s.bot.StopLongPolling()

	for _, fn := range s.stopFns {
		fn(ctx)
	}
}

type stopFunc func(ctx context.Context)

func (s *Service) initHandlers(ctx context.Context) stopFunc {
	pollChan, err := s.bot.UpdatesViaLongPolling(&tgbotapi.GetUpdatesParams{},
		tgbotapi.WithLongPollingContext(ctx))
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to get updates via long polling")
	}

	handler, err := th.NewBotHandler(s.bot, pollChan)
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to create bot handler")
	}

	handler.Use(s.panicRecovery())
	handler.Use(s.setContextMiddleware(ctx))
	handler.Use(s.setSessionMiddleware())
	handler.Use(s.loggerMiddleware())

	handler.Handle(s.helpHandler(), th.CommandEqual(cmdHelp))
	handler.Handle(s.startHandler(ctx), th.CommandEqual(cmdStart))
	handler.Handle(s.newTripHandler(), th.CommandEqual(cmdNewTrip))
	handler.Handle(s.tripsHandler(), th.CommandEqual(cmdTrips))
	handler.Handle(s.subscribeHandler(), th.CommandEqual(cmdSubscribe))
	handler.Handle(s.unsubscribeHandler(), th.CommandEqual(cmdUnsubscribe))
	handler.Handle(s.myTripsHandler(), th.CommandEqual(cmdMyTrips))
	handler.Handle(s.subscribedHandler(), th.CommandEqual(cmdSubscribed))
	handler.Handle(s.notFoundHandler(ctx), th.AnyCommand())
	handler.Handle(s.textHandler(), th.AnyMessageWithText())

	go handler.Start()

	return func(ctx context.Context) {
		log.Info(ctx, "Stopping bot handler")

		handler.Stop()

		log.Info(ctx, "Bot handler stopped")
	}
}

func (s *Service) updateOnStart(ctx context.Context) {
	maybeUpdateBotName(ctx, s.bot)
	maybeUpdateDescriptionBot(ctx, s.bot)
	maybeUpdateCommands(ctx, s.bot)
}

func maybeUpdateBotName(ctx context.Context, bot *tgbotapi.Bot) {
	self, err := bot.GetMe()
	if err != nil {
		log.WithError(ctx, err).Error("Failed to get bot info")
	}

	isUpToDate := self.Username != botName

	if isUpToDate {
		log.Info(ctx, "Bot name is up to date")

		return
	}

	log.Info(ctx, "Updating bot name")

	err = bot.SetMyName(&tgbotapi.SetMyNameParams{
		Name: botName,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot name")

		return
	}

	log.Info(ctx, "Bot name is up to date")
}

func maybeUpdateDescriptionBot(ctx context.Context, bot *tgbotapi.Bot) {
	desc, err := bot.GetMyDescription(&tgbotapi.GetMyDescriptionParams{})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to get bot info")
	}

	isUpToDate := desc.Description != botDescription

	if isUpToDate {
		log.Info(ctx, "Bot description is up to date")

		return
	}

	log.Info(ctx, "Updating bot description")

	err = bot.SetMyDescription(&tgbotapi.SetMyDescriptionParams{
		Description: botDescription,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot description")

		return
	}

	log.Info(ctx, "Bot description is up to date")
}

func filterCommands(cmds []tgbotapi.BotCommand) []tgbotapi.BotCommand {
	filtered := make([]tgbotapi.BotCommand, 0, len(cmds))

	disabled := make(map[string]struct{}, len(disabledCmds))

	for _, cmd := range disabledCmds {
		disabled[cmd] = struct{}{}
	}

	for _, cmd := range cmds {
		if _, ok := disabled[cmd.Command]; ok {
			continue
		}

		filtered = append(filtered, cmd)
	}

	return filtered
}

func maybeUpdateCommands(ctx context.Context, bot *tgbotapi.Bot) {
	cmds, err := bot.GetMyCommands(&tgbotapi.GetMyCommandsParams{})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to get bot commands")
	}

	registeredCommands := make(map[string]string, len(cmds))

	for _, cmd := range cmds {
		registeredCommands[cmd.Command] = cmd.Description
	}

	var equal bool

	enabled := filterCommands(commands)

	for _, cmd := range enabled {
		desc, ok := registeredCommands[cmd.Command]
		if !ok || desc != cmd.Description {
			equal = false

			break
		}
	}

	if equal {
		log.Info(ctx, "Bot commands are up to date")

		return
	}

	log.Info(ctx, "Updating bot commands")

	err = bot.SetMyCommands(&tgbotapi.SetMyCommandsParams{
		Commands: enabled,
	})
	if err != nil {
		log.WithError(ctx, err).Error("failed to set bot commands")
	}

	log.Info(ctx, "Bot commands set")
}
