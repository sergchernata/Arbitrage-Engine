package discord

import (
	"fmt"
	"strings"
	"time"

	// go get github.com/bwmarrin/discordgo
	"github.com/bwmarrin/discordgo"

	// database package
	"../db/mongo"

	// utility
	"../utils"
)

var auth_token, bot_id, channel_id, host, database, username, password string

var session *discordgo.Session

var errors = map[string]string{

	"db_error": "I failed to connect to database, I am ashamed",
}

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
	var err error

	// initialize database connection
	mongo.Initialize(host, database, username, password)

	// initialize discord bot
	session, err = discordgo.New("Bot " + auth_token)
	utils.Check(err)

	session.AddHandler(message_handler)

	err = session.Open()
	utils.Check(err)

	// defer session.Close()

}

func message_handler(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := ""
	author_id := m.Author.ID
	author_username := m.Author.Username
	author_channel_id := m.ChannelID

	// don't talk to itself
	if author_id == bot_id {
		return
	}

	// check if this user is already within database
	discorder := mongo.Get_discorder(author_id)

	if discorder.ID == "" {

		discorder := utils.Discorder{
			ID:        author_id,
			Username:  author_username,
			Channel:   author_channel_id,
			On:        false,
			Threshold: 5,
			Timestamp: time.Now(),
		}

		// if not, create him
		mongo.Create_discorder(discorder)
	}

	// trim spaces and lowercase
	content := strings.ToLower(strings.Trim(m.Content, " "))

	if content == "on" {

		done := false

		if done {
			message = "Ok, I'll monitor the prices for you"
		} else {
			message = errors["db_error"]
		}

	} else if content == "off" {

		done := false

		if done {
			message = "I will no longer monitor the prices for you"
		} else {
			message = errors["db_error"]
		}

	} else if strings.HasPrefix(content, "add ") {

		done := false
		token := ""

		if done {
			message = "Ok, I'll monitor " + token + " as well"
		} else {
			message = errors["db_error"]
		}

	} else if strings.HasPrefix(content, "remove ") {

		done := false
		token := ""

		if done {
			message = "Ok, I will no longer monitor " + token
		} else {
			message = errors["db_error"]
		}

	} else if content == "show" {

		done := false

		if done {
			message = ""
		} else {
			message = errors["db_error"]
		}

	} else if strings.HasPrefix(content, "set ") {

		done := false

		if done {
			message = ""
		} else {
			message = errors["db_error"]
		}

	} else if content == "help" {

		message = "Alright ~~dipshit~~ " + author_username + ", here's a list of available commands. Some contain a small example at the end.\n"
		message += "Don't type multiple commands per message; send one at a time.\n\n"
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

	} else if strings.Contains(content, "serg") {

		message = "Please don't talk about my master"

	} else if is_potty_mouth(content) {

		message = "There's no need for that kind of language"

	} else {

		message = "Say `help` for a list of available commands"

	}

	s.ChannelMessageSend(author_channel_id, message)

}

func Send_messages(messages []string) {

	if len(messages) > 0 {

		for _, message := range messages {

			session.ChannelMessageSend(channel_id, message)

		}

	}

}

func is_potty_mouth(message string) bool {

	bad_words := []string{"fuck", "shit", "dick", "bitch", "cunt"}

	for _, word := range bad_words {
		if strings.Contains(message, word) {
			return true
		}
	}

	return false

}
