// ridesannouncer is a bot for scheduling and announcing planned bicycle trips in chat groups.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram"
)

const (
	envTGAPIToken = "RIDE_ANNOUNCER_TELEGRAM_TOKEN"
)

const (
	// BotName is a Command of the bot.
	botName        = "Ride Announcer Bot"
	botDescription = "Bot for scheduling and announcing planned bicycle trips in chat groups."
)

var commands = telegram.Commands{
	telegram.NewCommand(service.CmdStart, "start using the bot", true),
	telegram.NewCommand(service.CmdHelp, "show help", true),
	telegram.NewCommand(service.CmdNewTrip, "create new trip", true),
	telegram.NewCommand(service.CmdTrips, "show all trips", false),
	telegram.NewCommand(service.CmdSubscribe, "subscribe to a trip", false),
	telegram.NewCommand(service.CmdUnsubscribe, "unsubscribe from a trip", false),
	telegram.NewCommand(service.CmdMyTrips, "show trips you've created", false),
	telegram.NewCommand(service.CmdSubscribed, "show trips you've subscribed to", false),
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	log.Init(ctx, log.Params{
		Writer:       nil,
		Level:        "DEBUG",
		Format:       "text",
		SentryParams: log.SentryParams{},
	})

	ctx = log.ContextWithLogger(ctx, log.FromContext(ctx))

	log.Info(ctx, "Starting bot")

	token, err := getenv.Env[string](envTGAPIToken)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to get telegram api token")
	}

	opts := []telegram.BotOption{
		telegram.WithCommands(commands),
		telegram.WithDescription(botDescription),
		telegram.WithUsername(botName),
	}

	bot, err := telegram.NewBot(ctx, token, opts...)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create telegram bot")
	}

	sessionsRepo := sessions.NewInMemory()
	usersRepo := users.NewInMemory()
	statesRepo := states.NewInMemory()
	tripsRepo := trips.NewInMemory()

	params := service.NewParams{
		SessionsRepo: sessionsRepo,
		UsersRepo:    usersRepo,
		StatesRepo:   statesRepo,
		TripsRepo:    tripsRepo,
	}

	svc, err := service.New(bot, params)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create service")
	}

	svc.Start(ctx)

	<-ctx.Done()

	log.Info(ctx, "Received stop signal")

	svc.Shutdown(ctx)

	log.Info(ctx, "Bot stopped")
}
