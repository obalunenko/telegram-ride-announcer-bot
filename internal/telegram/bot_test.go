package telegram

import (
	"context"
	"testing"

	tgbotapi "github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram/telegramtest"
)

func TestCommand(t *testing.T) {
	type args struct {
		command     string
		description string
		enabled     bool
	}

	tests := []struct {
		name string
		args args
		want Command
	}{
		{
			name: "command enabled",
			args: args{
				command:     "command",
				description: "description",
				enabled:     true,
			},
			want: Command{
				BotCommand: tgbotapi.BotCommand{
					Command:     "command",
					Description: "description",
				},
				enabled: true,
			},
		},
		{
			name: "command disabled",
			args: args{
				command:     "command",
				description: "description",
				enabled:     false,
			},
			want: Command{
				BotCommand: tgbotapi.BotCommand{
					Command:     "command",
					Description: "description",
				},
				enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCommand(tt.args.command, tt.args.description, tt.args.enabled)
			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.args.enabled, got.Enabled())
		})
	}
}

func TestCommands_EnabledCommands(t *testing.T) {
	tests := []struct {
		name string
		c    Commands
		want []tgbotapi.BotCommand
	}{
		{
			name: "all commands enabled",
			c: Commands{
				NewCommand("command1", "description1", true),
				NewCommand("command2", "description2", true),
			},
			want: []tgbotapi.BotCommand{
				{
					Command:     "command1",
					Description: "description1",
				},
				{
					Command:     "command2",
					Description: "description2",
				},
			},
		},
		{
			name: "some commands disabled",
			c: Commands{
				NewCommand("command1", "description1", true),
				NewCommand("command2", "description2", false),
			},
			want: []tgbotapi.BotCommand{
				{
					Command:     "command1",
					Description: "description1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, tt.c.EnabledCommands())
		})
	}
}

func TestNewBot(t *testing.T) {
	api := telegramtest.NewServer(t)
	api.SetResult("getMyDescription", tgbotapi.BotDescription{Description: "Ride announcer"})
	api.SetResult("getMyCommands", []tgbotapi.BotCommand{{Command: "help", Description: "show help"}})

	bot, err := NewBot(context.Background(), telegramtest.Token, WithClientOptions(api.ClientOptions()...))
	require.NoError(t, err)

	assert.Equal(t, telegramtest.BotID, bot.ID())
	assert.Equal(t, telegramtest.BotUsername, bot.Username())
	assert.Equal(t, "Ride announcer", bot.Description())
	assert.Equal(t, Commands{NewCommand("help", "show help", true)}, bot.Commands())
	assert.NotNil(t, bot.Client())

	for _, method := range []string{"getMe", "getMyDescription", "getMyCommands"} {
		assert.Equal(t, method, api.NextRequest(t).Method)
	}
}

func TestNewBot_Error(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		fail    string
		wantErr string
	}{
		{
			name:    "invalid token",
			token:   "invalid",
			wantErr: "failed to create bot",
		},
		{
			name:    "get bot info",
			token:   telegramtest.Token,
			fail:    "getMe",
			wantErr: "failed to get bot info",
		},
		{
			name:    "get bot description",
			token:   telegramtest.Token,
			fail:    "getMyDescription",
			wantErr: "failed to get bot description",
		},
		{
			name:    "get bot commands",
			token:   telegramtest.Token,
			fail:    "getMyCommands",
			wantErr: "failed to get bot commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := telegramtest.NewServer(t)
			if tt.fail != "" {
				api.Fail(tt.fail)
			}

			bot, err := NewBot(context.Background(), tt.token, WithClientOptions(api.ClientOptions()...))
			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, bot)
		})
	}
}

func TestNewBot_WithCommands(t *testing.T) {
	tests := []struct {
		name string
		fail bool
	}{
		{
			name: "commands updated",
			fail: false,
		},
		{
			name: "update fails",
			fail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := telegramtest.NewServer(t)
			api.SetResult("getMyCommands", []tgbotapi.BotCommand{{Command: "help", Description: "old help"}})

			if tt.fail {
				api.Fail("setMyCommands")
			}

			_, err := NewBot(context.Background(), telegramtest.Token,
				WithClientOptions(api.ClientOptions()...),
				WithCommands(Commands{
					NewCommand("help", "show help", true),
					NewCommand("trips", "show trips", false),
				}),
			)
			// Failing to update the commands does not fail the bot creation.
			require.NoError(t, err)

			for range 3 { // getMe, getMyDescription, getMyCommands.
				api.NextRequest(t)
			}

			req := api.NextRequest(t)
			assert.Equal(t, "setMyCommands", req.Method)
			assert.Equal(t, []any{map[string]any{"command": "help", "description": "show help"}}, req.Params["commands"])
		})
	}
}
