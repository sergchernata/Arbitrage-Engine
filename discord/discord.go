package discord

import (
	"fmt"
	"strconv"
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

var status = []string{
	// movies
	"Wolf of Wall Street",
	"Capitalism: A Love Story",
	"Boiler Room",
	"Wall Street",
	"Margin Call",
	"The Big Short",
	"Wall Street: Money Never Sleeps",
}

func Initialize(discord_auth_token, discord_bot_id, discord_channel_id string) {

	fmt.Println("initializing discord package")

	auth_token = discord_auth_token
	bot_id = discord_bot_id
	channel_id = discord_channel_id
	var err error

	// initialize discord bot
	session, err = discordgo.New("Bot " + auth_token)
	utils.Check(err)

	session.AddHandler(message_handler)
	session.AddHandler(ready)

	err = session.Open()
	utils.Check(err)

	// defer session.Close()

}

func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "Wolf of Wall Street")
}

func message_handler(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := ""
	new_user := false
	author_id := m.Author.ID
	author_username := m.Author.Username
	author_channel_id := m.ChannelID
	channel, _ := s.State.Channel(m.ChannelID)

	// don't talk to itself and don't respond within group channels
	if author_id == bot_id || channel.Type != discordgo.ChannelTypeDM {
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
			Frequency: 5,
			Timestamp: time.Now(),
		}

		// if not, create him
		mongo.Create_discorder(discorder)

		new_user = true
	}

	// trim spaces
	content := strings.ToLower(strings.Trim(m.Content, " "))

	if content == "on" {

		done := mongo.Discorder_toggle(author_id, true)

		if done {
			message = "Ok, I'll monitor the prices for you."
		} else {
			message = errors["db_error"]
		}

	} else if content == "off" {

		done := mongo.Discorder_toggle(author_id, false)

		if done {
			message = "I will no longer monitor the prices for you."
		} else {
			message = errors["db_error"]
		}

	} else if strings.HasPrefix(content, "add") {

		parts := strings.Split(content, " ")

		if len(parts) > 1 {

			token := strings.ToUpper(strings.Split(content, " ")[1])
			listed_on := mongo.Get_listed_token_exchanges(token)

			if len(listed_on) > 0 {

				if utils.StringInSlice(token, discorder.Tokens) {
					message = token + " is already being monitored."
				} else {
					done := mongo.Discorder_update_tokens(author_id, "$addToSet", token)

					if done {
						message = "Ok, I'll monitor " + token + " as well."
					} else {
						message = errors["db_error"]
					}
				}

			} else {
				message = token + " is not currently supported. The exchanges I monitor aren't trading it on ETH pair."
			}

		} else {
			message = "`add` command requires a second paremeter, ex: `add OMG`"
		}

	} else if strings.HasPrefix(content, "remove") {

		parts := strings.Split(content, " ")

		if len(parts) > 1 {

			token := strings.ToUpper(strings.Split(content, " ")[1])
			done := mongo.Discorder_update_tokens(author_id, "$pull", token)

			if done {
				if token == "ALL" {
					message = "Ok, I have erased all of the tokens you've told me to monitor."
				} else {
					message = "Ok, I will no longer monitor " + token + "."
				}

			} else {
				message = errors["db_error"]
			}

		} else {
			message = "`remove` command requires a second paremeter. A token or keyword `ALL`, ex: `remove OMG`"
		}

	} else if strings.HasPrefix(content, "info") {

		parts := strings.Split(content, " ")

		if len(parts) > 1 {

			token := strings.ToUpper(parts[1])
			analysis := mongo.Get_token_analysis(token)
			listeded_on := mongo.Get_listed_token_exchanges(token)

			// format for display
			avg_diff := strconv.FormatFloat(analysis.Avg_diff, 'f', 2, 64)
			min_diff := strconv.FormatFloat(analysis.Min_diff, 'f', 2, 64)
			max_diff := strconv.FormatFloat(analysis.Max_diff, 'f', 2, 64)
			num_exchanges := strconv.Itoa(len(listeded_on))
			exchanges := strings.Join(listeded_on, ", ")

			// format the time timestamp of max price event
			//time_now := time.Now()
			time_of_max := analysis.Max_diff_time
			date_of_max := time_of_max.Format("02/01/06 at 03:04")
			//time_diff := time_now.Sub(time_of_max)

			if len(listeded_on) > 1 {
				message = "```ini\n"
				message += "Difference: Avg [" + avg_diff + "%] | Max [" + max_diff + "%] |  Min [" + min_diff + "%]\n"
				message += "--------------------------------------------------------------\n"
				message += "Max diff. between " + analysis.Max_diff_min_exch + "(min) and " + analysis.Max_diff_max_exch + "(max) on " + date_of_max + " \n"
				message += "--------------------------------------------------------------\n"
				message += "Listed on [" + num_exchanges + "]: " + exchanges + ".\n\n"
				message += "```"
			} else if len(listeded_on) == 1 {
				message = "```ini\n"
				message += token + " is traded on less than 2 exchanges that I monitor.\n"
				message += "Arbitrage analysis is unavailable.\n"
				message += "--------------------------------------------------------------\n"
				message += "Listed on [" + num_exchanges + "]: " + exchanges + ".\n\n"
				message += "```"
			} else {
				message = "None of the exchanges I monitor trade " + token + " on ETH pair."
			}

		} else {
			message = "`info` command requires a second paremeter, ex: `info OMG`"
		}

	} else if content == "status" {

		threshold := strconv.FormatFloat(discorder.Threshold, 'f', 2, 64)
		frequency := strconv.FormatFloat(discorder.Frequency, 'f', 2, 64)
		status := "on"
		tokens := "none"

		if len(discorder.Tokens) > 0 {
			tokens = strings.Join(discorder.Tokens, ", ")
		}

		if !discorder.On {
			status = "off"
		}

		message = "```ini\n"
		message += "Notifications [" + status + "] | Frequency [" + frequency + " min] | Threshold [" + threshold + "%]\n"
		message += "--------------------------------------------------------------\n"
		message += "Tokens: " + tokens + ".\n\n"
		message += "```"

	} else if strings.HasPrefix(content, "threshold ") {

		t := strings.Split(content, " ")[1]
		threshold, err := strconv.ParseFloat(t, 64)

		if err == nil {
			done := mongo.Discorder_set_threshold(author_id, threshold)

			if done {
				message = "Ok, I changed the notification threshold to " + t + "%"
			} else {
				message = errors["db_error"]
			}
		} else {
			message = "Threshold must be a number, decimal or integer."
		}

	} else if strings.HasPrefix(content, "frequency ") {

		t := strings.Split(content, " ")[1]
		frequency, err := strconv.ParseFloat(t, 64)

		if err == nil {
			done := mongo.Discorder_set_frequency(author_id, frequency)

			if done {
				message = "Ok, I changed the notification frequency to " + t + " minutes"
			} else {
				message = errors["db_error"]
			}
		} else {
			message = "Frequency must be a number, decimal or integer."
		}

	} else if content == "help" {

		// message = "Alright, here's a list of available commands. Some contain a small example at the end.\n"
		// message += "Don't type multiple commands per message; send one at a time.\n\n"
		message = "```ini\n"
		message += "[on]          Turn on the bot\n"
		message += "[off]         Turn off the bot\n"
		message += "[status]      Show all settings and current bot status\n"
		message += "---\n"
		message += "[add]         Add token to be monitored, ex: 'add OMG'\n"
		message += "[remove]      Accepts token symbol or 'ALL' keyword.\n"
		message += "[info]        Last 30 days of token performance, ex: 'info NULS'\n"
		message += "---\n"
		message += "[threshold]   Threshold for notifications in percent, ex: 'threshold 5'\n"
		message += "[frequency]   Frequency of notifications in minutes, ex: 'frequency 5'\n"
		message += "---\n"
		message += "[about]       Show supported exhanges.\n"
		message += "```"

	} else if content == "about" {

		status := "on"

		if !discorder.On {
			status = "off"
		}

		message = "```ini\n"
		message += "Your notifications are turned [" + status + "]\n"
		message += "---\n"
		message += "I support monitoring of all ETH pairs on the following exchanges:\n\n"
		message += "• Binance\n"
		message += "• Kucoin\n"
		message += "• OKex\n"
		message += "• Bitz\n"
		message += "```"

	} else if strings.Contains(content, "serg") {

		message = "Please don't talk about my master"

	} else if is_potty_mouth(content) {

		message = "There's no need for that kind of language"

	} else {

		if new_user {
			message = "Nice to meet you. "
		}
		message += "Say `help` for a list of available commands"

	}

	s.ChannelMessageSend(author_channel_id, message)

}

// notify users who currently have an active bot configuraiton
// since the bot can be toggled on / off
func Notify_discorders(comparisons map[string]utils.Comparison) {

	discorders := mongo.Get_active_discorders()

	if len(discorders) > 0 {

		for _, d := range discorders {

			message := ""

			for token, comparison := range comparisons {

				// calculte percentage difference
				difference := (1 - comparison.Min_price/comparison.Max_price) * 100
				difference = utils.ToFixed(difference, 0)

				// if this is a token the user wants us to monitor
				// and the notification threshold matches set prefernce
				token_match := utils.StringInSlice(token, d.Tokens)
				threshold_match := difference >= d.Threshold
				frequency_match := time.Since(d.Last_notification).Minutes() >= d.Frequency

				if token_match && threshold_match && frequency_match {

					string_diff := strconv.FormatFloat(difference, 'f', 0, 64)
					message += token + " " + string_diff + "% difference between "
					message += comparison.Min_exchange + "(min) and " + comparison.Max_exchange + "(max)" + " on ETH pair\n"

				}

			}

			if message != "" {

				session.ChannelMessageSend(d.Channel, message)
				mongo.Discorder_update_notification_time(d.ID)

			}

		}

	}

}

// personal method, for notifying me in a shared channel
func Send_messages(messages map[string]string) {

	if len(messages) > 0 {

		for _, message := range messages {

			session.ChannelMessageSend(channel_id, message)

		}

	}

}

func Send_daily_summary(message string) {

	if message != "" {

		session.ChannelMessageSend(channel_id, message)

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
