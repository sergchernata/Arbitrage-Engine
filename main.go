package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	// individual exchange packages
	"./exchanges/binance"
	"./exchanges/bitz"
	"./exchanges/kucoin"
	"./exchanges/okex"

	// database package
	"./db/mongo"

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

// each key is a token symbol, matching the array of tokens above
// each value is the number of tokens to be sold at once per trade
var trade_quantity = make(map[string]int)

var comparisons = make(map[string]Comparison)

type Comparison struct {
	Min_price    float64
	Max_price    float64
	Min_exchange string
	Max_exchange string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {

	fmt.Println("initializing main package")

	// process environment variables
	dat, err := ioutil.ReadFile(".env")
	check(err)

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

					// tokens with trade quantity of 0 are to be ignored
					if quantity > 0 {
						tokens[temp[0]] = true
						trade_quantity[temp[0]], _ = strconv.Atoi(temp[1])
					}
				}

			} else {

				props[split[0]] = split[1]

			}
		}
	}

	// initialize database connection
	mongo.Initialize(props["HOST"], props["DATABASE"], props["USERNAME"], props["PASSWORD"])

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"])
	kucoin.Initialize(props["KUCOIN_URL"], props["KUCOIN_KEY"], props["KUCOIN_SECRET"])
	bitz.Initialize(props["BITZ_URL"], props["BITZ_KEY"], props["BITZ_SECRET"], props["BITZ_TRADEPW"])
	okex.Initialize(props["OKEX_URL"], props["OKEX_KEY"], props["OKEX_SECRET"], props["OKEX_TRADEPW"])

}

func main() {

	//-----------------------------------//
	// get prices from all exchanges
	//-----------------------------------//
	binance_prices := binance.Get_price(tokens)
	kucoin_prices := kucoin.Get_price(tokens)
	bitz_prices := bitz.Get_price(tokens)
	okex_prices := okex.Get_price(tokens)

	//-----------------------------------//
	// get balances from all exchanges
	//-----------------------------------//
	binance_balances := binance.Get_balances(tokens)
	kucoin_balances := kucoin.Get_balances(tokens)
	bitz_balances := bitz.Get_balances(tokens)
	okex_balances := okex.Get_balances(tokens)

	//-----------------------------------//
	// exclude tokens that have available balance
	// on only 1 exchange, need 2 min for arbitrage
	//-----------------------------------//
	exclude := exclude_tokens(binance_balances, kucoin_balances, bitz_balances, okex_balances, tokens)

	//-----------------------------------//
	// start new transactions
	//-----------------------------------//
	compare_prices(binance_prices, kucoin_prices, bitz_prices, okex_prices, exclude)
	fmt.Println(okex.Check_if_sold("NULS", "eciwn8h4f"))

	//-----------------------------------//
	// get incomplete transactions
	//-----------------------------------//
	resume_transactions(mongo.Get_incomplete_transactions())

	//-----------------------------------//
	// save prices from all exchanges
	//-----------------------------------//
	// mongo.Save_prices(binance_prices)
	// mongo.Save_prices(kucoin_prices)
	// mongo.Save_prices(bitz_prices)

}

func exclude_tokens(binance, kucoin, bitz, okex map[string]float64, tokens map[string]bool) map[string]bool {

	var exclude = make(map[string]bool)

	return exclude

}

// finds transactions that are in progress
// checks on their current status and moves things along
func resume_transactions(transactions []utils.Transaction) {

	for _, t := range transactions {

		switch t.Status {

		case utils.SellPlaced:
			check_if_sold(t.Token, t.Sell_exchange, t.Sell_tx_id)

		case utils.SellCompleted:
			exchange := strings.ToUpper(comparisons[t.Token].Min_exchange)
			destination := props[exchange+"_ETH_ADDRESS"]
			start_transfer(t.Token, t.Sell_exchange, destination, t.Sell_cost)

		case utils.TransferStarted:
			check_if_transferred(t.Sell_cost, t.Buy_exchange)

		case utils.TransferCompleted:
			place_buy_order(t.Token, t.Buy_exchange, t.Sell_cost)

		case utils.BuyPlaced:
			check_if_bought(t.Token, t.Buy_exchange, t.Buy_tx_id)

		default:
			panic("Invalid transaction status.")

		}

	}

}

func check_if_sold(token, sell_exchange, sell_tx_id string) {

	sold := false

	switch sell_exchange {

	case "binance":
		sold = binance.Check_if_sold(token, sell_tx_id)

	case "kucoin":
		sold = kucoin.Check_if_sold(token, sell_tx_id)

	case "bitz":
		sold = bitz.Check_if_sold(token, sell_tx_id)

	case "okex":
		sold = okex.Check_if_sold(token, sell_tx_id)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if sold {

	}

}

func start_transfer(token, sell_exchange, destination string, amount float64) {

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
		fmt.Println(tx_id)
	}

}

func check_if_transferred(sell_cost float64, buy_exchange string) {

	transferred := false

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

	}

}

func place_buy_order(token, buy_exchange string, buy_cost float64) {

	tx_id := ""
	placed := false

	switch buy_exchange {

	case "binance":
		tx_id, placed = binance.Place_buy_order(token, buy_cost)

	case "kucoin":
		tx_id, placed = kucoin.Place_buy_order(token, buy_cost)

	case "bitz":
		tx_id, placed = bitz.Place_buy_order(token, buy_cost)

	case "okex":
		tx_id, placed = okex.Place_buy_order(token, buy_cost)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if placed {
		fmt.Println(tx_id)
	}

}

func check_if_bought(token, buy_exchange, buy_tx_id string) {

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

	}

}

// starting point, loops over all tokens
// uses find_min_max_exchanges() on each token
// if there is sufficient price gap, begins a transaction with sell()
func compare_prices(binance, kucoin, bitz, okex map[string]float64, exclude map[string]bool) {

	for token := range tokens {

		pair := token + "-ETH"

		prices := map[string]float64{
			"binance": binance[pair],
			"kucoin":  kucoin[pair],
			// "bitz":    bitz[pair], their api is being reworked
			"okex": okex[pair],
		}

		comparison := find_min_max_exchanges(prices)
		comparisons[token] = comparison

		// calculte percentage difference
		difference := (1 - comparison.Max_price/comparison.Min_price) * 100

		// check if difference is over the thershold
		// if so, trigger the sell
		percent_threshold, err := strconv.ParseFloat(props["PERCENT_THRESHOLD"], 64)
		check(err)

		if difference >= percent_threshold {

			// place_sell_order(token, max_exchange, max_price)

		}

	}

}

// accepts a list of prices for 1 token
// fints the minimum and maximum price
// as well as which exchange they're on
func find_min_max_exchanges(prices map[string]float64) Comparison {

	c := Comparison{}

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
		mongo.Create_transaction(token, exchange, transaction_id, price)
	}

}

// TODO:

// check transaction progress
// if sale is complete, transfer ETH to exchange with lowest price

// finalize transaction, restore balances on all exchanges

// occasionally, send our profit coins to trezor address
