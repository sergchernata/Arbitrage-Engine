package bitz

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	// "reflect"
	"strconv"
	"strings"
	"time"

	// utility
	"../../utils"
)

var api_url, api_key, api_secret, api_tradepw string
var api_eth_fee float64

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

func Initialize(url, key, secret, tradepw, eth_fee string) {

	fmt.Println("initializing bitz package")

	api_url = url
	api_key = key
	api_secret = secret
	api_tradepw = tradepw
	api_eth_fee, _ = strconv.ParseFloat(eth_fee, 64)

}

func Get_balances(tokens map[string]bool) map[string]float64 {

	var holdings = make(map[string]float64)
	// var body []byte

	// for token, _ := range tokens {

	// 	var data = new(Holdings)
	// 	var endpoint = "/v1/account/" + token + "/balance"
	// 	var params = ""

	// 	// perform api call
	// 	body = execute("GET", api_url, endpoint, params)

	// 	err := json.Unmarshal(body, &data)
	// 	utils.Check(err)

	// 	holdings[data.Holding.Symbol] = data.Holding.Amount

	// }

	return holdings
}

func Get_price(tokens map[string]bool) map[string]float64 {

	var params = ""
	var endpoint = "/api_v1/tickerall"
	var data interface{}
	var prices = make(map[string]float64)
	var body []byte

	// perform api call
	body = execute("GET", api_url, endpoint, params)

	err := json.Unmarshal(body, &data)
	utils.Check(err)

	all := data.(map[string]interface{})
	allPrices := all["data"].(map[string]interface{})

	//parse data and format for return
	for k, v := range allPrices {

		// bitz formats pairs as "LINK_ETH"
		// they also use token as key itself, which is the reason
		// for parsing this data into a generic interface and not a struct
		details := v.(map[string]interface{})
		symbol := strings.ToUpper(k)
		is_eth_pair := strings.HasSuffix(symbol, "_ETH")
		token := strings.TrimSuffix(symbol, "_ETH")
		// TODO: throws error fmt.Println(details, reflect.TypeOf(details["last"]))
		price, err := strconv.ParseFloat(details["last"].(string), 64)
		utils.Check(err)

		if is_eth_pair && tokens[token] {
			prices[token+"-ETH"] = price
		}
	}

	return prices
}

func Get_listed_tokens() []string {

	var params = ""
	var endpoint = "/api_v1/tickerall"
	var data interface{}
	var tokens []string
	var body []byte

	// perform api call
	body = execute("GET", api_url, endpoint, params)

	err := json.Unmarshal(body, &data)
	utils.Check(err)

	all := data.(map[string]interface{})
	allPrices := all["data"].(map[string]interface{})

	//parse data and format for return
	for k, _ := range allPrices {

		// bitz formats pairs as "link_eth"
		// they also use token as key itself, which is the reason
		// for parsing this data into a generic interface and not a struct
		symbol := strings.ToUpper(k)
		is_eth_pair := strings.HasSuffix(symbol, "_ETH")
		token := strings.TrimSuffix(symbol, "_ETH")

		if is_eth_pair {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

func Place_sell_order(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

	token += "_ETH"
	var timestamp = strconv.Itoa(int(time.Now().Unix() * 1000))
	var params = fmt.Sprintf("api_key=%s&coin=%s&nonce=235195&number=%d&price=%f&timestamp=%d&tradepwd=%s&type=out&sign=%s",
		api_key, token, quantity, price, timestamp, api_tradepw)
	var signature = make_signature(params)
	var endpoint = "/api_v1/tradeAdd"
	var order = new(Order)
	var body []byte

	params = params + "&sign=" + signature

	// perform api call
	body = execute("POST", api_url, endpoint, params)

	err := json.Unmarshal(body, &order)
	utils.Check(err)

	if order.Data.Id == "" {
		return "", false
	}

	return order.Data.Id, true

}

func Check_if_sold(token, sell_tx_id string) (float64, bool) {

	var amount = 0.0
	return amount, true

}

func Start_transfer(token, destination string, amount float64) (string, bool) {

	return "", true

}

func Check_if_transferred(sell_cost float64) bool {

	return true

}

func Place_buy_order(token string, quantity, buy_cost float64) (string, bool) {

	return "", true

}

func Check_if_bought(token, buy_tx_id string) bool {

	return true

}

func make_signature(params string) string {

	hasher := md5.New()
	hasher.Write([]byte(params))
	return hex.EncodeToString(hasher.Sum(nil))

}

func execute(method string, url string, endpoint string, params string) []byte {

	req, err := http.NewRequest(method, url+endpoint+"?"+params, nil)
	utils.Check(err)

	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	res, err := client.Do(req)
	utils.Check(err)

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	utils.Check(err)

	return body

}
