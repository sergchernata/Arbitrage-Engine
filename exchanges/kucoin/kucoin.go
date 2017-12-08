package kucoin

import (
	"encoding/json"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	//"net/url"
	//"reflect"
)

var api_url, api_key, api_secret string

type Data struct {
	Prices  []Price `json:"data"`
	Success bool    `json:"success"`
}

type Price struct {
	Symbol string      `json:"symbol"`
	Price  json.Number `json:"lastDealPrice,Number"`
}

func Initialize(url string, key string, secret string) {

	api_url = url
	api_key = key
	api_secret = secret

}

func Get_price(tokens map[string]bool) map[string]string {

	var endpoint = "/v1/open/tick"
	var data *Data
	var prices = make(map[string]string)

	// perform api call
	data = execute(api_url + endpoint)

	//parse data and format for return
	for _, v := range data.Prices {

		symbol := v.Symbol
		price := string(v.Price)
		is_eth_pair := strings.HasSuffix(symbol, "-ETH")
		token := strings.TrimSuffix(symbol, "-ETH")

		if is_eth_pair && tokens[token] {
			prices[symbol] = price
		}
	}

	return prices
}

func execute(url string) *Data {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
	}

	req.Header.Set("User-Agent", "test")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
	}

	defer res.Body.Close()

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	// general interface
	// decode json without a predefined structure
	var data = new(Data)

	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return data

}
