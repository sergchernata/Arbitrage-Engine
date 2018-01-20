package kucoin

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
	"strings"
	"time"
)

var api_url, api_key, api_secret string
var api_eth_fee float64

type Transfer_request struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Data    string `json:"data"`
}

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
	Symbol string `json:"coinType"`
	Amount string `json:"balanceStr,Number"`
}

type Prices struct {
	Prices  []Price `json:"data"`
	Success bool    `json:"success"`
}

type Price struct {
	Symbol string      `json:"symbol"`
	Price  json.Number `json:"lastDealPrice,Number"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Initialize(url, key, secret, eth_fee string) {

	fmt.Println("initializing kucoin package")

	api_url = url
	api_key = key
	api_secret = secret
	api_eth_fee, _ = strconv.ParseFloat(eth_fee, 64)

}

func Get_balances(tokens map[string]bool) map[string]float64 {

	var holdings = make(map[string]float64)
	var body []byte

	for token, _ := range tokens {

		var data = new(Holdings)
		var endpoint = "/v1/account/" + token + "/balance"
		var params = ""

		// perform api call
		body = execute("GET", api_url, endpoint, params, true)

		err := json.Unmarshal(body, &data)
		check(err)

		amount, err := strconv.ParseFloat(data.Holding.Amount, 64)
		check(err)

		holdings[data.Holding.Symbol] = amount

	}

	return holdings
}

func Get_price(tokens map[string]bool) map[string]float64 {

	var params = ""
	var endpoint = "/v1/open/tick"
	var data = new(Prices)
	var prices = make(map[string]float64)
	var body []byte

	// perform api call
	body = execute("GET", api_url, endpoint, params, false)

	err := json.Unmarshal(body, &data)
	check(err)

	//parse data and format for return
	for _, v := range data.Prices {

		// kucoin formats pairs as "LINK-ETH"
		// this will be the format we convert others to
		symbol := v.Symbol
		is_eth_pair := strings.HasSuffix(symbol, "-ETH")
		token := strings.TrimSuffix(symbol, "-ETH")
		price, err := strconv.ParseFloat(string(v.Price), 64)
		check(err)

		if is_eth_pair && tokens[token] {
			prices[token+"-ETH"] = price
		}
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

func Check_if_sold(token, sell_tx_id string) (float64, bool) {

	var amount = 0.0
	token += "-ETH"
	var params = fmt.Sprintf("limit=%d&orderOid=%s&page=%d&symbol=%s&type=%s", 5, sell_tx_id, 1, token, "SELL")
	var endpoint = "/v1/order/detail"
	var order = new(Order)
	var body []byte

	// perform api call
	body = execute("GET", api_url, endpoint, params, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Data.Id == "" {
		return amount, false
	}

	return amount, true

}

func Start_transfer(token, destination string, amount float64) bool {

	var params = fmt.Sprintf("address=%s&amount=%f&coin=%s", destination, amount, token)
	var endpoint = "/v1/account/" + token + "/withdraw/apply"
	var transfer = new(Transfer_request)
	var body []byte

	// perform api call
	body = execute("POST", api_url, endpoint, params, true)

	err := json.Unmarshal(body, &transfer)
	check(err)

	if transfer.Success {
		return true
	}

	return false

}

func Check_if_transferred(sell_cost float64) bool {

	return true

}

func Place_buy_order(token string, buy_cost float64) (string, bool) {

	return "", true

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

	req.Header.Set("User-Agent", "test")
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
