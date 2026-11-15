![Go Version](https://img.shields.io/github/go-mod/go-version/obalunenko/telegram-ride-announcer-bot)
[![GoDoc](https://pkg.go.dev/badge/github.com/obalunenko/telegram-ride-announcer-bot?status.svg)](https://pkg.go.dev/github.com/obalunenko/telegram-ride-announcer-bot)
[![Go Report Card](https://goreportcard.com/badge/github.com/obalunenko/telegram-ride-announcer-bot)](https://goreportcard.com/report/github.com/obalunenko/telegram-ride-announcer-bot)
![build](https://github.com/obalunenko/telegram-ride-announcer-bot/actions/workflows/go.yml/badge.svg)
![coverbadger-tag-do-not-edit](https://img.shields.io/badge/coverage-3.75%25-brightgreen?longCache=true&style=flat)
[![codecov](https://codecov.io/gh/obalunenko/telegram-ride-announcer-bot/branch/master/graph/badge.svg?token=zYYMoHikI7)](https://codecov.io/gh/obalunenko/telegram-ride-announcer-bot)
[![License](https://img.shields.io/github/license/obalunenko/telegram-ride-announcer-bot)](/LICENSE)

# telegram-ride-announcer-bot
Telegram bot for scheduling and announcing planned bicycle trips in chat groups

## Commands

| Command | Description |
|---------|-------------|
| `/start` | Start using the bot. |
| `/help` | Show the available commands. |
| `/newtrip` | Create a trip: name, date (`today`, `tomorrow`, or `DD.MM.YYYY`), start time (`HH:MM`), and description, then announce and pin it in the chat. |
| `/cancel` | Stop creating a trip and discard it. |
| `/mytrips` | Show your upcoming trips, the nearest first. |

The bot also knows `/trips`, `/subscribe`, `/unsubscribe`, and `/subscribed`; they are not implemented yet and are hidden from the Telegram menu.

## Configuration

The bot reads its settings from environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `RIDE_ANNOUNCER_TELEGRAM_TOKEN` | yes | — | Telegram Bot API token from [@BotFather](https://t.me/BotFather). |
| `RIDE_ANNOUNCER_LOG_LEVEL` | no | `INFO` | Log level: `DEBUG`, `INFO`, `WARN`, `ERROR`, or `FATAL`. An unknown value falls back to `INFO`. |
| `RIDE_ANNOUNCER_LOG_FORMAT` | no | `text` | Log format: `text` or `json`. |
| `RIDE_ANNOUNCER_TIMEZONE` | no | `UTC` | IANA time zone of trip dates and times, for example `Asia/Tbilisi`. The bot exits when the zone is unknown. |

The bot keeps all data in memory, so users, sessions, and trips are lost when it restarts.

## Running locally

```shell
export RIDE_ANNOUNCER_TELEGRAM_TOKEN=<token>
make run-local
```

`make run-local` builds the Docker image and starts it with Docker Compose; the variables above are passed to the container.
