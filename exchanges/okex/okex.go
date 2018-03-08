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

	// utility
	"../../utils"
)

var api_url, api_key, api_secret, api_tradepw string
var api_eth_fee float64

type Deposits struct {
	List []struct {
		Addr              string  `json:"addr"`
		Account           string  `json:"account"`
		Amount            float64 `json:"amount,string"`
		Transaction_value string  `json:"transaction_value"`
		Fee               string  `json:"fee"`
		Status            int     `json:"status,Number"`
	} `json:"records"`
}

type Place_order struct {
	Success bool        `json:"result"`
	Id      json.Number `json:"order_id,Number"`
}

type Place_transfer struct {
	Success bool   `json:"result"`
	Id      string `json:"withdraw_id"`
}

type Orders struct {
	Success bool `json:"result"`
	List    []struct {
		Amount      float64     `json:"amount,Number"`
		Avg_price   float64     `json:"avg_price,Number"`
		Deal_amount float64     `json:"deal_amount,Number"`
		Order_id    json.Number `json:"order_id,Number"`
		Orders_id   json.Number `json:"orders_id,Number"`
		Price       float64     `json:"price,Number"`
		Status      int         `json:"status,Number"`
		Symbol      string      `json:"symbol"`
		Type        string      `json:"type"`
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

func Initialize(url, key, secret, tradepw, eth_fee string) {

	fmt.Println("initializing okex package")

	api_url = url
	api_key = key
	api_secret = secret
	api_tradepw = tradepw
	api_eth_fee, _ = strconv.ParseFloat(eth_fee, 64)

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
	// check if there's a way to deal with timeouts and errors here
	err := json.Unmarshal(body, &data)
	if err != nil {
		return holdings
	}

	// remove tokens that we don't care about
	for token, amount := range data.Info.Funds.Free.(map[string]interface{}) {

		if amount.(string) != "" {
			token = strings.ToUpper(token)
			amount, err := strconv.ParseFloat(amount.(string), 64)
			utils.Check(err)

			if tokens[token] {
				holdings[token] = amount
			}
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
		if err != nil {
			return prices
		}

		if data.Data.Last != "" {
			price, err := strconv.ParseFloat(data.Data.Last, 64)
			utils.Check(err)

			prices[token+"-ETH"] = price
		}

	}

	return prices
}

func Get_listed_tokens(search []string) []string {

	var endpoint = "/ticker.do"
	var tokens []string
	var body []byte

	// perform api call per token
	for _, token := range search {

		var data = new(Prices)
		var params = fmt.Sprintf("symbol=%s", token+"_ETH")

		// perform api call
		body = execute("GET", api_url, endpoint, params)

		err := json.Unmarshal(body, &data)
		if err != nil || data.Data.Buy == "" {
			continue
		}

		tokens = append(tokens, token)

	}

	return tokens
}

func Place_sell_order(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	var endpoint = "/trade.do"
	var params = fmt.Sprintf("amount=%d&api_key=%s&price=%f&symbol=%s&type=%s", quantity, api_key, price, token+"_ETH", "sell")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var place_order = new(Place_order)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &place_order)
	utils.Check(err)

	if place_order.Success {
		return place_order.Id.String(), true
	}

	return "", false

}

func Check_if_sold(token, sell_tx_id string) (float64, bool) {

	var amount = 0.0
	var endpoint = "/order_info.do"
	var params = fmt.Sprintf("api_key=%s&order_id=%s&symbol=%s", api_key, sell_tx_id, token+"_ETH")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var orders = new(Orders)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &orders)
	utils.Check(err)

	for _, order := range orders.List {
		if order.Order_id.String() == sell_tx_id && order.Status == 2 {
			return order.Amount * order.Price, true
		}
	}

	return amount, false

}

func Start_transfer(token, destination string, amount float64) (string, bool) {

	var endpoint = "/withdraw.do"
	var params = fmt.Sprintf("api_key=%s&chargefee=0.01&symbol=%s&target=address&trade_pwd=%s&withdraw_address=%s&withdraw_amount=%f",
		api_key, token+"_ETH", api_tradepw, destination, amount)
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var transfer = new(Place_transfer)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &transfer)
	utils.Check(err)

	if transfer.Success == false {
		return "", false
	}

	return transfer.Id, true

}

func Check_if_transferred(sell_cost float64) bool {

	var endpoint = "/account_records.do"
	var params = fmt.Sprintf("api_key=%s&current_page=1&page_length=10&symbol=eth&type=0", api_key)
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var deposits = new(Deposits)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &deposits)
	utils.Check(err)

	for _, deposit := range deposits.List {
		if deposit.Amount == sell_cost && deposit.Status == 1 {
			return true
		}
	}

	return false

}

func Place_buy_order(token string, amount, price float64) (string, bool) {

	token += "_ETH"
	var endpoint = "/trade.do"
	var params = fmt.Sprintf("amount=%f&api_key=%s&price=%f&symbol=%s&type=%s", amount, api_key, price, token, "buy")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var place_order = new(Place_order)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &place_order)
	utils.Check(err)

	if place_order.Success {
		return place_order.Id.String(), true
	}

	return "", false

}

func Check_if_bought(token, buy_tx_id string) bool {

	var endpoint = "/order_info.do"
	var params = fmt.Sprintf("api_key=%s&order_id=%s&symbol=%s", api_key, buy_tx_id, token+"_ETH")
	var signature = make_signature(params + "&secret_key=" + api_secret)
	var orders = new(Orders)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &orders)
	utils.Check(err)

	for _, order := range orders.List {
		if order.Order_id.String() == buy_tx_id && order.Status == 2 {
			return true
		}
	}

	return false

}

func make_signature(params string) string {

	hasher := md5.New()
	hasher.Write([]byte(params))
	return strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))

}

func execute(method string, url string, endpoint string, params string) []byte {

	req, err := http.NewRequest(method, url+endpoint+"?"+params, nil)
	utils.Check(err)

	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	res, err := client.Do(req)
	utils.Check(err)

	if res != nil {

		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		utils.Check(err)

		return body

	}

	return nil
}
