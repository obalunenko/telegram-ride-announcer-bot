// ridesannouncer is a bot for scheduling and announcing planned bicycle trips in chat groups.
package main

import (
	"context"
	"os"
	"os/signal"

	tgbotapi "github.com/mymmrac/telego"
	"github.com/obalunenko/getenv"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service"
)

const (
	envTGAPIToken = "RIDE_ANNOUNCER_TELEGRAM_TOKEN"
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

	sessionsRepo := sessions.NewInMemory()
	usersRepo := users.NewInMemory()

	svc := service.New(bot, sessionsRepo, usersRepo)

	svc.Start(ctx)

	<-ctx.Done()

	log.Info(ctx, "Received stop signal")

	svc.Shutdown(ctx)

	log.Info(ctx, "Bot stopped")
}
