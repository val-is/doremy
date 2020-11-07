package doremy

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/val-is/doremy/src/commands"
	"github.com/val-is/doremy/src/util"
)

type Bot struct {
	Config  util.Config
	Session *discordgo.Session
	DB      util.DBInterface

	Commands map[commands.BotCommandId]commands.BotCommand
}

type BotInterface interface {
	GetInviteLink() string
	CleanlyClose() error
	Initialize() error

	messageCreate(m *discordgo.MessageCreate) error
	messageReactionAdd(m *discordgo.MessageReactionAdd) error
	messageReactionRemove(m *discordgo.MessageReactionRemove) error

	processPollDaemon() error
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
		"sleep": {
			CommandDoc:  "💤",
			CommandFunc: commands.StartSleeping,
		},
		"cancel": {
			CommandDoc:  "🛑",
			CommandFunc: commands.CancelPeriod,
		},
	}

	db, err := util.NewJSONDB(config.Datafile)
	if err != nil {
		return nil, err
	}
	botInterface.DB = db

	return &botInterface, nil
}

func (b Bot) GetInviteLink() string {
	return fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=8", b.Session.State.User.ID)
}

func (b *Bot) CleanlyClose() error {
	if err := b.DB.Save(); err != nil {
		return err
	}
	return b.Session.Close()
}

func (b *Bot) Initialize() error {
	if err := b.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Game: &discordgo.Game{
			Name: fmt.Sprintf("you sleep. %shelp for more info", b.Config.Discord.Prefix),
			Type: discordgo.GameTypeWatching,
		},
		AFK: false,
	}); err != nil {
		return err
	}

	pollDaemonSleep, err := time.ParseDuration(fmt.Sprintf("%fm", b.Config.Polling.DaemonTime))
	if err != nil {
		return err
	}

	go func() {
		for {
			if err := b.processPollDaemon(); err != nil {
				log.Printf("Error when running polling daemon: %s", err)
			}
			if err := b.DB.Save(); err != nil {
				log.Printf("Error when periodically saving data: %s", err)
			}
			time.Sleep(pollDaemonSleep)
		}
	}()

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
	if len(commandParts) == 1 {
		commandParts = append(commandParts, "")
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
		if err := command.CommandFunc(b.Session, b.DB, m, commandParts[1]); err != nil {
			b.Session.ChannelMessageSend(m.ChannelID, "There was an internal error when running the command.")
			return fmt.Errorf("Error running command: %s, %s", m.Content, err)
		}
		return nil
	}
	// get help embed quine thing
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
	if m.MessageReaction.UserID == b.Session.State.User.ID {
		return nil
	}

	// less generalized
	if !b.DB.GetPollActive(m.MessageID) {
		return nil
	}

	pollResult := -1
	for val, emoji := range b.Config.Polling.Emojis {
		if emoji == m.MessageReaction.Emoji.Name {
			pollResult = val
			break
		}
	}
	if pollResult == -1 {
		return nil
	}

	sessionRecap, err := b.DB.EndSleepSession(m.ChannelID, time.Now(), pollResult, map[string]string{})
	if err != nil {
		b.Session.ChannelMessageSend(m.ChannelID, fmt.Sprintf("There was an internal error when handling the reaction"))
		return fmt.Errorf("Error when handling reaction: %s", err)
	}

	hSlept := int(sessionRecap.Duration.Hours())
	mSlept := int(sessionRecap.Duration.Minutes()) - 60*hSlept
	b.Session.ChannelMessageSend(m.ChannelID,
		fmt.Sprintf("You slept for %d hours, %d minute(s)", hSlept, mSlept))

	return nil
}

func (b *Bot) messageReactionRemove(m *discordgo.MessageReactionRemove) error {
	if m.UserID == b.Session.State.User.ID {
		return nil
	}

	// TODO updating poll results is not in scope of POC.

	return nil
}

func (b *Bot) processPollDaemon() error {
	unaddedPolls := b.DB.GetAllUnaddedPolls()
	pollDelay, err := time.ParseDuration(fmt.Sprintf("%fm", b.Config.Polling.SleepPeriodDuration))
	if err != nil {
		return err
	}
	for _, poll := range unaddedPolls {
		if time.Now().Sub(poll.Start) > pollDelay {
			b.Session.ChannelMessageSend(poll.ChannelID, "Good morning!")
			message, _ := b.Session.ChannelMessageSend(poll.ChannelID, "React to how you feel rn (1 is bad, 5 is good)")
			for i := 0; i < 5; i++ {
				b.Session.MessageReactionAdd(message.ChannelID, message.ID, b.Config.Polling.Emojis[i])
			}
			if err := b.DB.AddPollMessage(poll.ChannelID, message.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
