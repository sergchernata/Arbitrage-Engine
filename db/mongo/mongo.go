package mongo

import (
	"fmt"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
	//"log"
)

func Initialize(host string, database string, username string, password string) {

	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    []string{host},
		Database: database,
		Username: username,
		Password: password,
	}

	session, err := mgo.DialWithInfo(mongoDBDialInfo)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	fmt.Println("connected")

}

func Query() {

	// c := session.DB("test").C("people")
	// err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
	//  &Person{"Cla", "+55 53 8402 8510"})
	// if err != nil {
	//  log.Fatal(err)
	// }

	// result := Person{}
	// err = c.Find(bson.M{"name": "Ale"}).One(&result)
	// if err != nil {
	//  log.Fatal(err)
	// }

	// fmt.Println("Phone:", result.Phone)

}
