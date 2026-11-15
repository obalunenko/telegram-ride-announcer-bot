// Package service provides Telegram bot service.
package service

import (
	"context"
	"errors"
	"fmt"

	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	templates "github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram"
)

const (
	// CmdHelp is a command for getting help.
	CmdHelp = "help"
	// CmdStart is a command for starting the bot.
	CmdStart = "start"
	// CmdNewTrip is a command for creating a new trip.
	CmdNewTrip = "newtrip"
	// CmdTrips is a command for getting trips.
	CmdTrips = "trips"
	// CmdSubscribe is a command for subscribing to trip.
	CmdSubscribe = "subscribe"
	// CmdUnsubscribe is a command for unsubscribing from a trip.
	CmdUnsubscribe = "unsubscribe"
	// CmdMyTrips is a command for getting user's trips.
	CmdMyTrips = "mytrips"
	// CmdSubscribed is a command for getting user's subscribed trips.
	CmdSubscribed = "subscribed"
)

// Service is a Telegram bot service.
type Service struct {
	bot       *telegram.Bot
	backends  backends
	templates templates.Renderer

	stopFns []stopFunc
}

// New creates a new Service.
func New(bot *telegram.Bot, b backends) (*Service, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}

	tpls, err := templates.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &Service{
		bot:       bot,
		backends:  b,
		templates: tpls,
		stopFns:   nil,
	}, nil
}

// Start is a helper function that will be called when the program starts.
func (s *Service) Start(ctx context.Context) {
	log.WithField(ctx, "Username", s.bot.Username()).Info("Authorized on account")

	s.stopFns = append(s.stopFns, s.initHandlers(ctx))
}

// Shutdown is a helper function that will be called when the program receives an interrupt signal.
// It will gracefully shut down the bot by waiting for all requests to be processed before shutting down.
func (s *Service) Shutdown(ctx context.Context) {
	list, err := ops.ListSessions(ctx, s.backends)
	if err != nil {
		log.WithError(ctx, err).Error("Failed to get sessions")

		return
	}

	for _, sess := range list {
		msg := fmt.Sprintf("I'm going to sleep. Bye, %s!", sess.User.Firstname)

		s.sendMessage(contextWithSession(ctx, sess), msg)

		if err = ops.DeleteSession(ctx, s.backends, sess); err != nil {
			log.WithError(ctx, err).WithField("user_id", sess.User.ID).Warn("Failed to delete session")

			continue
		}
	}

	for _, fn := range s.stopFns {
		fn(ctx)
	}
}

type stopFunc func(ctx context.Context)

func (s *Service) initHandlers(ctx context.Context) stopFunc {
	// Canceling pollCtx stops long polling and closes pollChan.
	// stopPolling can't be deferred here: the returned stop function calls it on Shutdown.
	pollCtx, stopPolling := context.WithCancel(ctx) // NOSONAR

	// nil params use the default long polling timeout.
	pollChan, err := s.bot.Client().UpdatesViaLongPolling(pollCtx, nil)
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to get updates via long polling")
	}

	handler, err := th.NewBotHandler(s.bot.Client(), pollChan)
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to create bot handler")
	}

	handler.Use(s.panicRecovery())
	handler.Use(s.setContextMiddleware(ctx))
	handler.Use(s.setSessionMiddleware())
	handler.Use(s.loggerMiddleware())

	handler.Handle(s.helpHandler(), th.CommandEqual(CmdHelp))
	handler.Handle(s.startHandler(ctx), th.CommandEqual(CmdStart))
	handler.Handle(s.newTripHandler(), th.CommandEqual(CmdNewTrip))
	handler.Handle(s.tripsHandler(), th.CommandEqual(CmdTrips))
	handler.Handle(s.subscribeHandler(), th.CommandEqual(CmdSubscribe))
	handler.Handle(s.unsubscribeHandler(), th.CommandEqual(CmdUnsubscribe))
	handler.Handle(s.myTripsHandler(), th.CommandEqual(CmdMyTrips))
	handler.Handle(s.subscribedHandler(), th.CommandEqual(CmdSubscribed))
	handler.Handle(s.notFoundHandler(ctx), th.AnyCommand())
	handler.Handle(s.textHandler(), th.AnyMessageWithText())

	go func() {
		if err := handler.Start(); err != nil {
			log.WithError(ctx, err).Error("Bot handler stopped with error")
		}
	}()

	return func(ctx context.Context) {
		log.Info(ctx, "Stop receiving updates")

		stopPolling()

		log.Info(ctx, "Stopping bot handler")

		// Waits for the updates that are still being handled.
		if err := handler.StopWithContext(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to stop bot handler")
		}

		log.Info(ctx, "Bot handler stopped")
	}
}
