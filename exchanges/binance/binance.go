package binance

import (
	"encoding/json"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"
	//"reflect"
)

var api_url, api_key, api_secret string

func Initialize(url string, key string, secret string) {

	api_url = url
	api_key = key
	api_secret = secret

}

func Get_price(tokens map[string]bool) map[string]string {

	var endpoint = "/api/v3/ticker/price"
	var data []interface{}
	var prices = make(map[string]string)

	data = execute(api_url + endpoint)

	for _, v := range data {
		row := v.(map[string]interface{})
		symbol := row["symbol"].(string)
		price := row["price"].(string)

		if tokens[symbol] {
			prices[symbol] = price
		}
	}

	return prices
}

func execute(url string) []interface{} {

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
	var data []interface{}

	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return data

}
