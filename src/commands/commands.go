package commands

import (
	"github.com/bwmarrin/discordgo"
)

type BotCommandId string
type BotCommand struct {
	CommandDoc  string
	CommandFunc func(*discordgo.Session, *discordgo.MessageCreate, string) error
}

func Ping(s *discordgo.Session, m *discordgo.MessageCreate, _ string) error {
	if _, err := s.ChannelMessageSend(m.ChannelID, "Pong!"); err != nil {
		return err
	}
	return nil
}
