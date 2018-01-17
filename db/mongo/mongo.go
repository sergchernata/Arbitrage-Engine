package mongo

import (
	//"fmt"
	// go get gopkg.in/mgo.v2
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
	//"log"
)

var mgoSession *mgo.Session
var mgoDatabase string

type Status int

const(
	SellPlaced Status = iota
	SellCompleted
	TransferStarted
	TransferCompleted
	BuyPlaced
	BuyCompleted
)

type Price struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Token     string
	Price     string
	Exchange  string
	Timestamp time.Time
}

type Transaction struct {
	ID            bson.ObjectId `bson:"_id,omitempty"`
	Status        int
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Initialize(host string, database string, username string, password string) {

	session, err := mgo.Dial(host)
	check(err)

	err = session.DB("admin").Login(username, password)
	check(err)

	//defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	mgoSession = session
	mgoDatabase = database

}

func Create_transaction(token, exchange, transaction_id string, price float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	row := Transaction{
		Status:        SellPlaced
		Token:         token,
		Sell_price:    price,
		Sell_exchange: exchange,
		Sell_tx_id:    transaction_id,
		Timestamp:     time.Now(),
	}

	if err := collection.Insert(row); err != nil {
		panic(err)
	}

}

func Save_prices(tokens map[string]string, exchange string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("prices")

	for token, price := range tokens {

		row := Price{
			Token:     token,
			Price:     price,
			Exchange:  exchange,
			Timestamp: time.Now(),
		}

		if err := collection.Insert(row); err != nil {
			panic(err)
		}

	}

}
