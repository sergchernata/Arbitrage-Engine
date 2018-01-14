package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	//"net/url"
	//"reflect"
	"strings"
)

var api_url, api_key, api_secret string

type Holdings struct {
	Holdings []Holding `json:"balances"`
	Success  bool      `json:"success"`
}

type Holding []struct {
	Symbol string `json:"asset"`
	Amount string `json:"free"`
}

type Prices []struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Initialize(url string, key string, secret string) {

	api_url = url
	api_key = key
	api_secret = secret

}

func Get_balances(tokens map[string]bool) map[string]string {

	var endpoint = "/api/v3/account"
	var holdings = make(map[string]string)
	var data = new(Holdings)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, true)
	fmt.Println(string(body))
	err := json.Unmarshal(body, &data)
	check(err)

	fmt.Println(data)

	return holdings

}

func Get_price(tokens map[string]bool) map[string]string {

	var endpoint = "/api/v3/ticker/price"
	var prices = make(map[string]string)
	var data = new(Prices)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, false)

	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

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

func Sell(token string) (transaction_id string, sell_placed bool) {

	transaction_id = ""
	sell_placed = false

	return transaction_id, sell_placed

}

func execute(method string, url string, auth bool) []byte {

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
	}

	req.Header.Set("User-Agent", "test")
	req.Header.Add("Accept", "application/json")

	if auth {

		req.Header.Add("X-MBX-APIKEY", api_key)

		q := req.URL.Query()

		timestamp := time.Now().Unix() * 1000
		q.Set("timestamp", fmt.Sprintf("%d", timestamp))

		mac := hmac.New(sha256.New, []byte(api_secret))
		_, err := mac.Write([]byte(q.Encode()))
		check(err)

		signature := hex.EncodeToString(mac.Sum(nil))
		req.URL.RawQuery = q.Encode() + "&signature=" + signature
	}

	client := &http.Client{}

	res, err := client.Do(req)
	check(err)

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	check(err)

	return body

}
