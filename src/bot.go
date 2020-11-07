package doremy

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/val-is/doremy/src/util"
)

type Bot struct {
	Config  util.Config
	Session *discordgo.Session
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
	return nil
}

func (b *Bot) messageReactionAdd(m *discordgo.MessageReactionAdd) error {
	return nil
}

func (b *Bot) messageReactionRemove(m *discordgo.MessageReactionRemove) error {
	return nil
}
