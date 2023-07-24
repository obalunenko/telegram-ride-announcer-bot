// ridesannouncer is a bot for scheduling and announcing planned bicycle trips in chat groups.
package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"
)

const (
	// BotName is a Command of the bot.
	botName        = "Ride Announcer Bot"
	botDescription = "Bot for scheduling and announcing planned bicycle trips in chat groups."
	envTGAPIToken  = "RIDE_ANNOUNCER_TELEGRAM_TOKEN"

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

	bot, err := tgbotapi.NewBot(token)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create telegram bot")
	}

	updateOnStart(ctx, bot)

	self, err := bot.GetMe()
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to get bot info")
	}

	log.WithField(ctx, "username", self.Username).Info("Authorized on account")

	var wg sync.WaitGroup

	wg.Add(1)

	go gracefulShutdown(ctx, &wg, bot, initHandlers(ctx, bot))

	wg.Wait()

	log.Info(ctx, "Bot stopped")
}

// gracefulShutdown is a helper function that will be called when the program receives an interrupt signal.
// It will gracefully shut down the bot by waiting for all requests to be processed before shutting down.
func gracefulShutdown(ctx context.Context, wg *sync.WaitGroup, bot *tgbotapi.Bot, stopFns ...stopFunc) {
	<-ctx.Done()

	log.Info(ctx, "Received stop signal")

	mu.RLock()

	for id := range chatIDs {
		msg := "I'm going to sleep. Bye!"

		sendMessage(contextWithSession(ctx, &session{
			chatID: id,
		}), bot, msg)
	}

	mu.RUnlock()

	log.Info(ctx, "Stop receiving updates")
	bot.StopLongPolling()

	for _, fn := range stopFns {
		fn(ctx)
	}

	wg.Done()
}

type stopFunc func(ctx context.Context)

func initHandlers(ctx context.Context, bot *tgbotapi.Bot) stopFunc {
	pollChan, err := bot.UpdatesViaLongPolling(&tgbotapi.GetUpdatesParams{},
		tgbotapi.WithLongPollingContext(ctx))
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to get updates via long polling")
	}

	handler, err := th.NewBotHandler(bot, pollChan)
	if err != nil {
		log.WithError(ctx, err).Fatal("Failed to create bot handler")
	}

	handler.Use(th.PanicRecovery)
	handler.Use(setContextMiddleware(ctx))
	handler.Use(setSessionMiddleware())
	handler.Use(loggerMiddleware())
	handler.Use(getChatIDMiddleware())

	handler.Handle(helpHandler(), th.CommandEqual(cmdHelp))
	handler.Handle(startHandler(ctx), th.CommandEqual(cmdStart))
	handler.Handle(newTripHandler(), th.CommandEqual(cmdNewTrip))
	handler.Handle(tripsHandler(), th.CommandEqual(cmdTrips))
	handler.Handle(subscribeHandler(), th.CommandEqual(cmdSubscribe))
	handler.Handle(unsubscribeHandler(), th.CommandEqual(cmdUnsubscribe))
	handler.Handle(myTripsHandler(), th.CommandEqual(cmdMyTrips))
	handler.Handle(subscribedHandler(), th.CommandEqual(cmdSubscribed))

	handler.Handle(notFoundHandler(ctx), th.AnyCommand())

	go handler.Start()

	return func(ctx context.Context) {
		log.Info(ctx, "Stopping bot handler")

		handler.Stop()

		log.Info(ctx, "Bot handler stopped")
	}
}

func updateOnStart(ctx context.Context, bot *tgbotapi.Bot) {
	maybeUpdateBotName(ctx, bot)
	maybeUpdateDescriptionBot(ctx, bot)
	maybeUpdateCommands(ctx, bot)
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
