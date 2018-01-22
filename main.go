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

// comparisons are stored per token
// Ex: ["NULS"] = {"Min_price" : 0.04, ...}
var comparisons = make(map[string]Comparison)

// currently limited to holding ETH fee
// charged by each exchange upon withdrawal
var fees = make(map[string]float64)

// percentage threshold is the difference between min and max price
// required for us to profit from running arbitrage
var percent_threshold float64

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

			} else if strings.HasSuffix(split[0], "_FEE") {

				exchange := strings.Split(split[0], "_")[0]
				exchange = strings.ToLower(exchange)
				fee, err := strconv.ParseFloat(split[1], 64)
				check(err)

				fees[exchange] = fee

			} else {

				props[split[0]] = split[1]

			}
		}
	}

	// initialize any secondary variables
	// that need to be globablly available
	percent_threshold, err = strconv.ParseFloat(props["PERCENT_THRESHOLD"], 64)
	check(err)

	// initialize database connection
	mongo.Initialize(props["HOST"], props["DATABASE"], props["USERNAME"], props["PASSWORD"])

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"], props["BINANCE_ETH_FEE"])
	kucoin.Initialize(props["KUCOIN_URL"], props["KUCOIN_KEY"], props["KUCOIN_SECRET"], props["KUCOIN_ETH_FEE"])
	bitz.Initialize(props["BITZ_URL"], props["BITZ_KEY"], props["BITZ_SECRET"], props["BITZ_TRADEPW"], props["BITZ_ETH_FEE"])
	okex.Initialize(props["OKEX_URL"], props["OKEX_KEY"], props["OKEX_SECRET"], props["OKEX_TRADEPW"], props["OKEX_ETH_FEE"])

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
	exclude := exclude_tokens(binance_balances, kucoin_balances, bitz_balances, okex_balances)

	//-----------------------------------//
	// start new transactions
	//-----------------------------------//
	compare_prices(binance_prices, kucoin_prices, bitz_prices, okex_prices, exclude)

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

func exclude_tokens(binance, kucoin, bitz, okex map[string]float64) map[string]bool {

	var exclude = make(map[string]bool)

	for token := range tokens {

		quant := float64(trade_quantity[token])

		binance_sufficient := utils.Ternary(1, 0, binance[token] >= quant)
		kucoin_sufficient := utils.Ternary(1, 0, kucoin[token] >= quant)
		bitz_sufficient := utils.Ternary(1, 0, bitz[token] >= quant)
		okex_sufficient := utils.Ternary(1, 0, okex[token] >= quant)
		sufficient_balance := binance_sufficient + kucoin_sufficient + bitz_sufficient + okex_sufficient

		if sufficient_balance < 2 {
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
			exchange := strings.ToUpper(comparisons[t.Token].Min_exchange)
			destination := props[exchange+"_ETH_ADDRESS"]
			buy_price := comparisons[t.Token].Min_price

			// time has passed since the sale was first placed
			// it has been fulfilled, but the prices may have changed
			// enough for us to lose the % difference required to profit
			comparison := comparisons[t.Token]

			// calculte percentage difference
			difference := (1 - comparison.Min_price/comparison.Max_price) * 100
			difference = utils.ToFixed(difference, 0)

			// check if difference is over the thershold
			// if so, trigger the sell
			if difference >= percent_threshold {
				start_transfer(t.ID.Hex(), "ETH", t.Sell_exchange, destination, t.Sell_cost, buy_price)
			}

		case utils.TransferStarted:
			check_if_transferred(t.ID.Hex(), t.Buy_exchange, t.Sell_cost)

		case utils.TransferCompleted:
			quantity := t.Sell_cost / t.Buy_price
			buy_price := comparisons[t.Token].Min_price
			place_buy_order(t.ID.Hex(), t.Token, t.Buy_exchange, buy_price, quantity)

		case utils.BuyPlaced:
			check_if_bought(t.ID.Hex(), t.Token, t.Buy_exchange, t.Buy_tx_id)

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

func start_transfer(row_id, token, sell_exchange, destination string, amount, buy_price float64) {

	tx_id := ""
	started := false

	switch sell_exchange {

	case "binance":
		tx_id, started = binance.Start_transfer(token, destination, amount)

	case "kucoin":
		started = kucoin.Start_transfer(token, destination, amount)

	case "bitz":
		tx_id, started = bitz.Start_transfer(token, destination, amount)

	case "okex":
		tx_id, started = okex.Start_transfer(token, destination, amount)

	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if started {
		fmt.Println(tx_id)
		mongo.Transfer_started(row_id, tx_id, buy_price)
	}

}

func check_if_transferred(row_id, buy_exchange string, sell_cost float64) {

	transferred := false
	sell_cost -= fees[buy_exchange]

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
		fmt.Println(tx_id)
		mongo.Buy_order_placed(row_id, tx_id, quantity, buy_price)
	}

}

func check_if_bought(row_id, token, buy_exchange, buy_tx_id string) {

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
func compare_prices(binance, kucoin, bitz, okex map[string]float64, exclude map[string]bool) {

	for token := range tokens {

		if exclude[token] {
			continue
		}

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
		difference := (1 - comparison.Min_price/comparison.Max_price) * 100
		difference = utils.ToFixed(difference, 0)
		fmt.Println(comparison, "Difference:", difference, "%")
		// check if difference is over the thershold
		// if so, trigger the sell
		if difference >= percent_threshold {
			// place_sell_order(token, comparison.Max_exchange, comparison.Max_price)
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
		mongo.Place_sell_order(token, exchange, transaction_id, price)
	}

}
