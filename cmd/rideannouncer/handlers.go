package main

import (
	"context"
	"fmt"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	log "github.com/obalunenko/logger"
)

func notFoundHandler(ctx context.Context) th.Handler {
	return unsupportedHandler(ctx, "Command not found.")
}

func notCommandHandler(ctx context.Context) th.Handler {
	return unsupportedHandler(ctx, "This is not a command.")
}

func unsupportedHandler(ctx context.Context, text string) th.Handler {
	ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "unsupported"))

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := update.Message.Chat.ID

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", chatID))

		log.Debug(ctx, "Called unsupported handler")

		msg := fmt.Sprintf("%s Use /%s command to see all available commands.", text, cmdHelp)

		sendMessage(ctx, bot, chatID, msg)
	}
}

func startHandler(ctx context.Context) th.Handler {
	ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdStart))

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
		chatID := update.Message.Chat.ID

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", chatID))

		log.Debug(ctx, "Called start handler")

		msg := fmt.Sprintf(msgFormat, sender.FirstName, botName, cmdHelp, sender.FirstName)

		sendMessage(ctx, bot, chatID, msg)
	}
}

func helpHandler(ctx context.Context) th.Handler {
	ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdHelp))

	msgFormat := `Welcome to %s Help!

Here's a list of commands that you can use: 
%s

Remember, you can always type /%s to see this list of commands again. 

Enjoy planning and going on your bike trips with %s!
`

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := update.Message.Chat.ID

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", chatID))

		log.Debug(ctx, "Called help handler")

		cmds, err := bot.GetMyCommands(&tgbotapi.GetMyCommandsParams{})
		if err != nil {
			log.WithError(ctx, err).Error("Failed to get bot commands")
		}

		var cmdsStr string

		for _, cmd := range cmds {
			cmdsStr += fmt.Sprintf("\t/%s - %s\n", cmd.Command, cmd.Description)
		}

		msg := fmt.Sprintf(msgFormat, botName, cmdsStr, cmdHelp, botName)

		sendMessage(ctx, bot, chatID, msg)
	}
}

func newTripHandler(ctx context.Context) th.Handler {
	// 1. Ask for date and time

	// 2. Ask for description

	// 3. Ask for a track link. If not provided, ask for track file

	// 4. Ask for a photo (optional)

	// 5. Subscribe the creator to the trip (maybe implement it later)

	// 6. Announce the trip.

	// 7. Pin the trip announcement to the chat.

	return notImplementedHandler(ctx, cmdNewTrip)
}

func tripsHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdTrips)
}

func subscribeHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdSubscribe)
}

func unsubscribeHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdUnsubscribe)
}

func myTripsHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdMyTrips)
}

func subscribedHandler(ctx context.Context) th.Handler {
	return notImplementedHandler(ctx, cmdSubscribed)
}

func notImplementedHandler(ctx context.Context, cmd string) th.Handler {
	ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmd))

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := update.Message.Chat.ID

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "chat_id", chatID))

		log.Debug(ctx, "Called not_implemented handler")

		msg := fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", cmdHelp)

		sendMessage(ctx, bot, chatID, msg)
	}
}
