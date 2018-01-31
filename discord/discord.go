package discord

import (
	"fmt"
	// go get github.com/bwmarrin/discordgo
	"github.com/bwmarrin/discordgo"

	// utility
	"./utils"
)

var auth_token, bot_id, channel_id string

var session *discordgo.Session

func Initialize(discord_auth_token, discord_bot_id, discord_channel_id string) {

	fmt.Println("initializing discord package")

	auth_token = discord_auth_token
	bot_id = discord_bot_id
	channel_id = discord_channel_id

	session, err := discordgo.New("Bot " + auth_token)
	utils.Check(err)

	session.AddHandler(message_handler)

	err = session.Open()
	utils.Check(err)

	defer session.Close()

	<-make(chan struct{})
	return

}

func message_handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println(m.Content)
}

func Send_messages(messages []string) {

	if len(messages) > 0 {

		for _, message := range messages {

			discord.ChannelMessageSend(channel_id, message)

		}

	}

}
