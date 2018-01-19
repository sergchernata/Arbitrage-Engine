package utils

import (
	"gopkg.in/mgo.v2/bson"
	"math"
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

func main() {

}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}
