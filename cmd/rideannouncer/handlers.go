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

func unsupportedHandler(ctx context.Context, text string) th.Handler {
	msg := fmt.Sprintf("%s Use /%s command to see all available commands.", text, cmdHelp)

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", "unsupported"))

		log.Debug(ctx, "Called unsupported handler")

		sendMessage(ctx, bot, msg)
	}
}

func startHandler(ctx context.Context) th.Handler {
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
		ctx = update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdStart))

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		log.Debug(ctx, "Called start handler")

		msg := fmt.Sprintf(msgFormat, sess.user.firstname, botName, cmdHelp, sess.user.firstname)

		sendMessage(ctx, bot, msg)
	}
}

func helpHandler() th.Handler {
	msgFormat := `Welcome to %s Help!

Here's a list of commands that you can use: 
%s

Remember, you can always type /%s to see this list of commands again. 

Enjoy planning and going on your bike trips with %s!
`

	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdHelp))

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

		sendMessage(ctx, bot, msg)
	}
}

func newTripHandler() th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		ctx = log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmdNewTrip))

		sess := sessionFromContext(ctx)
		if sess == nil {
			log.Error(ctx, "Session is nil")

			return
		}

		// 1. Ask for date and time
		keyboard := tu.Keyboard(
			tu.KeyboardRow(
				tu.KeyboardButton("today"),
			),
			tu.KeyboardRow(
				tu.KeyboardButton("tomorrow"),
			),
		).WithResizeKeyboard().WithInputFieldPlaceholder("Enter date")

		msg := tu.Message(tu.ID(sess.chatID), "Please select date and time")

		msg.WithReplyMarkup(keyboard)

		sent, err := bot.SendMessage(msg)
		if err != nil {
			log.WithError(ctx, err).Error("Failed to send message")
		}

		if reply := sent.ReplyToMessage; reply != nil {
			log.WithField(ctx, "reply_to_message", reply.Text).Debug("Reply to message")
		}

		// 2. Ask for description

		// 3. Ask for a track link. If not provided, ask for track file

		// 4. Ask for a photo (optional)

		// 5. Subscribe the creator to the trip (maybe implement it later)

		// 6. Announce the trip.

		// 7. Pin the trip announcement to the chat.
	}
}

func tripsHandler() th.Handler {
	return notImplementedHandler(cmdTrips)
}

func subscribeHandler() th.Handler {
	return notImplementedHandler(cmdSubscribe)
}

func unsubscribeHandler() th.Handler {
	return notImplementedHandler(cmdUnsubscribe)
}

func myTripsHandler() th.Handler {
	return notImplementedHandler(cmdMyTrips)
}

func subscribedHandler() th.Handler {
	return notImplementedHandler(cmdSubscribed)
}

func notImplementedHandler(cmd string) th.Handler {
	return func(bot *tgbotapi.Bot, update tgbotapi.Update) {
		ctx := update.Context()

		log.ContextWithLogger(ctx, log.WithField(ctx, "command_handler", cmd))

		log.Debug(ctx, "Called not_implemented handler")

		msg := fmt.Sprintf("Not implemented yet. Use /%s command to see all available commands.", cmdHelp)

		sendMessage(ctx, bot, msg)
	}
}
