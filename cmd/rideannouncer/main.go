package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"
)

const (
	// BotName is a Command of the bot.
	botName        = "Ride Announcer Bot"
	botDescription = "Bot for scheduling and announcing planned bicycle trips in chat groups."
	envTGAPIToken  = "RIDE_ANNOUNCER_TELEGRAM_TOKEN"

	cmdHelp     = "help"
	cmdStart    = "start"
	cmdNewTrips = "newtrip"
	cmdTrips    = "trips"
)

var commands = []tgbotapi.BotCommand{
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
		Command:     cmdNewTrips,
		Description: "create new trip",
	},
}

var (
	chatIDs map[int64]struct{}
	mu      sync.RWMutex
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

	wg := &sync.WaitGroup{}
	wg.Add(1)

	stopHandler := handleUpdate(ctx, bot)

	go gracefulShutdown(ctx, wg, bot, stopHandler)

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
		msg := tu.Message(tu.ID(id), "I'm going to sleep. Bye!")

		if _, err := bot.SendMessage(msg); err != nil {
			log.WithError(ctx, err).Error("failed to send message")
		}
	}

	mu.RUnlock()

	log.Info(ctx, "Stop receiving updates")
	bot.StopLongPolling()

	log.Info(ctx, "Stopping bot handlers")

	for _, fn := range stopFns {
		fn(ctx)
	}

	wg.Done()
}

type stopFunc func(ctx context.Context)

func handleUpdate(ctx context.Context, bot *tgbotapi.Bot) stopFunc {
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
	handler.Use(getChatIDMiddleware(ctx))

	handler.Handle(helpHandler(ctx), th.CommandEqual(cmdHelp))
	handler.Handle(startHandler(ctx), th.CommandEqual(cmdStart))
	handler.Handle(newTripHandler(ctx), th.CommandEqual(cmdNewTrips))
	handler.Handle(tripsHandler(ctx), th.CommandEqual(cmdTrips))

	handler.Handle(notFoundHandler(ctx), th.AnyCommand())
	handler.Handle(notCommandHandler(ctx), th.AnyMessage())

	go handler.Start()

	return func(ctx context.Context) {
		log.Info(ctx, "Stopping bot handler")

		handler.Stop()

		log.Info(ctx, "Bot handler stopped")
	}
}

func helpHandler(ctx context.Context) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", cmdHelp)

	msgFormat := `Welcome to %s Help!

Here's a list of commands that you can use: 
%s

Remember, you can always type /%s to see this list of commands again. 

Enjoy planning and going on your bike trips with %s!
`
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := tu.ID(update.Message.Chat.ID)

		l.WithField("chat_id", chatID).Debug("Called help handler")

		cmds, err := bot.GetMyCommands(&tgbotapi.GetMyCommandsParams{})
		if err != nil {
			l.WithError(err).Error("Failed to get bot commands")
		}

		var cmdsStr string

		for _, cmd := range cmds {
			cmdsStr += fmt.Sprintf("\t/%s - %s\n", cmd.Command, cmd.Description)
		}

		msg := tu.Message(chatID,
			fmt.Sprintf(msgFormat, botName, cmdsStr, cmdHelp, botName),
		)

		if _, err = bot.SendMessage(msg); err != nil {
			l.WithError(err).Error("Failed to send message")
		}
	}
}

func startHandler(ctx context.Context) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", cmdStart)

	msgFormat := `Hello, %s! 👋 Welcome to %s. 

I'm here to assist you in scheduling, announcing, and joining exciting bike trips with your community. 

Here are some of the things I can do:

🔹 Create new bike trip announcements.
🔹 Display a list of upcoming trips.
🔹 Subscribe you to a bike trip.
🔹 Provide outfit and SPF recommendations based on the weather.
🔹 Show a list of bike trips you've subscribed to.


To get started and see more detailed instructions, use /%s command.

Happy cycling and let's embark on this journey together, %s! 🚴‍♂️
`

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		sender := update.Message.From
		chatID := tu.ID(update.Message.Chat.ID)

		l.WithField("chat_id", chatID).Debug("Called start handler")

		msg := tu.Message(chatID,
			fmt.Sprintf(msgFormat, sender.FirstName, botName, cmdHelp, sender.FirstName))

		if _, err := bot.SendMessage(msg); err != nil {
			l.WithError(err).Error("Failed to send message")
		}
	}
}

func notFoundHandler(ctx context.Context) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", "not_found")

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := tu.ID(update.Message.Chat.ID)

		l.WithField("chat_id", chatID).Debug("Called not_found handler")

		msg := tu.Message(chatID,
			fmt.Sprintf("Unknown command. Use /%s command to see all available commands.", cmdHelp))

		if _, err := bot.SendMessage(msg); err != nil {
			l.WithError(err).Error("Failed to send message")
		}
	}
}

func notCommandHandler(ctx context.Context) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", "not_command")

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := tu.ID(update.Message.Chat.ID)

		l.WithField("chat_id", chatID).Debug("Called not_command handler")

		msg := tu.Message(chatID,
			fmt.Sprintf("Please use commands. Use /%s command to see all available commands.", cmdHelp))

		if _, err := bot.SendMessage(msg); err != nil {
			l.WithError(err).Error("Failed to send message")
		}
	}
}

func getChatIDMiddleware(ctx context.Context) th.Middleware {
	mu.Lock()
	if chatIDs == nil {
		chatIDs = make(map[int64]struct{})
	}
	mu.Unlock()

	return func(next th.Handler) th.Handler {
		return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
			chatID := update.Message.Chat.ID

			mu.RLock()
			_, exist := chatIDs[chatID]
			mu.RUnlock()

			if !exist {
				mu.Lock()
				chatIDs[chatID] = struct{}{}
				mu.Unlock()

				log.WithField(ctx, "chat_id", chatID).Info("New chat added")
			}

			next(bot, update)
		}
	}
}

func newTripHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdNewTrips)
}

func tripsHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdTrips)
}

func notImplementedHandler(ctx context.Context, cmd string) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", cmd)

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := update.Message.Chat.ID

		l.WithField("chat_id", chatID).Debug("Called not_implemented handler")

		msg := tu.Message(tu.ID(chatID),
			fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", cmdHelp))

		if _, err := bot.SendMessage(msg); err != nil {
			l.WithError(err).Error("Failed to send message")
		}
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

	for _, cmd := range commands {
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
		Commands: commands,
	})
	if err != nil {
		log.WithError(ctx, err).Error("failed to set bot commands")
	}

	log.Info(ctx, "Bot commands set")
}
