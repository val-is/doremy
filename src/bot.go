package doremy

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	config  configStruct
	session *discordgo.Session
	db      dbInterface

	commands map[botCommandId]botCommand
}

type BotInterface interface {
	GetInviteLink() string
	CleanlyClose() error
	Initialize() error

	messageCreate(m *discordgo.MessageCreate) error
	messageReactionAdd(m *discordgo.MessageReactionAdd) error
	messageReactionRemove(m *discordgo.MessageReactionRemove) error

	processPollDaemon() error
	sendPoll(poll sleepSessionStruct) error
}

func NewBotInterface(config configStruct, session *discordgo.Session) (BotInterface, error) {
	botInterface := Bot{
		config:  config,
		session: session,
	}

	botInterface.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if err := botInterface.messageCreate(m); err != nil {
			log.Printf("Error handling message \"%s\": %s", m.Message.Content, err)
		}
	})
	botInterface.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageReactionAdd) {
		if err := botInterface.messageReactionAdd(m); err != nil {
			log.Printf("Error handling reaction add event: %s", err)
		}
	})
	botInterface.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageReactionRemove) {
		if err := botInterface.messageReactionRemove(m); err != nil {
			log.Printf("Error handling reaction remove event: %s", err)
		}
	})

	botInterface.commands = map[botCommandId]botCommand{
		"ping": {
			CommandDoc:  "🏓",
			CommandFunc: commandPing,
		},
		"sleep": {
			CommandDoc:  "💤",
			CommandFunc: commandStartSleeping,
		},
		"cancel": {
			CommandDoc:  "❌",
			CommandFunc: commandCancelPeriod,
		},
		"stop": {
			CommandDoc:  "🛑",
			CommandFunc: commandStopSleeping,
		},
	}

	db, err := newJSONDB(config.Datafile)
	if err != nil {
		return nil, err
	}
	botInterface.db = db

	return &botInterface, nil
}

func (b *Bot) GetInviteLink() string {
	return fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=8", b.session.State.User.ID)
}

func (b *Bot) CleanlyClose() error {
	if err := b.db.Save(); err != nil {
		return err
	}
	return b.session.Close()
}

func (b *Bot) Initialize() error {
	if err := b.session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Game: &discordgo.Game{
			Name: fmt.Sprintf("you sleep. %shelp for more info", b.config.Discord.Prefix),
			Type: discordgo.GameTypeWatching,
		},
		AFK: false,
	}); err != nil {
		return err
	}

	pollDaemonSleep, err := time.ParseDuration(fmt.Sprintf("%fm", b.config.Polling.DaemonTime))
	if err != nil {
		return err
	}

	go func() {
		for {
			if err := b.processPollDaemon(); err != nil {
				log.Printf("Error when running polling daemon: %s", err)
			}
			if err := b.db.Save(); err != nil {
				log.Printf("Error when periodically saving data: %s", err)
			}
			time.Sleep(pollDaemonSleep)
		}
	}()

	return nil
}

func (b *Bot) messageCreate(m *discordgo.MessageCreate) error {
	if m.Author.ID == b.session.State.User.ID {
		return nil
	}

	// check for prefix
	if !strings.HasPrefix(m.Content, b.config.Discord.Prefix) {
		return nil
	}

	// parse out command
	prefixStripped := strings.TrimPrefix(m.Content, b.config.Discord.Prefix)
	commandParts := strings.SplitN(prefixStripped, " ", 2)
	if len(commandParts) == 0 {
		b.session.ChannelMessageSend(m.ChannelID, "Make sure to specify a command.")
		return nil
	}
	if len(commandParts) == 1 {
		commandParts = append(commandParts, "")
	}
	normalizedCommand := botCommandId(strings.ToLower(commandParts[0]))

	// check and see if in dms
	channel, err := b.session.Channel(m.ChannelID)
	if err != nil {
		b.session.ChannelMessageSend(m.ChannelID, "Error getting channel info.")
		return fmt.Errorf("error getting channel info: %s", err)
	}
	if channel.GuildID != "" {
		b.session.ChannelMessageSend(m.ChannelID, "This bot's really only made to be used in dms.")
		return nil
	}

	// run call and response
	if command, ok := b.commands[normalizedCommand]; ok {
		if err := command.CommandFunc(b, m, commandParts[1]); err != nil {
			b.session.ChannelMessageSend(m.ChannelID, "There was an internal error when running the command.")
			return fmt.Errorf("error running command: %s, %s", m.Content, err)
		}
		return nil
	}
	// get help embed quine thing
	if normalizedCommand == botCommandId("help") {
		commandHelpString := ""
		for command := range b.commands {
			commandStruct := b.commands[command]
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
		b.session.ChannelMessageSendEmbed(m.ChannelID, &helpEmbed)
		return nil
	}

	return nil
}

func (b *Bot) messageReactionAdd(m *discordgo.MessageReactionAdd) error {
	if m.MessageReaction.UserID == b.session.State.User.ID {
		return nil
	}

	// less generalized
	if !b.db.GetPollActive(m.MessageID) {
		return nil
	}

	pollResult := -1
	for val, emoji := range b.config.Polling.Emojis {
		if emoji == m.MessageReaction.Emoji.Name {
			pollResult = val
			break
		}
	}
	if pollResult == -1 {
		return nil
	}

	sessionRecap, err := b.db.EndSleepSession(m.ChannelID, time.Now(), pollResult, map[string]string{})
	if err != nil {
		b.session.ChannelMessageSend(m.ChannelID, "There was an internal error when handling the reaction")
		return fmt.Errorf("error when handling reaction: %s", err)
	}

	hSlept := int(sessionRecap.Duration.Hours())
	mSlept := int(sessionRecap.Duration.Minutes()) - 60*hSlept
	b.session.ChannelMessageSend(m.ChannelID,
		fmt.Sprintf("You slept for %d hours, %d minute(s)", hSlept, mSlept))

	return nil
}

func (b *Bot) messageReactionRemove(m *discordgo.MessageReactionRemove) error {
	if m.UserID == b.session.State.User.ID {
		return nil
	}

	// TODO updating poll results is not in scope of POC.

	return nil
}

func (b *Bot) processPollDaemon() error {
	unaddedPolls := b.db.GetAllUnaddedPolls()
	pollDelay, err := time.ParseDuration(fmt.Sprintf("%fm", b.config.Polling.SleepPeriodDuration))
	if err != nil {
		return err
	}
	for _, poll := range unaddedPolls {
		if time.Since(poll.Start) < pollDelay {
			continue
		}
		if err := b.sendPoll(poll); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) sendPoll(poll sleepSessionStruct) error {
	b.session.ChannelMessageSend(poll.ChannelID, "Good morning!")
	message, _ := b.session.ChannelMessageSend(poll.ChannelID, "React to how you feel rn (1 is bad, 5 is good)")
	for i := 0; i < 5; i++ {
		b.session.MessageReactionAdd(message.ChannelID, message.ID, b.config.Polling.Emojis[i])
	}
	if err := b.db.AddPollMessage(poll.ChannelID, message.ID); err != nil {
		return err
	}
	return nil
}
