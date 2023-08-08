package telegram

import (
	"testing"

	tgbotapi "github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
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
