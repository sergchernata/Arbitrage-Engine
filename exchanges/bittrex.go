package bittrex

import (

	"fmt"

)

var url, key, secret string

func Initialize(api_url string, api_key string, api_secret string) {

	url = api_url
	key = api_key
	secret = api_secret

}

func Test() {

	fmt.Println(url)
	
}