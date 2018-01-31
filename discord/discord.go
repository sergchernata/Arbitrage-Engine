package discord

import (
	"fmt"
	"strings"

	// go get github.com/bwmarrin/discordgo
	"github.com/bwmarrin/discordgo"

	// database package
	"../db/mongo"

	// utility
	"../utils"
)

var auth_token, bot_id, channel_id string

var session *discordgo.Session

func Initialize(discord_auth_token, discord_bot_id, discord_channel_id,
	mongo_host, mongo_database, mongo_username, mongo_password string) {

	fmt.Println("initializing discord package")

	auth_token = discord_auth_token
	bot_id = discord_bot_id
	channel_id = discord_channel_id
	host = mongo_host
	database = mongo_database
	username = mongo_username
	password = mongo_password

	// initialize database connection
	mongo.Initialize(host, database, username, password)

	// initialize discord bot
	session, err := discordgo.New("Bot " + auth_token)
	utils.Check(err)

	session.AddHandler(message_handler)

	err = session.Open()
	utils.Check(err)

	// defer session.Close()

	<-make(chan struct{})
	return

}

func message_handler(s *discordgo.Session, m *discordgo.MessageCreate) {

	var messages []string
	author_id := m.Author.ID
	author_username := m.Author.Username
	author_channel_id := m.ChannelID
	content := strings.Replace(m.Content, " ", "", -1)

	if content == "help" {
		s.ChannelMessageSend(author_channel_id, "helping you")
	}

}

func Send_messages(messages []string) {

	if len(messages) > 0 {

		for _, message := range messages {

			session.ChannelMessageSend(channel_id, message)

		}

	}

}
