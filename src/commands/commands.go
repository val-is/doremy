package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/val-is/doremy/src/util"
)

type BotCommandId string
type BotCommand struct {
	CommandDoc  string
	CommandFunc func(*discordgo.Session, util.DBInterface, *discordgo.MessageCreate, string) error
}

func Ping(s *discordgo.Session, _ util.DBInterface, m *discordgo.MessageCreate, _ string) error {
	if _, err := s.ChannelMessageSend(m.ChannelID, "Pong!"); err != nil {
		return err
	}
	return nil
}
