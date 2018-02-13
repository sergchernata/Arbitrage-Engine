package mongo

import (
	"fmt"
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

func Initialize(host string, database string, username string, password string) {

	fmt.Println("initializing mongo package")

	session, err := mgo.Dial(host)
	utils.Check(err)

	err = session.DB("admin").Login(username, password)
	utils.Check(err)

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

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.SellCompleted, "sell_cost": amount}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Transfer_started(row_id, tx_id, buy_exchange string, buy_price float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.TransferStarted, "buy_exchange": buy_exchange}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Transfer_completed(row_id string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.TransferCompleted}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Buy_order_placed(row_id, tx_id string, quantity, buy_price float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.BuyPlaced, "buy_tx_id": tx_id, "buy_price": buy_price, "buy_quantity": quantity}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Buy_order_completed(row_id string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.BuyCompleted}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Token_reset_completed(row_id, transaction_id string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	query := bson.M{"_id": bson.ObjectIdHex(row_id)}
	change := bson.M{"$set": bson.M{"status": utils.BalancesReset}}
	err := collection.Update(query, change)
	utils.Check(err)

}

func Get_incomplete_transactions() []utils.Transaction {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("transactions")

	var transactions []utils.Transaction

	query := bson.M{"status": bson.M{"$lt": utils.BalancesReset}}
	err := collection.Find(query).All(&transactions)
	utils.Check(err)

	return transactions

}

func Save_comparisons(comparisons map[string]utils.Comparison) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("comparisons")
	bulk := collection.Bulk()

	var rows []interface{}

	for token, comparison := range comparisons {

		row := bson.M{
			"token":      token,
			"comparison": comparison,
			"timestamp":  time.Now(),
		}

		rows = append(rows, row)

	}

	bulk.Insert(rows...)

	_, err := bulk.Run()
	utils.Check(err)

}

func Save_prices(exchange_prices map[string]map[string]float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("prices")
	bulk := collection.Bulk()

	var rows []interface{}

	for exchange, prices := range exchange_prices {

		for token, value := range prices {

			row := Price{
				Token:     token,
				Price:     value,
				Exchange:  exchange,
				Timestamp: time.Now(),
			}

			rows = append(rows, row)

		}

	}

	bulk.Insert(rows...)

	_, err := bulk.Run()
	utils.Check(err)

}

func Save_balances(exchange_balances map[string]map[string]float64) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("balances")
	bulk := collection.Bulk()

	var rows []interface{}

	for exchange, tokens := range exchange_balances {

		for token, amount := range tokens {

			row := utils.Balance{
				Token:     token,
				Amount:    amount,
				Exchange:  exchange,
				Timestamp: time.Now(),
			}

			rows = append(rows, row)

		}

	}

	bulk.Insert(rows...)

	_, err := bulk.Run()
	utils.Check(err)

}

func Get_balances(from_date, to_date time.Time) []utils.Balance {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("balances")

	var balances []utils.Balance

	query := bson.M{"timestamp": bson.M{"$gt": from_date, "$lt": to_date}}
	err := collection.Find(query).All(&balances)
	utils.Check(err)

	return balances

}

func Get_transactions(from_date, to_date time.Time) []utils.Transaction {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("balances")

	var transactions []utils.Transaction

	query := bson.M{"timestamp": bson.M{"$gt": from_date, "$lt": to_date}}
	err := collection.Find(query).All(&transactions)
	utils.Check(err)

	return transactions

}

func Get_token_analysis(token string) utils.Analysis {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("comparisons")
	var analysis utils.Analysis
	biggest_diff := make(map[string]utils.Comparison)
	smallest_diff := make(map[string]utils.Comparison)

	pipe := collection.Pipe(
		[]bson.M{
			bson.M{
				"$match": bson.M{
					"token": token,
				},
			},
			bson.M{
				"$group": bson.M{
					"_id":      nil,
					"avg_diff": bson.M{"$avg": "$comparison.difference"},
				},
			},
		},
	)

	err := pipe.One(&analysis)

	if err == nil {

		current_time := time.Now()
		query_time := current_time.Add(-30 * 24 * time.Hour)
		query := bson.M{"token": token, "timestamp": bson.M{"$gt": query_time}}

		// record of biggest price difference in last 30 days
		err := collection.Find(query).Select(bson.M{"comparison": 1}).Sort("-comparison.difference").One(&biggest_diff)
		utils.Check(err)

		// record of smallest price difference in last 30 days
		err = collection.Find(query).Select(bson.M{"comparison": 1}).Sort("comparison.difference").One(&smallest_diff)
		utils.Check(err)

		analysis.Max_diff = biggest_diff["comparison"].Difference
		analysis.Min_diff = smallest_diff["comparison"].Difference
		analysis.Max_diff_min_exch = biggest_diff["comparison"].Min_exchange
		analysis.Max_diff_max_exch = biggest_diff["comparison"].Max_exchange
		analysis.Max_diff_time = biggest_diff["comparison"].Timestamp
		analysis.Timestamp = time.Now()

	}

	return analysis

}

//-----------------------------------//
// utility data storage
//-----------------------------------//

func Get_listed_token_exchanges(token string) []string {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("utils")

	var tokens utils.Listed

	query := bson.M{"type": "listed_tokens"}
	err := collection.Find(query).One(&tokens)
	utils.Check(err)

	return tokens.Data[token]

}

func Update_listed_tokens(tokens map[string][]string) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("utils")

	query := bson.M{"type": "listed_tokens"}

	row := bson.M{
		"type":    "listed_tokens",
		"data":    tokens,
		"updated": time.Now(),
	}

	_, err := collection.Upsert(query, row)
	utils.Check(err)

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

	row := utils.Flag{
		Message:   message,
		Timestamp: time.Now(),
	}

	if err := collection.Insert(row); err != nil {
		panic(err)
	}

}

func Get_flags() []utils.Flag {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("flags")

	var flags []utils.Flag

	err := collection.Find(nil).All(&flags)
	utils.Check(err)

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

	row := utils.Log{
		Message:   message,
		Timestamp: time.Now(),
	}

	if err := collection.Insert(row); err != nil {
		panic(err)
	}

}

func Get_logs() []utils.Log {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("log")

	var logs []utils.Log

	err := collection.Find(nil).All(&logs)
	utils.Check(err)

	return logs

}

func Empty_log() {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("log")
	collection.RemoveAll(nil)

}

//-----------------------------------//
// discord-specific methods
//-----------------------------------//
func Get_discorder(user_id string) utils.Discorder {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	var discroder utils.Discorder

	query := bson.M{"id": user_id}
	collection.Find(query).One(&discroder)

	return discroder

}

func Create_discorder(discorder utils.Discorder) {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	if err := collection.Insert(discorder); err != nil {
		panic(err)
	}

}

func Discorder_toggle(author_id string, toggle bool) bool {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	query := bson.M{"id": author_id}
	change := bson.M{"$set": bson.M{"on": toggle}}
	err := collection.Update(query, change)

	if err != nil {
		return false
	}

	return true

}

func Discorder_update_tokens(author_id, action, token string) bool {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	query := bson.M{"id": author_id}
	change := bson.M{action: bson.M{"tokens": token}}

	// special command for wiping all tokens
	if action == "$pull" && token == "ALL" {
		change = bson.M{action: bson.M{"tokens": bson.M{"$exists": true}}}
	}

	err := collection.Update(query, change)

	if err != nil {
		return false
	}

	return true

}

func Discorder_set_threshold(author_id string, threshold float64) bool {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	query := bson.M{"id": author_id}
	change := bson.M{"$set": bson.M{"threshold": threshold}}
	err := collection.Update(query, change)

	if err != nil {
		return false
	}

	return true

}

func Discorder_set_frequency(author_id string, frequency float64) bool {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	query := bson.M{"id": author_id}
	change := bson.M{"$set": bson.M{"frequency": frequency}}
	err := collection.Update(query, change)

	if err != nil {
		return false
	}

	return true

}

func Discorder_update_notification_time(author_id string) bool {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	query := bson.M{"id": author_id}
	change := bson.M{"$set": bson.M{"last_notification": time.Now()}}
	err := collection.Update(query, change)

	if err != nil {
		return false
	}

	return true

}

func Get_active_discorders() []utils.Discorder {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	var discroders []utils.Discorder

	query := bson.M{"on": true}
	collection.Find(query).All(&discroders)

	return discroders

}

func Get_discorders_distinct_tokens() []string {

	session := mgoSession.Clone()
	defer session.Close()

	collection := session.DB(mgoDatabase).C("discord")

	var tokens []string

	query := bson.M{"on": true}
	collection.Find(query).Distinct("tokens", &tokens)

	return tokens

}
