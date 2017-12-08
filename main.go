package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	//"strconv"

	// individual exchange packages
	"./exchanges/binance"
	"./exchanges/kucoin"
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

	// process environment variables
	dat, err := ioutil.ReadFile(".env")
	check(err)

	lines := strings.Split(string(dat), "\n")

	for _, line := range lines {
		// check for blank lines in .env config
		if line != "" {
			split := strings.Split(line, "=")
			props[split[0]] = split[1]
		}
	}

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"])
	kucoin.Initialize(props["KUCOIN_URL"], props["KUCOIN_KEY"], props["KUCOIN_SECRET"])

}

func main() {

	fmt.Println(binance.Get_price(tokens))
	fmt.Println(kucoin.Get_price(tokens))

}
