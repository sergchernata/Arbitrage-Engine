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

var auth_token, bot_id, channel_id, host, database, username, password string

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

	// don't talk to itself
	if author_id == bot_id {
		return
	}

	// trim spaces and lowercase
	content := strings.ToLower(strings.Trim(m.Content, " "))

	mongo.Check_discord_user_exists(author_id, author_username, author_channel_id)

	if content == "help" {

		message := "Alright ~~dipshit~~ " + author_username + ", here's a list of available commands. Some contain a small example at the end.\n"
		message += "Don't type multiple commands per message; one at a time.\n\n"
		message += "```ini\n"
		message += "[on]      Turn on the bot\n"
		message += "[off]     Turn off the bot\n"
		message += "\n"
		message += "[add]     Add token to be monitored, ex: 'add OMG'\n"
		message += "[remove]  Remove token from monitoring, ex: 'remove OMG'\n"
		message += "[show]    Show a list of tokens that are being monitored\n"
		message += "\n"
		message += "[set]     Threshold for notifications in percent, ex: 'set 5'\n"
		message += "```"

		messages = append(messages, message)

	} else {

		messages = append(messages, "use `help` for a list of available commands")

	}

	for _, message := range messages {

		s.ChannelMessageSend(author_channel_id, message)

	}

}

func Send_messages(messages []string) {

	if len(messages) > 0 {

		for _, message := range messages {

			session.ChannelMessageSend(channel_id, message)

		}

	}

}
