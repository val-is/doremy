package doremy

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/bwmarrin/discordgo"
)

type (
	botCommandId string
	botCommand   struct {
		CommandDoc  string
		CommandFunc func(*Bot, *discordgo.MessageCreate, string) error
	}
)

func commandPing(b *Bot, m *discordgo.MessageCreate, _ string) error {
	b.session.ChannelMessageSend(m.ChannelID, "Pong!")
	return nil
}

func commandStartSleeping(b *Bot, m *discordgo.MessageCreate, _ string) error {
	if _, pending := b.db.GetChannelPending(m.ChannelID); pending {
		b.session.ChannelMessageSend(m.ChannelID, "You're already in a sleep period.")
		b.session.ChannelMessageSend(m.ChannelID, "Either respond to the poll or cancel last period")
		return nil
	}
	if err := b.db.StartSleepSession(m.ChannelID, time.Now(), map[string]string{}); err != nil {
		return err
	}
	b.session.ChannelMessageSend(m.ChannelID, "I started a sleeping period. Good night! 🌙")
	return nil
}

func commandCancelPeriod(b *Bot, m *discordgo.MessageCreate, _ string) error {
	if _, pending := b.db.GetChannelPending(m.ChannelID); !pending {
		b.session.ChannelMessageSend(m.ChannelID, "You're not currently in a sleep period.")
		return nil
	}
	if err := b.db.DeletePendingSleepSession(m.ChannelID); err != nil {
		return err
	}
	b.session.ChannelMessageSend(m.ChannelID, "I've stopped/deleted the most recent sleep period.")
	return nil
}

func commandStopSleeping(b *Bot, m *discordgo.MessageCreate, _ string) error {
	sleepSession, pending := b.db.GetChannelPending(m.ChannelID)
	if !pending {
		b.session.ChannelMessageSend(m.ChannelID, "You're not currently in a sleep period.")
		return nil
	}
	if err := b.sendPoll(sleepSession); err != nil {
		return err
	}
	return nil
}

func commandGetData(b *Bot, m *discordgo.MessageCreate, _ string) error {
	sessions := b.db.GetSleepSessionsByUserChannel(m.ChannelID)
	sessionsJson, err := json.Marshal(sessions)
	if err != nil {
		b.session.ChannelMessageSend(m.ChannelID, "There was an issue processing your session data")
		return err
	}
	b.session.ChannelFileSend(m.ChannelID, "data.json", bytes.NewReader(sessionsJson))
	return nil
}
