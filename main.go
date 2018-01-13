package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

	// individual exchange packages
	"./exchanges/binance"
	"./exchanges/kucoin"

	// database package
	"./db/mongo"
)

// holds environment variables
// such as api endpoints and their keys
var props = make(map[string]string)

// golang doesn't like detecting value existance within an array
// giving every key a boolean makes for easy checks of existance
var tokens = map[string]bool{
	"NULS": true,
	"LINK": true,
	"REQ":  true,
	"NEO":  true,
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {

	fmt.Println("initializing...")

	// process environment variables
	dat, err := ioutil.ReadFile(".env")
	check(err)

	lines := strings.Split(string(dat), "\n")

	for _, line := range lines {
		// check for blank and comment lines in .env config
		if line != "" && !strings.HasPrefix(line, "#") {
			split := strings.Split(line, "=")
			props[split[0]] = split[1]
		}
	}

	// initialize database connection
	mongo.Initialize(props["HOST"], props["DATABASE"], props["USERNAME"], props["PASSWORD"])

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"])
	kucoin.Initialize(props["KUCOIN_URL"], props["KUCOIN_KEY"], props["KUCOIN_SECRET"])

}

func main() {

	binance_prices := binance.Get_price(tokens)
	kucoin_prices := kucoin.Get_price(tokens)

	compare_prices(binance_prices, kucoin_prices)

	// mongo.Save_prices(binance_prices)
	// mongo.Save_prices(kucoin_prices)

}

func compare_prices(binance, kucoin map[string]string) {

	for token := range tokens {

		pair := token + "-ETH"

		binance_value, binance_ok := binance[pair]
		kucoin_value, kucoin_ok := kucoin[pair]

		if binance_ok && kucoin_ok {

			binance_float, err := strconv.ParseFloat(binance_value, 64)
			kucoin_float, err := strconv.ParseFloat(kucoin_value, 64)

			if err != nil {
				panic(err)
			}

			difference := (1 - binance_float/kucoin_float) * 100
			fmt.Println(pair, difference, "Binance: ", binance_float, "KuCoin: ", kucoin_float)

		}

	}

}

// start transaction, selling high

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
