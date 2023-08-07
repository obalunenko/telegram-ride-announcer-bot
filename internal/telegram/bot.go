// Package telegram provides Telegram bot wrapper.
package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/mymmrac/telego"
	log "github.com/obalunenko/logger"
)

// Command is a bot command.
type Command struct {
	tgbotapi.BotCommand
	enabled bool
}

// Enabled returns true if command is enabled.
func (c Command) Enabled() bool {
	return c.enabled
}

func toCommands(commands []tgbotapi.BotCommand) Commands {
	cmds := make(Commands, 0, len(commands))

	for _, cmd := range commands {
		cmds = append(cmds, NewCommand(cmd.Command, cmd.Description, true))
	}

	return cmds
}

// Commands is a list of bot commands.
type Commands []Command

// EnabledCommands returns enabled commands.
func (c Commands) EnabledCommands() []tgbotapi.BotCommand {
	cmds := make([]tgbotapi.BotCommand, 0, len(c))

	for _, cmd := range c {
		if cmd.Enabled() {
			cmds = append(cmds, cmd.BotCommand)
		}
	}

	return cmds
}

// NewCommand creates a new Command.
func NewCommand(command, description string, enabled bool) Command {
	return Command{
		BotCommand: tgbotapi.BotCommand{
			Command:     command,
			Description: description,
		},
		enabled: enabled,
	}
}

// Bot represents Telegram bot wrapper.
type Bot struct {
	client *tgbotapi.Bot

	id          int64
	username    string
	description string
	commands    Commands
}

// Description returns bot description.
func (b *Bot) Description() string {
	return b.description
}

// ID returns bot ID.
func (b *Bot) ID() int64 {
	return b.id
}

// Client returns bot client.
func (b *Bot) Client() *tgbotapi.Bot {
	return b.client
}

// Username returns bot username.
func (b *Bot) Username() string {
	return b.username
}

// Commands returns bot commands.
func (b *Bot) Commands() Commands {
	return b.commands
}

type botOptions struct {
	commands    Commands
	description string
	username    string
}

// BotOption is a bot option.
type BotOption func(*botOptions)

// WithCommands sets bot commands.
func WithCommands(commands Commands) BotOption {
	return func(o *botOptions) {
		o.commands = commands
	}
}

// WithDescription sets bot description.
func WithDescription(description string) BotOption {
	return func(o *botOptions) {
		o.description = description
	}
}

// WithUsername sets bot username.
func WithUsername(username string) BotOption {
	return func(o *botOptions) {
		o.username = username
	}
}

// NewBot creates new bot instance.
func NewBot(ctx context.Context, token string, opts ...BotOption) (*Bot, error) {
	client, err := tgbotapi.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	self, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get bot info: %w", err)
	}

	description, err := client.GetMyDescription(&tgbotapi.GetMyDescriptionParams{
		LanguageCode: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bot description: %w", err)
	}

	registedCmds, err := client.GetMyCommands(&tgbotapi.GetMyCommandsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get bot commands: %w", err)
	}

	bot := &Bot{
		id:          self.ID,
		client:      client,
		username:    self.Username,
		description: description.Description,
		commands:    toCommands(registedCmds),
	}

	if len(opts) > 0 {
		bot.updateOnStart(ctx, opts...)
	}

	return bot, nil
}

func (b *Bot) updateOnStart(ctx context.Context, opts ...BotOption) {
	var params botOptions

	for _, opt := range opts {
		opt(&params)
	}

	if params.description != "" {
		b.maybeUpdateDescriptionBot(ctx, params.description)
	}

	if params.username != "" {
		b.maybeUpdateBotName(ctx, params.username)
	}

	if len(params.commands) > 0 {
		b.maybeUpdateCommands(ctx, params.commands)
	}
}

func (b *Bot) maybeUpdateBotName(ctx context.Context, botName string) {
	isUpToDate := b.Username() != botName

	if isUpToDate {
		log.Info(ctx, "Bot name is up to date")

		return
	}

	log.Info(ctx, "Updating bot name")

	err := b.Client().SetMyName(&tgbotapi.SetMyNameParams{
		Name: botName,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot name")

		return
	}

	log.Info(ctx, "Bot name is up to date")
}

func (b *Bot) maybeUpdateDescriptionBot(ctx context.Context, botDescription string) {
	isUpToDate := b.Description() != botDescription

	if isUpToDate {
		log.Info(ctx, "Bot description is up to date")

		return
	}

	log.Info(ctx, "Updating bot description")

	err := b.Client().SetMyDescription(&tgbotapi.SetMyDescriptionParams{
		Description: botDescription,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot description")

		return
	}

	log.Info(ctx, "Bot description is up to date")
}

func (b *Bot) maybeUpdateCommands(ctx context.Context, botCommands Commands) {
	registeredCommands := make(map[string]string, len(b.Commands()))

	for _, cmd := range b.Commands() {
		registeredCommands[cmd.Command] = cmd.Description
	}

	var equal bool

	enabled := botCommands.EnabledCommands()

	for _, cmd := range enabled {
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

	err := b.Client().SetMyCommands(&tgbotapi.SetMyCommandsParams{
		Commands: enabled,
	})
	if err != nil {
		log.WithError(ctx, err).Error("failed to set bot commands")
	}

	log.Info(ctx, "Bot commands set")
}
