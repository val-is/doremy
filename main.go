package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	doremy "github.com/val-is/doremy/src"
	"github.com/val-is/doremy/src/util"
)

func main() {
	config, err := util.LoadConfig("doremy/config.json")
	if err != nil {
		log.Printf("Error loading config: %s", err)
		return
	}

	log.Println("Starting bot...")
	session, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		log.Printf("Error creating bot: %s", err)
		return
	}

	err = session.Open()
	if err != nil {
		log.Printf("Error opening connection: %s", err)
		return
	}

	bot, err := doremy.NewBotInterface(config, session)
	if err != nil {
		log.Printf("Error creating bot interface %s", err)
		return
	}

	log.Printf("Invite link: %s", bot.GetInviteLink())

	if err := bot.Initialize(); err != nil {
		log.Printf("Error initializing bot %s", err)
		return
	}

	log.Println("Bot is now running. Terminate to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err := bot.CleanlyClose(); err != nil {
		log.Printf("Error closing bot: %s", err)
		return
	}
}
