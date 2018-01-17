package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

	// individual exchange packages
	"./exchanges/binance"
	"./exchanges/bitz"
	"./exchanges/kucoin"

	// database package
	"./db/mongo"
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

}

func main() {

	// pull prices from all exchanges
	// binance_prices := binance.Get_price(tokens)
	// kucoin_prices := kucoin.Get_price(tokens)
	// bitz_prices := bitz.Get_price(tokens)
	// bitz.Sell("NULS", trade_quantity["NULS"], 0.006)

	// binance_balances := binance.Get_balances(tokens)
	// kucoin_balances := kucoin.Get_balances(tokens)
	// fmt.Println(binance_balances, kucoin_balances)

	// exclude tokens that are already being transacted or transfered
	// exclude := check_balances(binance_balances)

	// compare_prices(binance_prices, kucoin_prices, bitz_prices, exclude)

	// mongo.Save_prices(binance_prices)
	// mongo.Save_prices(kucoin_prices)
	// mongo.Save_prices(bitz_prices)

}

func check_balances(binance map[string]string) map[string]bool {

	var exclude = make(map[string]bool)

	return exclude

}

func compare_prices(binance, kucoin, bitz map[string]float64, exclude map[string]bool) {

	for token := range tokens {

		pair := token + "-ETH"

		prices := map[string]float64{
			"binance": binance[pair],
			"kucoin":  kucoin[pair],
			"bitz":    bitz[pair],
		}

		min_price, max_price, min_exchange, max_exchange := find_min_max_exchanges(prices)
		fmt.Println(min_price, max_price, min_exchange, max_exchange)

		// calculte percentage difference
		difference := (1 - max_price/min_price) * 100

		// check if difference is over the thershold
		// if so, trigger the sell
		percent_threshold, err := strconv.ParseFloat(props["PERCENT_THRESHOLD"], 64)
		check(err)

		if difference >= percent_threshold {

			sell(token, max_exchange, max_price)

		}

	}

}

func find_min_max_exchanges(prices map[string]float64) (float64, float64, string, string) {

	min_price := 0.0
	max_price := 0.0
	min_exchange := ""
	max_exchange := ""

	for exchange, price := range prices {

		// starting point
		if min_price == 0 && max_price == 0 {
			min_price = price
			max_price = price
			min_exchange = exchange
			max_exchange = exchange

			continue
		}

		if price < min_price {
			min_price = price
			min_exchange = exchange
		}

		if price > max_price {
			max_price = price
			max_exchange = exchange
		}

	}

	return min_price, max_price, min_exchange, max_exchange

}

// start transaction, selling high
func sell(token, exchange string, price float64) {

	sell_placed := false
	transaction_id := ""

	switch exchange {

	case "binance":
		transaction_id, sell_placed = binance.Sell(token, trade_quantity[token], price)
	case "kucoin":
		transaction_id, sell_placed = kucoin.Sell(token, trade_quantity[token], price)
	case "bitz":
		transaction_id, sell_placed = bitz.Sell(token, trade_quantity[token], price)
	default:
		panic("Exchange selection not provided or doesn't match available choices.")

	}

	if sell_placed {
		mongo.Create_transaction(token, exchange, transaction_id, price)
	}

}

// check transaction progress
// if sale is complete, transfer ETH to exchange with lowest price

// finalize transaction, restore balances on all exchanges

// occasionally, send our profit coins to trezor address

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
