package doremy

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/val-is/doremy/src/commands"
	"github.com/val-is/doremy/src/util"
)

type Bot struct {
	Config  util.Config
	Session *discordgo.Session

	Commands map[commands.BotCommandId]commands.BotCommand
}

type BotInterface interface {
	GetInviteLink() string
	CleanlyClose() error
	Initialize() error

	messageCreate(m *discordgo.MessageCreate) error
	messageReactionAdd(m *discordgo.MessageReactionAdd) error
	messageReactionRemove(m *discordgo.MessageReactionRemove) error
}

func NewBotInterface(config util.Config, session *discordgo.Session) (BotInterface, error) {
	botInterface := Bot{
		Config:  config,
		Session: session,
	}

	botInterface.Session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if err := botInterface.messageCreate(m); err != nil {
			log.Printf("Error handling message \"%s\": %s", m.Message.Content, err)
		}
	})
	botInterface.Session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageReactionAdd) {
		if err := botInterface.messageReactionAdd(m); err != nil {
			log.Printf("Error handling reaction add event: %s", err)
		}
	})
	botInterface.Session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageReactionRemove) {
		if err := botInterface.messageReactionRemove(m); err != nil {
			log.Printf("Error handling reaction remove event: %s", err)
		}
	})

	botInterface.Commands = map[commands.BotCommandId]commands.BotCommand{
		"ping": {
			CommandDoc:  "🏓",
			CommandFunc: commands.Ping,
		},
	}

	return &botInterface, nil
}

func (b Bot) GetInviteLink() string {
	return fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=8", b.Session.State.User.ID)
}

func (b *Bot) CleanlyClose() error {
	return b.Session.Close()
}

func (b *Bot) Initialize() error {
	if err := b.Session.UpdateStatus(0, fmt.Sprintf("🌙 %shelp for more info 🌙", b.Config.Discord.Prefix)); err != nil {
		return err
	}
	return nil
}

func (b *Bot) messageCreate(m *discordgo.MessageCreate) error {
	if m.Author.ID == b.Session.State.User.ID {
		return nil
	}

	// check for prefix
	if !strings.HasPrefix(m.Content, b.Config.Discord.Prefix) {
		return nil
	}

	// parse out command
	prefixStripped := strings.TrimPrefix(m.Content, b.Config.Discord.Prefix)
	commandParts := strings.SplitN(prefixStripped, " ", 2)
	if len(commandParts) == 0 {
		b.Session.ChannelMessageSend(m.ChannelID, "Make sure to specify a command.")
		return nil
	}
	normalizedCommand := commands.BotCommandId(strings.ToLower(commandParts[0]))

	// check and see if in dms
	channel, err := b.Session.Channel(m.ChannelID)
	if err != nil {
		b.Session.ChannelMessageSend(m.ChannelID, "Error getting channel info.")
		return fmt.Errorf("Error getting channel info: %s", err)
	}
	if channel.GuildID != "" {
		b.Session.ChannelMessageSend(m.ChannelID, "This bot's really only made to be used in dms.")
		return nil
	}

	// run call and response
	if command, ok := b.Commands[normalizedCommand]; ok {
		if err := command.CommandFunc(b.Session, m, commandParts[1]); err != nil {
			b.Session.ChannelMessageSend(m.ChannelID, "There was an internal error when running the command.")
			return fmt.Errorf("Error running command: %s, %s", m.Content, err)
		}
		return nil
	}
	// get help embed
	if normalizedCommand == commands.BotCommandId("help") {
		commandHelpString := ""
		for command := range b.Commands {
			commandStruct := b.Commands[command]
			commandHelpString = fmt.Sprintf("%s\n- %s: %s", commandHelpString, command, commandStruct.CommandDoc)
		}
		helpEmbed := discordgo.MessageEmbed{
			Title: "🌙 Doremy Help",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Commands",
					Value: commandHelpString,
				},
			},
		}
		b.Session.ChannelMessageSendEmbed(m.ChannelID, &helpEmbed)
		return nil
	}

	return nil
}

func (b *Bot) messageReactionAdd(m *discordgo.MessageReactionAdd) error {
	return nil
}

func (b *Bot) messageReactionRemove(m *discordgo.MessageReactionRemove) error {
	return nil
}
