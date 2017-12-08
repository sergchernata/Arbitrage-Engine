package binance

import (
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"
)

var api_url, api_key, api_secret string

func Initialize(url string, key string, secret string) {

	api_url = url
	api_key = key
	api_secret = secret

}

func Get_price(token string) {

	endpoint := "/api/v3/ticker/price"

	execute(api_url + endpoint)

}

func execute(url string) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.Header.Set("User-Agent", "test")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer res.Body.Close()

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	fmt.Println(string(body))

}
