package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"
)

const (
	// BotName is a name of the bot.
	botName       = "ride-announcer"
	envTGAPIToken = "RIDE_ANNOUNCER_TELEGRAM_TOKEN"

	cmdHelp     = "help"
	cmdStart    = "start"
	cmdNewTrips = "newtrip"
	cmdTrips    = "trips"
)

var chatIDs map[int64]struct{}

func main() {
	ctx := context.Background()

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

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to create telegram bot")
	}

	bot.Debug = true

	log.WithField(ctx, "username", bot.Self.UserName).Info("Authorized on account")

	commands, err := bot.GetMyCommands()
	if err != nil {
		log.WithError(ctx, err).Fatal("failed to get bot commands")
	}

	for _, command := range commands {
		log.WithField(ctx, "command", command.Command).Info(command.Description)
	}

	// Create a new UpdateConfig struct with an offset of 0. Offsets are used
	// to make sure Telegram knows we've handled previous values and we don't
	// need them repeated.
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Info(ctx, "Start receiving updates")

		// Start polling Telegram for updates.
		updates := bot.GetUpdatesChan(updateConfig)

		// Stop polling for updates when the program is exiting.
		defer func() {
			log.Info(ctx, "Stop receiving updates")
			bot.StopReceivingUpdates()
		}()

		for {
			select {
			case <-ctx.Done():
				log.Info(ctx, "Received stop signal")

				for id := range chatIDs {
					msg := tgbotapi.NewMessage(id, "I'm going to sleep. Bye!")

					if _, err = bot.Send(msg); err != nil {
						log.WithError(ctx, err).Error("failed to send message")
					}
				}

				return
			case update := <-updates:
				log.WithField(ctx, "update_id", update.UpdateID).Debug("Received update")

				// Telegram can send many types of updates depending on what your Bot
				// is up to. We only want to look at messages for now, so we can
				// discard any other updates.
				if update.Message == nil {
					continue
				}

				if chatIDs == nil {
					chatIDs = make(map[int64]struct{})
				}

				if _, ok := chatIDs[update.Message.Chat.ID]; !ok {
					chatIDs[update.Message.Chat.ID] = struct{}{}

					log.WithField(ctx, "chat_id", update.Message.Chat.ID).Info("New chat")
				}

				if !update.Message.IsCommand() {
					msg := tgbotapi.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf("I don't understand you. Please use /%s to see what I can do", cmdHelp),
					)

					if _, err = bot.Send(msg); err != nil {
						log.WithError(ctx, err).Error("failed to send message")
					}

					continue
				}

				// Create a new MessageConfig. We don't have text yet,
				// so we leave it empty.
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

				// We'll also say that this message is a reply to the previous message.
				// For any other specifications than Chat ID or Text, you'll need to
				// set fields on the `MessageConfig`.
				msg.ReplyToMessageID = update.Message.MessageID

				// Extract the command from the Message.
				switch update.Message.Command() {
				case cmdHelp:
					msg.Text = fmt.Sprintf(
						"Available commands:\n\t/%s - show this help message\n\t/%s - start using the bot\n\t/%s - show all trips\n\t/%s - create new trip",
						cmdHelp,
						cmdStart,
						cmdTrips,
						cmdNewTrips,
					)
				case cmdStart:
					msg.Text = fmt.Sprintf(
						"Hello, %s!\n"+
							"I'm ride announcer bot.\n"+
							"I can help you to organize your trips. Please use /%s to see what I can do",
						update.SentFrom().FirstName,
						cmdHelp,
					)
				case cmdTrips:
					msg.Text = "Unfortunately this feature is not implemented yet"
				case cmdNewTrips:
					msg.Text = "Unfortunately this feature is not implemented yet"
				default:
					msg.Text = fmt.Sprintf("I don't understand you. Please use /%s to see what I can do", cmdHelp)
				}

				// Okay, we're sending our message off! We don't care about the message
				// we just sent, so we'll discard it.
				if _, err = bot.Send(msg); err != nil {
					// Note that panics are a bad way to handle errors. Telegram can
					// have service outages or network errors, you should retry sending
					// messages or more gracefully handle failures.
					log.WithError(ctx, err).Error("failed to send message")
				}
			}
		}
	}()

	<-ctx.Done()

	wg.Wait()

	log.Info(ctx, "Bot stopped")
}
