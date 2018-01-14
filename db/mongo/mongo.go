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

	// not sure why this doesn't work
	// seems to do with the database
	// being used for user authentication
	//
	// mongoDBDialInfo := &mgo.DialInfo{
	// 	Addrs:    []string{host},
	// 	Database: database,
	// 	Username: username,
	// 	Password: password,
	// 	Timeout:  60 * time.Second,
	// }

	// session, err := mgo.DialWithInfo(mongoDBDialInfo)
	// if err != nil {
	// 	panic(err)
	// }

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
