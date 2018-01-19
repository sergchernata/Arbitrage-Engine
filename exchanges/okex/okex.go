package okex

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var api_url, api_key, api_secret, api_tradepw string

type Place_order struct {
	Success bool   `json:"result"`
	Id      string `json:"order_id"`
}

type Place_transfer struct {
	Success bool   `json:"result"`
	Id      string `json:"withdraw_id"`
}

type Open_orders struct {
	Success bool `json:"result"`
	Orders  []struct {
		Amount      string `json:"amount"`
		Avg_price   string `json:"avg_price"`
		Create_date string `json:"create_date"`
		Deal_amount string `json:"deal_amount"`
		Order_id    string `json:"order_id"`
		Orders_id   string `json:"orders_id"`
		Price       string `json:"price"`
		Status      string `json:"status"`
		Symbol      string `json:"symbol"`
		Type        string `json:"type"`
	} `json:"orders"`
}

type Holdings struct {
	Info struct {
		Funds struct {
			Free interface{} `json:"free"`
		} `json:"funds"`
	} `json:"info"`
	Success bool `json:"result"`
}

type Prices struct {
	Data struct {
		High string `json:"high,Number"`
		Low  string `json:"low,Number"`
		Sell string `json:"sell,Number"`
		Buy  string `json:"buy,Number"`
		Last string `json:"last,Number"`
	} `json:"ticker"`
	Date string `json:"date"`
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

	var endpoint = "/userinfo.do"
	var holdings = make(map[string]float64)
	var params = fmt.Sprintf("api_key=%s", api_key)
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var data = new(Holdings)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &data)
	check(err)

	// remove tokens that we don't care about
	for token, amount := range data.Info.Funds.Free.(map[string]interface{}) {

		token = strings.ToUpper(token)
		amount, err := strconv.ParseFloat(amount.(string), 64)
		check(err)

		if tokens[token] {
			holdings[token] = amount
		}
	}

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
		body = execute("GET", api_url, endpoint, params)

		err := json.Unmarshal(body, &data)
		check(err)

		price, err := strconv.ParseFloat(data.Data.Last, 64)
		check(err)

		prices[token] = price

	}

	return prices
}

func Place_sell_order(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	var endpoint = "/trade.do"
	var params = fmt.Sprintf("amount=%d&api_key=%s&price=%f&symbol=%s&type=%s", quantity, api_key, price, token+"_ETH", "sell")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var order = new(Place_order)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Success == false {
		return "", false
	}

	return order.Id, true

}

func Check_if_sold(token, sell_tx_id string) bool {

	var endpoint = "/order_info.do"
	var params = fmt.Sprintf("api_key=%s&order_id=%s&symbol=%s", api_key, sell_tx_id, token+"_ETH")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var open_orders = new(Open_orders)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &open_orders)
	check(err)

	// no open orders means that everything has been filled
	if len(open_orders.Orders) == 0 {
		return true
	}

	// if we find an open order with matching tx id
	// then the order has not been filled
	// this check is necessary because we may have
	// multiple ongoing transactions in the future
	for _, order := range open_orders.Orders {
		if order.Order_id == sell_tx_id {
			return false
		}
	}

	return true

}

func Start_transfer(token, destination string, amount float64) (string, bool) {

	var endpoint = "/withdraw.do"
	var params = fmt.Sprintf("api_key=%s&chargefee=0.01&symbol=%s&target=address&trade_pwd=%s&withdraw_address=%s&withdraw_amount=%d",
		api_key, token+"_ETH", api_tradepw, destination, amount)
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var transfer = new(Place_transfer)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &transfer)
	check(err)

	if transfer.Success == false {
		return "", false
	}

	return transfer.Id, true

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
	var order = new(Place_order)
	var body []byte

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Id == "" {
		return "", false
	}

	return order.Id, true

}

func make_signature(params string) string {

	hasher := md5.New()
	hasher.Write([]byte(params))
	return strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))

}

func execute(method string, url string, endpoint string, params string) []byte {

	req, err := http.NewRequest(method, url+endpoint+"?"+params, nil)
	check(err)

	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	res, err := client.Do(req)
	check(err)

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	check(err)

	return body

}
