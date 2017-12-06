package main

import (
	//"fmt"
	"io/ioutil"
	"strings"
	//"strconv"

	"./exchanges"
)

// holds environment variables
// such as api endpoints and their keys
var props = make(map[string]string)

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
		split := strings.Split(l,"=")
		props[split[0]] = split[1]
	}

	// initialize exchange packages
	binance.Initialize(props["BINANCE_URL"], props["BINANCE_Key"], props["BINANCE_SECRET"])

}

func main() {

	binance.Test()

}