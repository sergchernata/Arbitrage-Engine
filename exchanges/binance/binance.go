package binance

import (
	"crypto/hmac"
	"crypto/sha256"
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
	Msg     string `json:"msg"`
	Id      string `json:"id"`
}

type Deposits struct {
	List []struct {
		Amount  float64 `json:"amount,Number"`
		Asset   string  `json:"asset"`
		address string  `json:"address"`
		txId    string  `json:"txId"`
		status  string  `json:"status"`
	} `json:"depositList"`
	Success bool `json:"success"`
}

type Place_order struct {
	Id json.Number `json:"orderId"`
}

type Order struct {
	Symbol        string  `json:"symbol"`
	OrderId       float64 `json:"orderId.string"`
	ClientOrderId string  `json:"clientOrderId"`
	Price         float64 `json:"price,string"`
	OrigQty       float64 `json:"origQty,string"`
	ExecutedQty   float64 `json:"executedQty,string"`
	Status        string  `json:"status"`
	TimeInForce   string  `json:"timeInForce"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	StopPrice     float64 `json:"stopPrice,string"`
	IcebergQty    float64 `json:"icebergQty,string"`
	IsWorking     bool    `json:"isWorking"`
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

func Initialize(url, key, secret, eth_fee string) {

	fmt.Println("initializing binance package")

	api_url = url
	api_key = key
	api_secret = secret
	api_eth_fee, _ = strconv.ParseFloat(eth_fee, 64)

}

func Get_balances(tokens map[string]bool) map[string]float64 {

	var endpoint = "/api/v3/account"
	var holdings = make(map[string]float64)
	var data = new(Holdings)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, true)

	err := json.Unmarshal(body, &data)
	check(err)

	// remove tokens that we don't care about
	for _, v := range data.Holdings {

		symbol := v.Symbol
		amount, err := strconv.ParseFloat(v.Amount, 64)
		check(err)

		if tokens[symbol] {
			holdings[symbol] = amount
		}
	}

	return holdings

}

func Get_price(tokens map[string]bool) map[string]float64 {

	var endpoint = "/api/v3/ticker/price"
	var prices = make(map[string]float64)
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
		is_eth_pair := strings.HasSuffix(symbol, "ETH")
		token := strings.TrimSuffix(symbol, "ETH")
		price, err := strconv.ParseFloat(v.Price, 64)
		check(err)

		if is_eth_pair && tokens[token] {
			prices[token+"-ETH"] = price
		}
	}

	return prices
}

func Get_listed_tokens() []string {

	var endpoint = "/api/v3/ticker/price"
	var tokens []string
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
		is_eth_pair := strings.HasSuffix(symbol, "ETH")
		token := strings.TrimSuffix(symbol, "ETH")

		if is_eth_pair {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

func Place_sell_order(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	token += "ETH"
	var endpoint = fmt.Sprintf("/api/v3/order?symbol=%s&side=%s&type=%s&quantity=%d&price=%f&timeInForce=GTC", token, "SELL", "LIMIT", quantity, price)
	var place_order = new(Place_order)
	var body []byte

	// perform api call
	body = execute("POST", api_url+endpoint, true)

	err := json.Unmarshal(body, &place_order)
	check(err)

	if place_order.Id == "" {
		return "", false
	}

	return place_order.Id.String(), true

}

func Check_if_sold(token, sell_tx_id string) (float64, bool) {

	token += "ETH"
	var endpoint = fmt.Sprintf("/api/v3/order?orderId=%s&symbol=%s", sell_tx_id, token)
	var order = new(Order)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.OrigQty != 0 && order.OrigQty == order.ExecutedQty {
		return order.OrigQty * order.Price, true
	}

	return 0.0, false

}

func Start_transfer(token, destination string, amount float64) (string, bool) {

	var endpoint = fmt.Sprintf("/wapi/v3/withdraw.html?address=%s&amount=%f&asset=%s&name=bot", destination, amount, token)
	var transfer = new(Transfer_request)
	var body []byte

	// perform api call
	body = execute("POST", api_url+endpoint, true)

	err := json.Unmarshal(body, &transfer)
	check(err)

	if transfer.Id == "" {
		return "", false
	}

	return transfer.Id, true

}

func Check_if_transferred(sell_cost float64) bool {

	var endpoint = fmt.Sprintf("/wapi/v3/depositHistory.html?asset=ETH&status=1")
	var deposits = new(Deposits)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, true)

	err := json.Unmarshal(body, &deposits)
	check(err)

	for _, d := range deposits.List {
		if d.Amount == sell_cost {
			return true
		}
	}

	return false

}

func Place_buy_order(token string, quantity, price float64) (string, bool) {

	token += "ETH"
	var endpoint = fmt.Sprintf("/api/v3/order?symbol=%s&side=%s&type=%s&quantity=%f&price=%f&timeInForce=GTC", token, "BUY", "LIMIT", quantity, price)
	var place_order = new(Place_order)
	var body []byte

	// perform api call
	body = execute("POST", api_url+endpoint, true)

	err := json.Unmarshal(body, &place_order)
	check(err)

	if place_order.Id == "" {
		return "", false
	}

	return place_order.Id.String(), true

}

func Check_if_bought(token, buy_tx_id string) bool {

	token += "ETH"
	var endpoint = fmt.Sprintf("/api/v3/order?orderId=%s&symbol=%s", buy_tx_id, token)
	var order = new(Order)
	var body []byte

	// perform api call
	body = execute("GET", api_url+endpoint, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.OrigQty != 0 && order.OrigQty == order.ExecutedQty {
		return true
	}

	return false

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
