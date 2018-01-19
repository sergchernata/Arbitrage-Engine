package mongo

import (
	//"fmt"
	// go get gopkg.in/mgo.v2
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
	//"log"

	// utility
	"../../utils"
)

var mgoSession *mgo.Session
var mgoDatabase string

type Price struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Token     string
	Price     string
	Exchange  string
	Timestamp time.Time
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

	row := utils.Transaction{
		Status:        utils.SellPlaced,
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

func Get_incomplete_transactions() []utils.Transaction {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	var transactions []utils.Transaction

	query := bson.M{"_id": bson.M{"$lt": utils.BuyCompleted}}
	err := collection.Find(query).All(&transactions)
	check(err)

	return transactions

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
