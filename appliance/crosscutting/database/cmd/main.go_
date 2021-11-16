package main

import (
	"flag"
	"log"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-fareCollection/internal/database"
)

var pathdb string
var datadb string
var collection string
var prefix string
var update bool
var data string

func init() {
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdb/temp", "path to database file")
	flag.StringVar(&datadb, "datadb", "temp", "database name")
	flag.StringVar(&collection, "collection", "tempcll", "collection name")
	flag.StringVar(&prefix, "keyname", "", "prefix to query")
	flag.StringVar(&data, "data", "", "cdata to update")
	flag.BoolVar(&update, "update", false, "update data?")
}

func main() {

	flag.Parse()

	rootctx := actor.NewActorSystem().Root
	db, err := database.Open(rootctx, pathdb)
	if err != nil {
		log.Fatalln(err)
	}
	svc := database.NewService(db)

	query := func(data []byte) bool {
		// log.Printf("el: %s", data)
		return true
	}
	time.Sleep(3 * time.Second)

	if update {
		res, err := svc.Update(prefix, []byte(data), datadb, collection)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("result: %s", res)
		return
	}
	if err := svc.Query(datadb, collection,
		prefix, false, query); err != nil {
		log.Fatalln(err)
	}
}
