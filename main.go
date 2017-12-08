package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	//"strconv"

	"./exchanges/binance"
)

// holds environment variables
// such as api endpoints and their keys
var props = make(map[string]string)

// golang doesn't like detecting value existance within an array
// giving every key a boolean makes for easy checks of existance
var tokens = map[string]bool{
	"NULSETH": true,
	"LINKETH": true,
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

	for _, l := range lines {
		split := strings.Split(l, "=")
		props[split[0]] = split[1]
	}

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_KEY"], props["BINANCE_SECRET"])

}

func main() {

	fmt.Println(binance.Get_price(tokens))

}
