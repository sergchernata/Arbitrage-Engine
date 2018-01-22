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
	Price     float64
	Exchange  string
	Timestamp time.Time
}

type Log struct {
	Message  string
	Timestamp time.Time
}

type Flag struct {
	Message  string
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

func Place_sell_order(token, exchange, transaction_id string, price float64) {

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

func Sell_order_completed(row_id, sell_exchange string, amount float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": row_id}
	change := bson.M{"$set": bson.M{"status": utils.SellCompleted, "sell_cost": amount}}
	err := collection.Update(query, change)
	check(err)

}

func Transfer_started(row_id, tx_id string, buy_price float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": row_id}
	change := bson.M{"$set": bson.M{"status": utils.TransferStarted}}
	err := collection.Update(query, change)
	check(err)

}

func Transfer_completed(row_id string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": row_id}
	change := bson.M{"$set": bson.M{"status": utils.TransferCompleted}}
	err := collection.Update(query, change)
	check(err)

}

func Buy_order_placed(row_id, tx_id string, quantity, buy_price float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": row_id}
	change := bson.M{"$set": bson.M{"status": utils.BuyPlaced, "buy_tx_id": tx_id, "buy_price": buy_price, "buy_quantity": quantity}}
	err := collection.Update(query, change)
	check(err)

}

func Buy_order_completed(row_id string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": row_id}
	change := bson.M{"$set": bson.M{"status": utils.BuyCompleted}}
	err := collection.Update(query, change)
	check(err)

}

func Get_incomplete_transactions() []utils.Transaction {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	var transactions []utils.Transaction

	query := bson.M{"status": bson.M{"$lt": utils.BuyCompleted}}
	err := collection.Find(query).All(&transactions)
	check(err)

	return transactions

}

func Save_prices(tokens map[string]float64, exchange string) {

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

//-----------------------------------//
// flag methods
//
// mostly used for killing bot
// in case of bad transaction
//-----------------------------------//
func Flag(message string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("flags")

	row := Flag{
		Message:   message,
		Timestamp: time.Now(),
	}

	if err := collection.Insert(row); err != nil {
		panic(err)
	}

}

func Get_flags() []Flag {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("flags")

	var flags []Flag

	err := collection.Find(nil).All(&flags)
	check(err)

	return flags

}

func Clear_flags() {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("flags")
	collection.RemoveAll(nil)

}

//-----------------------------------//
// log methods
//-----------------------------------//
func Log(message string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("log")

	row := Log{
		Message:   message,
		Timestamp: time.Now(),
	}

	if err := collection.Insert(row); err != nil {
		panic(err)
	}

}

func Get_logs() []Log {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("log")

	var logs []Log

	err := collection.Find(nil).All(&logs)
	check(err)

	return logs

}

func Empty_log() {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("log")
	collection.RemoveAll(nil)

}
