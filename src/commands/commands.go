package commands

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/val-is/doremy/src/util"
)

type BotCommandId string
type BotCommand struct {
	CommandDoc  string
	CommandFunc func(*discordgo.Session, util.DBInterface, *discordgo.MessageCreate, string) error
}

func Ping(s *discordgo.Session, _ util.DBInterface, m *discordgo.MessageCreate, _ string) error {
	s.ChannelMessageSend(m.ChannelID, "Pong!")
	return nil
}

func StartSleeping(s *discordgo.Session, db util.DBInterface, m *discordgo.MessageCreate, _ string) error {
	if _, pending := db.GetChannelPending(m.ChannelID); pending {
		s.ChannelMessageSend(m.ChannelID, "You're already in a sleep period.")
		s.ChannelMessageSend(m.ChannelID, "Either respond to the poll or cancel last period")
		return nil
	}
	if err := db.StartSleepSession(m.ChannelID, time.Now(), map[string]string{}); err != nil {
		return err
	}
	s.ChannelMessageSend(m.ChannelID, "I started a sleeping period. Good night! 🌙")
	return nil
}

func CancelPeriod(s *discordgo.Session, db util.DBInterface, m *discordgo.MessageCreate, _ string) error {
	if _, pending := db.GetChannelPending(m.ChannelID); !pending {
		s.ChannelMessageSend(m.ChannelID, "You're not currently in a sleep period.")
		return nil
	}
	if err := db.DeletePendingSleepSession(m.ChannelID); err != nil {
		return err
	}
	s.ChannelMessageSend(m.ChannelID, "I've stopped/deleted the most recent sleep period.")
	return nil
}
