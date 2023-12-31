// ridesannouncer is a bot for scheduling and announcing planned bicycle trips in chat groups.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/backends"
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
	telegram.NewCommand(service.CmdMyTrips, "show trips you've created", true),
	telegram.NewCommand(service.CmdSubscribed, "show trips you've subscribed to", false),
}

func main() {
	signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP}

	notifyChan := make(chan os.Signal, 1)

	signal.Notify(notifyChan, signals...)

	ctx, stop := context.WithCancelCause(context.Background())
	defer func() {
		stop(errors.New("main: exit"))
	}()

	go func() {
		s := <-notifyChan

		stop(fmt.Errorf("received signal %q", s.String()))
	}()

	printVersion(ctx)

	log.Init(ctx, log.Params{
		Writer: os.Stdout,
		Level:  "DEBUG",
		Format: "text",
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

	params := backends.NewParams{
		Sessions: sessionsRepo,
		Users:    usersRepo,
		States:   statesRepo,
		Trips:    tripsRepo,
	}

	b, err := backends.New(params)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create backends for service")
	}

	svc, err := service.New(bot, b)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create service")
	}

	svc.Start(ctx)

	<-ctx.Done()

	log.WithField(ctx, "reason", context.Cause(ctx)).Info("Exiting...")

	svc.Shutdown(ctx)

	log.Info(ctx, "Bot stopped")
}
