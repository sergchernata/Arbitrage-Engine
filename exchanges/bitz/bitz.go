package bitz

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
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

func Initialize(url string, key string, secret string, tradepw string) {

	fmt.Println("initializing bitz package")

	api_url = url
	api_key = key
	api_secret = secret
	api_tradepw = tradepw

}

func Get_price(tokens map[string]bool) map[string]float64 {

	var params = ""
	var endpoint = "/api_v1/tickerall"
	var data interface{}
	var prices = make(map[string]float64)
	var body []byte

	// perform api call
	body = execute("GET", api_url, endpoint, params, false)

	err := json.Unmarshal(body, &data)
	check(err)

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
		price, err := strconv.ParseFloat(details["last"].(string), 64)
		check(err)

		if is_eth_pair && tokens[token] {
			prices[token+"-ETH"] = price
		}
	}

	return prices
}

func Sell(token string, quantity int, price float64) (transaction_id string, sell_placed bool) {

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
	body = execute("POST", api_url, endpoint, params, true)

	err := json.Unmarshal(body, &order)
	check(err)

	if order.Data.Id == "" {
		return "", false
	}

	return order.Data.Id, true

}

func make_signature(params string) string {

	hasher := md5.New()
	hasher.Write([]byte(params))
	return hex.EncodeToString(hasher.Sum(nil))

}

func execute(method string, url string, endpoint string, params string, auth bool) []byte {

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
