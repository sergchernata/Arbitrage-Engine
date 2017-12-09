package binance

import (
	"encoding/json"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"
	//"reflect"
	"strings"
)

var api_url, api_key, api_secret string

type Prices []struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func Initialize(url string, key string, secret string) {

	api_url = url
	api_key = key
	api_secret = secret

}

func Get_price(tokens map[string]bool) map[string]string {

	var endpoint = "/api/v3/ticker/price"
	var data *Prices
	var prices = make(map[string]string)

	// perform api call
	data = execute(api_url + endpoint)

	// parse data and format for return
	for _, v := range *data {

		// binance formats pairs as "LINKETH"
		// we're going to instead use kucoin's format "LINK-ETH"
		symbol := v.Symbol
		price := v.Price
		is_eth_pair := strings.HasSuffix(symbol, "ETH")
		token := strings.TrimSuffix(symbol, "ETH")

		if is_eth_pair && tokens[token] {
			prices[token+"-ETH"] = price
		}
	}

	return prices
}

func execute(url string) *Prices {

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

	var data = new(Prices)

	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return data

}
