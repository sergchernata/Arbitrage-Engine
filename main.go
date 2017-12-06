package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	//"strconv"
)

var Props = make(map[string]string)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {

	dat, err := ioutil.ReadFile(".env")
	check(err)

	lines := strings.Split(string(dat), "\n")

	for _, l := range lines {
		split := strings.Split(l,"=")
		Props[split[0]] = split[1]
	}
	
}

func main() {

	fmt.Println(Props)

}