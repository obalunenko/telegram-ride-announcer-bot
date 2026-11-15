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

	id               int64
	username         string
	name             string
	description      string
	shortDescription string
	commands         Commands
	// menu holds the commands declared with WithCommands.
	menu Commands
}

// Description returns bot description.
func (b *Bot) Description() string {
	return b.description
}

// ShortDescription returns the bot short description shown on its profile page.
func (b *Bot) ShortDescription() string {
	return b.shortDescription
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

// Name returns bot name shown in chats.
func (b *Bot) Name() string {
	return b.name
}

// Commands returns the commands registered in Telegram.
func (b *Bot) Commands() Commands {
	return b.commands
}

// Menu returns the enabled commands declared with WithCommands, even when Telegram
// did not accept them on start. Without declared commands, it returns the registered ones.
func (b *Bot) Menu() []tgbotapi.BotCommand {
	if len(b.menu) == 0 {
		return b.commands.EnabledCommands()
	}

	return b.menu.EnabledCommands()
}

type botOptions struct {
	commands         Commands
	description      string
	shortDescription string
	name             string
	clientOptions    []tgbotapi.BotOption
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

// WithShortDescription sets bot short description shown on its profile page and in shared links.
func WithShortDescription(shortDescription string) BotOption {
	return func(o *botOptions) {
		o.shortDescription = shortDescription
	}
}

// WithName sets bot name shown in chats. The username of a bot can't be changed via the Bot API.
func WithName(name string) BotOption {
	return func(o *botOptions) {
		o.name = name
	}
}

// WithClientOptions sets options of the underlying telego client, e.g. a custom Bot API server.
func WithClientOptions(opts ...tgbotapi.BotOption) BotOption {
	return func(o *botOptions) {
		o.clientOptions = append(o.clientOptions, opts...)
	}
}

// NewBot creates new bot instance.
func NewBot(ctx context.Context, token string, opts ...BotOption) (*Bot, error) {
	var params botOptions

	for _, opt := range opts {
		opt(&params)
	}

	client, err := tgbotapi.NewBot(token, params.clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	self, err := client.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot info: %w", err)
	}

	description, err := client.GetMyDescription(ctx, &tgbotapi.GetMyDescriptionParams{
		LanguageCode: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bot description: %w", err)
	}

	shortDescription, err := client.GetMyShortDescription(ctx, &tgbotapi.GetMyShortDescriptionParams{
		LanguageCode: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bot short description: %w", err)
	}

	registedCmds, err := client.GetMyCommands(ctx, &tgbotapi.GetMyCommandsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get bot commands: %w", err)
	}

	bot := &Bot{
		id:               self.ID,
		client:           client,
		username:         self.Username,
		name:             self.FirstName,
		description:      description.Description,
		shortDescription: shortDescription.ShortDescription,
		commands:         toCommands(registedCmds),
		menu:             params.commands,
	}

	bot.updateOnStart(ctx, params)

	return bot, nil
}

func (b *Bot) updateOnStart(ctx context.Context, params botOptions) {
	if params.description != "" {
		b.maybeUpdateDescriptionBot(ctx, params.description)
	}

	if params.shortDescription != "" {
		b.maybeUpdateShortDescription(ctx, params.shortDescription)
	}

	if params.name != "" {
		b.maybeUpdateBotName(ctx, params.name)
	}

	if len(params.commands) > 0 {
		b.maybeUpdateCommands(ctx, params.commands)
	}
}

func (b *Bot) maybeUpdateBotName(ctx context.Context, botName string) {
	isUpToDate := b.Name() == botName

	if isUpToDate {
		log.Info(ctx, "Bot name is up to date")

		return
	}

	log.Info(ctx, "Updating bot name")

	err := b.Client().SetMyName(ctx, &tgbotapi.SetMyNameParams{
		Name: botName,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot name")

		return
	}

	b.name = botName

	log.Info(ctx, "Bot name is up to date")
}

func (b *Bot) maybeUpdateDescriptionBot(ctx context.Context, botDescription string) {
	isUpToDate := b.Description() == botDescription

	if isUpToDate {
		log.Info(ctx, "Bot description is up to date")

		return
	}

	log.Info(ctx, "Updating bot description")

	err := b.Client().SetMyDescription(ctx, &tgbotapi.SetMyDescriptionParams{
		Description: botDescription,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot description")

		return
	}

	b.description = botDescription

	log.Info(ctx, "Bot description is up to date")
}

func (b *Bot) maybeUpdateShortDescription(ctx context.Context, shortDescription string) {
	if b.ShortDescription() == shortDescription {
		log.Info(ctx, "Bot short description is up to date")

		return
	}

	log.Info(ctx, "Updating bot short description")

	err := b.Client().SetMyShortDescription(ctx, &tgbotapi.SetMyShortDescriptionParams{
		ShortDescription: shortDescription,
	})
	if err != nil {
		log.WithError(ctx, err).Error("Failed to set bot short description")

		return
	}

	b.shortDescription = shortDescription

	log.Info(ctx, "Bot short description is up to date")
}

func (b *Bot) maybeUpdateCommands(ctx context.Context, botCommands Commands) {
	registeredCommands := make(map[string]string, len(b.Commands()))

	for _, cmd := range b.Commands() {
		registeredCommands[cmd.Command] = cmd.Description
	}

	enabled := botCommands.EnabledCommands()

	// Equal when the bot has exactly the enabled commands with the same descriptions.
	equal := len(enabled) == len(registeredCommands)

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

	err := b.Client().SetMyCommands(ctx, &tgbotapi.SetMyCommandsParams{
		Commands: enabled,
	})
	if err != nil {
		log.WithError(ctx, err).Error("failed to set bot commands")

		return
	}

	b.commands = toCommands(enabled)

	log.Info(ctx, "Bot commands set")
}
