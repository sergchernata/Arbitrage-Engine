package okex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	//"strings"
	"time"
)

var api_url, api_key, api_secret, api_tradepw string

type Order struct {
	Success bool `json:"success"`
	Data    struct {
		Id string `json:"orderOid"`
	} `json:"data"`
}

type Holdings struct {
	Holding Holding `json:"data"`
	Success bool    `json:"success"`
}

type Holding struct {
	Symbol string  `json:"coinType"`
	Amount float64 `json:"balanceStr,Number"`
}

type Prices struct {
	Data Price  `json:"ticker"`
	Date string `json:"date"`
}

type Price struct {
	High string `json:"high,Number"`
	Low  string `json:"low,Number"`
	Sell string `json:"sell,Number"`
	Buy  string `json:"buy,Number"`
	Last string `json:"last,Number"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Initialize(url, key, secret, tradepw string) {

	fmt.Println("initializing okex package")

	api_url = url
	api_key = key
	api_secret = secret
	api_tradepw = tradepw

}

func Get_balances(tokens map[string]bool) map[string]float64 {

	var holdings = make(map[string]float64)
	// var body []byte

	// for token, _ := range tokens {

	// 	var data = new(Holdings)
	// 	var endpoint = "/v1/account/" + token + "/balance"
	// 	var params = ""

	// 	// perform api call
	// 	body = execute("GET", api_url, endpoint, params, true)

	// 	err := json.Unmarshal(body, &data)
	// 	check(err)

	// 	holdings[data.Holding.Symbol] = data.Holding.Amount

	// }

	return holdings
}

func Get_price(tokens map[string]bool) map[string]float64 {

	var endpoint = "/ticker.do"
	var prices = make(map[string]float64)
	var body []byte

	// perform api call per token
	for token, _ := range tokens {

		var data = new(Prices)
		var params = fmt.Sprintf("symbol=%s", token+"_ETH")

		// perform api call
		body = execute("GET", api_url, endpoint, params, false)

		err := json.Unmarshal(body, &data)
		check(err)

		price, err := strconv.ParseFloat(data.Data.Last, 64)
		check(err)

		prices[token] = price

	}

	return prices
}

func Place_sell_order(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	token += "-ETH"
	var params = fmt.Sprintf("amount=%d&price=%f&symbol=%s&type=%s", quantity, price, token, "SELL")
	var endpoint = "/v1/order"
	var order = new(Order)
	var body []byte

	// perform api call
	body = execute("POST", api_url, endpoint, params, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Data.Id == "" {
		return "", false
	}

	return order.Data.Id, true

}

func Check_if_sold(token, sell_tx_id string) bool {

	return true

}

func Start_transfer(token, destination string) bool {

	return true

}

func Check_if_transferred(sell_cost float64) bool {

	return true

}

func Place_buy_order(token string, buy_cost float64) bool {

	return true

}

func Check_if_bought(token, buy_tx_id string) bool {

	return true

}

func Withdraw(token, amount, address string) (transaction_id string, sell_placed bool) {

	var params = fmt.Sprintf("address=%s&amount=%s", address, amount)
	var endpoint = "/v1/account/" + token + "/withdraw/apply"
	var order = new(Order)
	var body []byte

	// perform api call
	body = execute("POST", api_url, endpoint, params, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Data.Id == "" {
		return "", false
	}

	return order.Data.Id, true

}

func execute(method string, url string, endpoint string, params string, auth bool) []byte {

	req, err := http.NewRequest(method, url+endpoint+"?"+params, nil)
	check(err)

	req.Header.Add("Accept", "application/json")

	if auth {

		timestamp := strconv.Itoa(int(time.Now().Unix() * 1000))

		//splice string for signing
		strForSign := endpoint + "/" + timestamp + "/" + params

		//Make a base64 encoding of the completed string
		signatureStr := base64.StdEncoding.EncodeToString([]byte(strForSign))

		mac := hmac.New(sha256.New, []byte(api_secret))
		_, err := mac.Write([]byte(signatureStr))
		check(err)

		signature := hex.EncodeToString(mac.Sum(nil))

		req.Header.Add("KC-API-KEY", api_key)
		req.Header.Add("KC-API-NONCE", timestamp)
		req.Header.Add("KC-API-SIGNATURE", signature)

	}

	client := &http.Client{}

	res, err := client.Do(req)
	check(err)

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	check(err)

	return body

}
