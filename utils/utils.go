package utils

import (
	"gopkg.in/mgo.v2/bson"
	"log"
	"math"
	"os"
	"time"
)

type Status int

const (
	SellPlaced        Status = iota // 0
	SellCompleted                   // 1
	TransferStarted                 // 2
	TransferCompleted               // 3
	BuyPlaced                       // 4
	BuyCompleted                    // 5
	BalancesReset                   // 6
)

type Transaction struct {
	ID            bson.ObjectId `bson:"_id,omitempty"`
	Status        Status
	Token         string
	Sell_price    float64
	Sell_cost     float64
	Sell_quantity float64
	Sell_exchange string
	Sell_tx_id    string
	Buy_price     float64
	Buy_cost      float64
	Buy_quantity  float64
	Buy_exchange  string
	Buy_tx_id     string
	Timestamp     time.Time
}

type Log struct {
	Message   string
	Timestamp time.Time
}

type Flag struct {
	Message   string
	Timestamp time.Time
}

type Balance struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Token     string
	Amount    float64
	Exchange  string
	Timestamp time.Time
}

type Comparison struct {
	Min_price    float64
	Max_price    float64
	Min_exchange string
	Max_exchange string
	Difference   float64
	Timestamp    time.Time
}

type Discorder struct {
	ID                string
	Username          string
	Channel           string
	On                bool
	Tokens            []string
	Threshold         float64
	Frequency         float64
	Last_notification time.Time
	Timestamp         time.Time
}

type Analysis struct {
	ID                string
	Avg_diff          float64
	Max_diff          float64
	Min_diff          float64
	Max_diff_min_exch string
	Max_diff_max_exch string
	Max_diff_time     time.Time
	Timestamp         time.Time
}

type Listed struct {
	ID      string
	Data    map[string][]string
	Updated time.Time
}

func main() {
	if !FileExists("log") {
		CreateFile("log")
	}
}

func Check(e error) {
	if e != nil {

		f, err := os.OpenFile("log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic("couldn't open log file")
		}
		defer f.Close()

		log.SetOutput(f)
		log.Println(e.Error())

	}
}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}

func Ternary(a, b int, condition bool) int {
	if condition {
		return a
	} else {
		return b
	}
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func Merge_uniques(temp ...[]string) []string {

	var unique []string

	for _, tokens := range temp {

		for _, token := range tokens {

			if !StringInSlice(token, unique) {
				unique = append(unique, token)
			}

		}

	}

	return unique

}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func CreateFile(name string) error {
	fo, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		fo.Close()
	}()
	return nil
}
