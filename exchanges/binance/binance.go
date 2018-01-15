package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var api_url, api_key, api_secret string

type Order struct {
	Id string `json:"orderId"`
}

type Holdings struct {
	Holdings []Holding `json:"balances"`
}

type Holding struct {
	Symbol string `json:"asset"`
	Amount string `json:"free,Number"`
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

	fmt.Println("initializing binance package")

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

	err := json.Unmarshal(body, &data)
	check(err)

	// remove tokens that we don't care about
	for _, v := range data.Holdings {

		symbol := v.Symbol
		amount := v.Amount

		if tokens[symbol] {
			holdings[symbol] = amount
		}
	}

	return holdings

}

func Get_price(tokens map[string]bool) map[string]string {

	var endpoint = "/api/v3/ticker/price"
	var prices = make(map[string]string)
	var data = new(Prices)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, false)

	err := json.Unmarshal(body, &data)
	check(err)

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

func Sell(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	token += "ETH"
	var endpoint = fmt.Sprintf("/api/v3/order?symbol=%s&side=%s&type=%s&quantity=%d&price=%f", token, "SELL", "MARKET", quantity, price)
	var data = new(Order)
	var body []byte

	// perform api call
	body = execute("POST", api_url+endpoint, true)

	err := json.Unmarshal(body, &data)
	check(err)

	if data.Id == "" {
		return "", false
	}

	return data.Id, true

}

func execute(method string, url string, auth bool) []byte {

	req, err := http.NewRequest(method, url, nil)
	check(err)

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
