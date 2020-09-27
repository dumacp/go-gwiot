package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/persistence"
	"github.com/dumacp/go-gwiot/appliance/business"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/comm/pubsub"
)

const (
	showVersion = "1.0.1"
)

var debug bool
var logStd bool
var version bool
var pathdb string
var hostname string

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdbs/gwiotdb", "path to db")
	flag.StringVar(&hostname, "hostname", "", "test hostname")
}

func main() {

	flag.Parse()

	if len(hostname) > 0 {
		business.SetHostname(hostname)
	}

	if version {
		fmt.Printf("version: %s\n", showVersion)
		os.Exit(2)
	}
	initLogs(debug, logStd)

	pubsub.Init()

	// var provider *provider
	// var err error
	// if !disablePersistence {
	provider, err := newProvider(pathdb, 10)
	if err != nil {
		log.Fatalln(err)
	}
	// }

	rootContext := actor.EmptyRootContext

	remote := business.NewRemote()

	propsRemote := actor.PropsFromProducer(func() actor.Actor { return remote }).
		WithReceiverMiddleware(persistence.Using(provider))

	propsApp := actor.PropsFromProducer(func() actor.Actor { return business.NewApp(propsRemote) })
	rootContext.SpawnNamed(propsApp, "deviceIot")

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for {
		select {
		case <-finish:
			log.Print("Finish")
			return
		}
	}
}
