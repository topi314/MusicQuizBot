package quizbot

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func (b *Bot) OnApplicationCommandInteractionCreate(event *events.ApplicationCommandInteractionCreate) {
	command, ok := b.Commands[event.Data.CommandName()]
	if !ok {
		return
	}
	if err := command.Handler(b, event); err != nil {
		b.Client.Logger().Error("Failed to handle autocomplete: ", err)
	}
}

func (b *Bot) OnAutocompleteInteractionCreate(event *events.AutocompleteInteractionCreate) {
	command, ok := b.Commands[event.Data.CommandName]
	if !ok {
		return
	}
	if err := command.AutocompleteHandler(b, event); err != nil {
		b.Client.Logger().Error("Failed to handle autocomplete: ", err)
	}
}

func (b *Bot) AddCommands(commands ...Command) {
	var commandCreates []discord.ApplicationCommandCreate
	for _, command := range commands {
		b.Commands[command.Create.Name()] = command
		commandCreates = append(commandCreates, command.Create)
	}
	if b.Config.SyncCommands {
		b.Client.Logger().Info("Syncing commands...")
		var err error
		if b.Config.DevMode {
			_, err = b.Client.Rest().SetGuildCommands(b.Client.ApplicationID(), b.Config.GuildID, commandCreates)
		} else {
			_, err = b.Client.Rest().SetGlobalCommands(b.Client.ApplicationID(), commandCreates)
		}
		if err != nil {
			b.Client.Logger().Error("Failed to set commands: ", err)
		}
	}
}

type Command struct {
	Create              discord.ApplicationCommandCreate
	Handler             func(b *Bot, event *events.ApplicationCommandInteractionCreate) error
	AutocompleteHandler func(b *Bot, event *events.AutocompleteInteractionCreate) error
}
