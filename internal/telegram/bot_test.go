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
	assert.Equal(t, "Test Bot", bot.Name())
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

func TestNewBot_UpdateOnStart(t *testing.T) {
	helpCmd := tgbotapi.BotCommand{Command: "help", Description: "show help"}
	tripsCmd := tgbotapi.BotCommand{Command: "trips", Description: "show trips"}

	// The bot has the name from getMe, and the description and commands set below.
	const (
		botName        = "Test Bot"
		botDescription = "Ride announcer"
	)

	botCommands := Commands{NewCommand(helpCmd.Command, helpCmd.Description, true)}

	tests := []struct {
		name       string
		registered []tgbotapi.BotCommand
		opts       []BotOption
		fail       string
		// wantCall is the method the bot calls to update itself; empty when it is up to date.
		wantCall        string
		wantParams      map[string]any
		wantName        string
		wantDescription string
		wantCommands    Commands
	}{
		{
			name:            "name is up to date",
			opts:            []BotOption{WithName(botName)},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name:            "name updated",
			opts:            []BotOption{WithName("Ride Bot")},
			wantCall:        "setMyName",
			wantParams:      map[string]any{"name": "Ride Bot"},
			wantName:        "Ride Bot",
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name:            "name update fails",
			opts:            []BotOption{WithName("Ride Bot")},
			fail:            "setMyName",
			wantCall:        "setMyName",
			wantParams:      map[string]any{"name": "Ride Bot"},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name:            "description is up to date",
			opts:            []BotOption{WithDescription(botDescription)},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name:            "description updated",
			opts:            []BotOption{WithDescription("Bike trips")},
			wantCall:        "setMyDescription",
			wantParams:      map[string]any{"description": "Bike trips"},
			wantName:        botName,
			wantDescription: "Bike trips",
			wantCommands:    botCommands,
		},
		{
			name:            "description update fails",
			opts:            []BotOption{WithDescription("Bike trips")},
			fail:            "setMyDescription",
			wantCall:        "setMyDescription",
			wantParams:      map[string]any{"description": "Bike trips"},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name: "commands are up to date",
			opts: []BotOption{WithCommands(Commands{
				NewCommand(helpCmd.Command, helpCmd.Description, true),
				NewCommand(tripsCmd.Command, tripsCmd.Description, false),
			})},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name: "command added",
			opts: []BotOption{WithCommands(Commands{
				NewCommand(helpCmd.Command, helpCmd.Description, true),
				NewCommand(tripsCmd.Command, tripsCmd.Description, true),
			})},
			wantCall: "setMyCommands",
			wantParams: map[string]any{"commands": []any{
				map[string]any{"command": "help", "description": "show help"},
				map[string]any{"command": "trips", "description": "show trips"},
			}},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    toCommands([]tgbotapi.BotCommand{helpCmd, tripsCmd}),
		},
		{
			name:       "command removed",
			registered: []tgbotapi.BotCommand{helpCmd, tripsCmd},
			opts: []BotOption{WithCommands(Commands{
				NewCommand(helpCmd.Command, helpCmd.Description, true),
				NewCommand(tripsCmd.Command, tripsCmd.Description, false),
			})},
			wantCall: "setMyCommands",
			wantParams: map[string]any{"commands": []any{
				map[string]any{"command": "help", "description": "show help"},
			}},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
		{
			name: "command description changed",
			opts: []BotOption{WithCommands(Commands{
				NewCommand(helpCmd.Command, "list commands", true),
			})},
			wantCall: "setMyCommands",
			wantParams: map[string]any{"commands": []any{
				map[string]any{"command": "help", "description": "list commands"},
			}},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    Commands{NewCommand(helpCmd.Command, "list commands", true)},
		},
		{
			name: "commands update fails",
			opts: []BotOption{WithCommands(Commands{
				NewCommand(helpCmd.Command, "list commands", true),
			})},
			fail:     "setMyCommands",
			wantCall: "setMyCommands",
			wantParams: map[string]any{"commands": []any{
				map[string]any{"command": "help", "description": "list commands"},
			}},
			wantName:        botName,
			wantDescription: botDescription,
			wantCommands:    botCommands,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registered := tt.registered
			if registered == nil {
				registered = []tgbotapi.BotCommand{helpCmd}
			}

			api := telegramtest.NewServer(t)
			api.SetResult("getMyDescription", tgbotapi.BotDescription{Description: botDescription})
			api.SetResult("getMyCommands", registered)

			if tt.fail != "" {
				api.Fail(tt.fail)
			}

			opts := append([]BotOption{WithClientOptions(api.ClientOptions()...)}, tt.opts...)

			// Failing to update the bot does not fail the bot creation.
			bot, err := NewBot(context.Background(), telegramtest.Token, opts...)
			require.NoError(t, err)

			for range 3 { // getMe, getMyDescription, getMyCommands.
				api.NextRequest(t)
			}

			if tt.wantCall != "" {
				req := api.NextRequest(t)
				assert.Equal(t, tt.wantCall, req.Method)
				assert.Equal(t, tt.wantParams, req.Params)
			}

			api.AssertNoRequest(t)

			assert.Equal(t, tt.wantName, bot.Name())
			assert.Equal(t, tt.wantDescription, bot.Description())
			assert.Equal(t, tt.wantCommands, bot.Commands())
		})
	}
}
