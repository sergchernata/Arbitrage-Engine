package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	// individual exchange packages
	"./exchanges/binance"
	"./exchanges/bitz"
	"./exchanges/kucoin"
	"./exchanges/okex"

	// database package
	"./db/mongo"

	// discord bot
	"./discord"

	// package for running cron jobs
	// go get github.com/jasonlvhit/gocron
	"github.com/jasonlvhit/gocron"

	// utility
	"./utils"
)

// holds environment variables
// such as api endpoints and their keys
var props = make(map[string]string)

// golang doesn't like detecting existance of key within an array
// giving every key a boolean makes for easy checks of existance
// each key is a token symbol, ie REQ, LINK, etc
var tokens = make(map[string]bool)

// started with holding exchange prices in individual variables
// but as we add more exchanges it becomes a hassle to pass around
// all those variables, so let's hold them in one map
// ex: ["binance"]["REQ-ETH"] = 0.000412
var exchange_prices = make(map[string]map[string]float64)

// same story and structure as exchange_prices above
var exchange_balances = make(map[string]map[string]float64)

// same story and structure as exchange_prices above
var listed_tokens = make(map[string][]string)

// each key is a token symbol, matching the array of tokens above
// each value is the number of tokens to be sold at once per trade
var trade_quantity = make(map[string]int)

// comparisons are stored per token
// Ex: ["NULS"] = {"Min_price" : 0.04, ...}
var comparisons = make(map[string]utils.Comparison)

// currently limited to holding ETH fee
// charged by each exchange upon withdrawal
var fees = make(map[string]float64)

// percentage threshold is the difference between min and max price
// required for us to profit from running arbitrage
var percent_threshold float64

// threshold for writing a message to discord
var discord_percent_threshold float64

func init() {

	fmt.Println("initializing main package")

	// process environment variables
	dat, err := ioutil.ReadFile(".env")
	utils.Check(err)

	lines := strings.Split(string(dat), "\n")

	for _, line := range lines {
		// check for blank and comment lines in .env config
		if line != "" && !strings.HasPrefix(line, "#") {
			split := strings.Split(line, "=")

			if split[0] == "TOKENS" {

				no_spaces := strings.Replace(split[1], " ", "", -1)
				pairs := strings.Split(no_spaces, ",")

				// parse tokens and trade quantities
				for _, pair := range pairs {
					temp := strings.Split(pair, ":")
					quantity, _ := strconv.Atoi(temp[1])

					// tokens with trade quantity of 0
					// won't be traded, but will be tracked
					tokens[temp[0]] = true
					trade_quantity[temp[0]] = quantity
				}

			} else if strings.HasSuffix(split[0], "_FEE") {

				exchange := strings.Split(split[0], "_")[0]
				exchange = strings.ToLower(exchange)
				fee, err := strconv.ParseFloat(split[1], 64)
				utils.Check(err)

				fees[exchange] = fee

			} else {

				props[split[0]] = split[1]

			}
		}
	}

	// initialize any secondary variables
	// that need to be globablly available
	percent_threshold, err = strconv.ParseFloat(props["PERCENT_THRESHOLD"], 64)
	utils.Check(err)

	discord_percent_threshold, err = strconv.ParseFloat(props["DISCORD_PERCENT_THRESHOLD"], 64)
	utils.Check(err)

	// initialize database connection
	mongo.Initialize(props["HOST"], props["DATABASE"], props["USERNAME"], props["PASSWORD"])

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"], props["BINANCE_ETH_FEE"])
	kucoin.Initialize(props["KUCOIN_URL"], props["KUCOIN_KEY"], props["KUCOIN_SECRET"], props["KUCOIN_ETH_FEE"])
	bitz.Initialize(props["BITZ_URL"], props["BITZ_KEY"], props["BITZ_SECRET"], props["BITZ_TRADEPW"], props["BITZ_ETH_FEE"])
	okex.Initialize(props["OKEX_URL"], props["OKEX_KEY"], props["OKEX_SECRET"], props["OKEX_TRADEPW"], props["OKEX_ETH_FEE"])

	// initialize discord bot
	discord.Initialize(props["DISCORD_AUTH_TOKEN"], props["DISCORD_BOT_ID"], props["DISCORD_CHANNEL_ID"])

}

func main() {

	// main arbitrage flow
	arbitrage := gocron.NewScheduler()
	arbitrage.Every(1).Minutes().Do(run)
	<-arbitrage.Start()

	// once a day update total balance
	// and post summary to discord
	daily := gocron.NewScheduler()
	daily.Every(1).Day().At("20:00").Do(daily)
	<-daily.Start()

	// every 3 days, look at all tokens
	// listed on supported exchanges
	analyze := gocron.NewScheduler()
	analyze.Every(3).Day().Do(analyze)
	<-analyze.Start()

}

func run() {

	//-----------------------------------//
	// check for flags that kill bot
	// for safety reasons, ie bad transaction
	//-----------------------------------//
	check_flags(mongo.Get_flags())

	//-----------------------------------//
	// combine personal and discord user tokens
	// this should be safe, because trading tokens
	// depends on minimum of 2 available balances
	// on distinct exchanges
	//-----------------------------------//
	discord_tokens := mongo.Get_discorders_distinct_tokens()
	combined_tokens := combine_personal_and_discord_tokens(tokens, discord_tokens)

	//-----------------------------------//
	// get prices from all exchanges
	//-----------------------------------//
	exchange_prices["binance"] = binance.Get_price(combined_tokens)
	exchange_prices["kucoin"] = kucoin.Get_price(combined_tokens)
	// exchange_prices["bitz"] = bitz.Get_price(combined_tokens)
	exchange_prices["okex"] = okex.Get_price(combined_tokens)

	//-----------------------------------//
	// get balances from all exchanges
	//-----------------------------------//
	exchange_balances["binance"] = binance.Get_balances(combined_tokens)
	exchange_balances["kucoin"] = kucoin.Get_balances(combined_tokens)
	// exchange_balances["bitz"] = bitz.Get_balances(combined_tokens)
	exchange_balances["okex"] = okex.Get_balances(combined_tokens)

	//-----------------------------------//
	// exclude tokens that have available balance
	// on only 1 exchange, need 2 min for arbitrage
	//-----------------------------------//
	exclude := exclude_tokens(exchange_balances)

	//-----------------------------------//
	// start new transactions
	//-----------------------------------//
	compare_prices(exchange_prices, exclude)

	//-----------------------------------//
	// get incomplete transactions
	//-----------------------------------//
	resume_transactions(mongo.Get_incomplete_transactions())

	//-----------------------------------//
	// save comparisons
	//-----------------------------------//
	mongo.Save_comparisons(comparisons)

	//-----------------------------------//
	// save prices from all exchanges
	//-----------------------------------//
	mongo.Save_prices(exchange_prices)

}

func analyze() {

	unique := make(map[string][]string)

	listed_tokens["binance"] = binance.Get_listed_tokens()
	listed_tokens["kucoin"] = kucoin.Get_listed_tokens()
	listed_tokens["bitz"] = bitz.Get_listed_tokens()

	// OKEX has no convenient way of getting a list of all listed tokens
	// thus, we have to pass everything we collected thus far
	// and look up each token one by one
	combo := utils.Merge_uniques(listed_tokens["binance"], listed_tokens["kucoin"], listed_tokens["bitz"])
	listed_tokens["okex"] = okex.Get_listed_tokens(combo)

	// format for storage
	for exchange, tokens := range listed_tokens {

		for _, token := range tokens {

			unique[token] = append(unique[token], exchange)

		}

	}

	mongo.Update_listed_tokens(unique)

}

func combine_personal_and_discord_tokens(tokens map[string]bool, discord_tokens []string) map[string]bool {

	var combined_tokens = make(map[string]bool)

	// copy the original tokens to a new array
	for k, v := range tokens {
		combined_tokens[k] = v
	}

	// add discord tokens
	for _, d := range discord_tokens {

		if !combined_tokens[d] {

			combined_tokens[d] = true

		}

	}

	return combined_tokens

}

func exclude_tokens(exchange_balances map[string]map[string]float64) map[string]bool {

	var exclude = make(map[string]bool)
	var count = make(map[string]int)

	for _, tokens := range exchange_balances {
		for token, balance := range tokens {

			trade_amount := float64(trade_quantity[token])

			// here we're adding +1 for every exchange with available balance
			// this way we can count exchanges and make sure we have at least 2
			// since arbitrage only works with 2+ exchanges
			if trade_amount > 0 && balance >= trade_amount {
				count[token]++
			} else {
				count[token] = 0
			}
		}
	}

	for token, score := range count {
		if score < 2 {
			exclude[token] = true
		}
	}

	return exclude

}

// finds transactions that are in progress
// checks on their current status and moves things along
func resume_transactions(transactions []utils.Transaction) {

	for _, t := range transactions {

		switch t.Status {

		case utils.SellPlaced:
			check_if_sold(t.ID.Hex(), t.Token, t.Sell_exchange, t.Sell_tx_id)

		case utils.SellCompleted:
			buy_exchange := comparisons[t.Token].Min_exchange
			destination := props[strings.ToUpper(buy_exchange)+"_ETH_ADDRESS"]
			buy_price := comparisons[t.Token].Min_price

			// time has passed since the sale was first placed
			// it has been fulfilled, but the prices may have changed
			// enough for us to lose the % difference required to profit
			comparison := comparisons[t.Token]

			// check if difference is over the thershold
			// if so, trigger the sell
			if comparison.Difference >= percent_threshold {
				start_transfer(t.ID.Hex(), "ETH", t.Sell_exchange, buy_exchange, destination, t.Sell_cost, buy_price)
			}

		case utils.TransferStarted:
			check_if_transferred(t.ID.Hex(), t.Buy_exchange, t.Sell_cost)

		case utils.TransferCompleted:

			token := strings.ToUpper(t.Token + "-ETH")
			buy_price := exchange_prices[t.Buy_exchange][token]
			quantity := t.Sell_cost / buy_price

			// if we're about to place a buy order
			// for a less than profitable amount of tokens
			// throw error and kill bot
			// 4 is an arbitrary number for now, should be revisited
			if quantity < float64(trade_quantity[t.Token]+4) {
				throw_flag()
			}

			place_buy_order(t.ID.Hex(), t.Token, t.Buy_exchange, buy_price, quantity)

		case utils.BuyPlaced:
			check_if_bought(t.ID.Hex(), t.Token, t.Buy_exchange, t.Sell_exchange, t.Buy_tx_id)

		case utils.BuyCompleted:
			exchange := strings.ToUpper(t.Sell_exchange)
			destination := props[exchange+"_ETH_ADDRESS"]
			// the fee is arbitrary for now but needs to be smarter
			amount := float64(trade_quantity[t.Token] + 4)

			reset(t.Token, t.Buy_exchange, destination, t.ID.Hex(), amount)

		default:
			panic("Invalid transaction status.")

		}

	}

}

func check_if_sold(row_id, token, sell_exchange, sell_tx_id string) {

	amount := 0.0
	sold := false

	switch sell_exchange {

	case "binance":
		amount, sold = binance.Check_if_sold(token, sell_tx_id)

	case "kucoin":
		amount, sold = kucoin.Check_if_sold(token, sell_tx_id)

	case "bitz":
		amount, sold = bitz.Check_if_sold(token, sell_tx_id)

	case "okex":
		amount, sold = okex.Check_if_sold(token, sell_tx_id)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if sold {
		mongo.Sell_order_completed(row_id, sell_exchange, amount)
	}

}

func start_transfer(row_id, token, sell_exchange, buy_exchange, destination string, amount, buy_price float64) {

	tx_id := ""
	started := false

	switch sell_exchange {

	case "binance":
		tx_id, started = binance.Start_transfer(token, destination, amount)

	case "kucoin":
		tx_id, started = kucoin.Start_transfer(token, destination, amount)

	case "bitz":
		tx_id, started = bitz.Start_transfer(token, destination, amount)

	case "okex":
		tx_id, started = okex.Start_transfer(token, destination, amount)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if started {
		mongo.Transfer_started(row_id, tx_id, buy_exchange, buy_price)
	}

}

func check_if_transferred(row_id, buy_exchange string, sell_cost float64) {

	transferred := false
	sell_cost = utils.ToFixed(sell_cost-fees[buy_exchange], 4)

	switch buy_exchange {

	case "binance":
		transferred = binance.Check_if_transferred(sell_cost)

	case "kucoin":
		transferred = kucoin.Check_if_transferred(sell_cost)

	case "bitz":
		transferred = bitz.Check_if_transferred(sell_cost)

	case "okex":
		transferred = okex.Check_if_transferred(sell_cost)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if transferred {
		mongo.Transfer_completed(row_id)
	}

}

func place_buy_order(row_id, token, buy_exchange string, buy_price, quantity float64) {

	tx_id := ""
	placed := false

	switch buy_exchange {

	case "binance":
		tx_id, placed = binance.Place_buy_order(token, quantity, buy_price)

	case "kucoin":
		tx_id, placed = kucoin.Place_buy_order(token, quantity, buy_price)

	case "bitz":
		tx_id, placed = bitz.Place_buy_order(token, quantity, buy_price)

	case "okex":
		tx_id, placed = okex.Place_buy_order(token, quantity, buy_price)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if placed {
		mongo.Buy_order_placed(row_id, tx_id, quantity, buy_price)
	}

}

func check_if_bought(row_id, token, buy_exchange, sell_exchange, buy_tx_id string) {

	bought := false

	switch buy_exchange {

	case "binance":
		bought = binance.Check_if_bought(token, buy_tx_id)

	case "kucoin":
		bought = kucoin.Check_if_bought(token, buy_tx_id)

	case "bitz":
		bought = bitz.Check_if_bought(token, buy_tx_id)

	case "okex":
		bought = okex.Check_if_bought(token, buy_tx_id)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if bought {
		mongo.Buy_order_completed(row_id)
	}

}

// starting point, loops over all tokens
// uses find_min_max_exchanges() on each token
// if there is sufficient price gap, begins a transaction with sell()
func compare_prices(exchange_prices map[string]map[string]float64, exclude map[string]bool) {

	messages := make(map[string]string, len(tokens)-len(exclude))

	for token := range tokens {

		prices := filter_prices(token, exchange_prices)

		comparison := find_min_max_exchanges(prices)
		comparisons[token] = comparison

		fmt.Println(token, comparison, "Difference:", comparison.Difference, "%")

		// comparisons are used for personal needs and discord subscribers
		// however, here we can skip the rest of the process
		//  if a token us not used for personal trading
		if exclude[token] {
			continue
		}

		// check if difference is over the thershold
		// if so, trigger the sell
		if comparison.Difference >= percent_threshold {

			place_sell_order(token, comparison.Max_exchange, comparison.Max_price)

		}

		// separate check for discord notifications
		if comparison.Difference >= discord_percent_threshold {

			string_diff := strconv.FormatFloat(comparison.Difference, 'f', 0, 64)
			message := token + " " + string_diff + "% difference between "
			message += comparison.Min_exchange + "(min) and " + comparison.Max_exchange + "(max)" + " on ETH pair"
			messages[token] = message

		}

	}

	// this is a pesonal method, notify me
	// discord.Send_messages(messages)

	// notify discord subscribers
	discord.Notify_discorders(comparisons)

}

// because not all tokens are available on all exchanges
// when we prepare token prices for comparison
// we need to make sure that we have an actual price, more than 0
func filter_prices(token string, exchange_prices map[string]map[string]float64) map[string]float64 {

	prices := make(map[string]float64)

	pair := token + "-ETH"

	for exchange, tokens := range exchange_prices {

		if tokens[pair] > 0 {
			prices[exchange] = tokens[pair]
		}

	}

	return prices

}

// accepts a list of prices for 1 token
// fints the minimum and maximum price
// as well as which exchange they're on
func find_min_max_exchanges(prices map[string]float64) utils.Comparison {

	c := utils.Comparison{}

	for exchange, price := range prices {

		// starting point
		if c.Min_price == 0 && c.Max_price == 0 {
			c.Min_price = price
			c.Max_price = price
			c.Min_exchange = exchange
			c.Max_exchange = exchange

			continue
		}

		if price < c.Min_price {
			c.Min_price = price
			c.Min_exchange = exchange
		}

		if price > c.Max_price {
			c.Max_price = price
			c.Max_exchange = exchange
		}

	}

	// calculte percentage difference
	difference := (1 - c.Min_price/c.Max_price) * 100
	c.Difference = utils.ToFixed(difference, 2)
	c.Timestamp = time.Now()

	return c

}

// start transaction, selling high
func place_sell_order(token, exchange string, price float64) {

	sell_placed := false
	transaction_id := ""

	switch exchange {

	case "binance":
		transaction_id, sell_placed = binance.Place_sell_order(token, trade_quantity[token], price)

	case "kucoin":
		transaction_id, sell_placed = kucoin.Place_sell_order(token, trade_quantity[token], price)

	case "bitz":
		transaction_id, sell_placed = bitz.Place_sell_order(token, trade_quantity[token], price)

	case "okex":
		transaction_id, sell_placed = okex.Place_sell_order(token, trade_quantity[token], price)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if sell_placed {
		mongo.Place_sell_order(token, exchange, transaction_id, price)
	}

}

// final step in arbitrage process, send tokens back to origin
func reset(token, buy_exchange, destination, row_id string, amount float64) {

	is_reset := false
	transaction_id := ""

	switch buy_exchange {

	case "binance":
		transaction_id, is_reset = binance.Start_transfer(token, destination, amount)

	case "kucoin":
		transaction_id, is_reset = kucoin.Start_transfer(token, destination, amount)

	case "bitz":
		transaction_id, is_reset = bitz.Start_transfer(token, destination, amount)

	case "okex":
		transaction_id, is_reset = okex.Start_transfer(token, destination, amount)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if is_reset {
		mongo.Token_reset_completed(row_id, transaction_id)
	}

}

func check_flags(flags []utils.Flag) {

	if len(flags) > 0 {
		panic("Flag detected, bot execution stalled.")
	}

}

func throw_flag() {

	mongo.Flag("Buying less than profitable quantity.")
	panic("Threw flag, killing bot.")

}

func daily() {

	var message string
	// save daily balance, for time scale tracking
	mongo.Save_balances(exchange_balances)

	// calculations and comparison of today vs previous day
	from_date := time.Now().AddDate(0, 0, -2)
	to_date := time.Now()
	prev_day_balances := mongo.Get_balances(from_date, to_date)
	from_date = time.Now().AddDate(0, 0, -1)
	to_date = time.Now()
	todays_transactions := mongo.Get_transactions(from_date, to_date)

	// composit the messages of daily summary
	message += "------------------------start\n"
	message += "DAILY SUMMARY\n"
	message += "-----------------------------\n"

	for _, b := range prev_day_balances {

		message += b.Exchange + "... coming soon \n"

	}

	message += "-----------------------trades\n"

	for _, t := range todays_transactions {

		sell_quantity := fmt.Sprintf("%.2f", t.Sell_quantity)
		buy_quantity := fmt.Sprintf("%.2f", t.Buy_quantity)
		message += t.Token + " - sold: " + sell_quantity + ", bought: " + buy_quantity + "\n"

	}

	message += "--------------------------end\n"

	// send daily summary to discord
	discord.Send_daily_summary(message)

}
