package main

import (
	"context"
	"fmt"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/obalunenko/logger"
)

func notFoundHandler(ctx context.Context) th.Handler {
	return unsupportedHandler(ctx, "Command not found.")
}

func notCommandHandler(ctx context.Context) th.Handler {
	return unsupportedHandler(ctx, "This is not a command.")
}

func unsupportedHandler(ctx context.Context, text string) th.Handler {
	l := log.FromContext(ctx).WithField("command_handler", "unsupported")

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		chatID := tu.ID(update.Message.Chat.ID)

		l.WithField("chat_id", chatID).Debug("Called unsupported handler")

		msg := tu.Message(chatID,
			fmt.Sprintf("%s Use /%s command to see all available commands.", text, cmdHelp))

		if _, err := bot.SendMessage(msg); err != nil {
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

func newTripHandler(ctx context.Context) th.Handler {
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
