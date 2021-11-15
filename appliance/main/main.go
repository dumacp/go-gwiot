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
	showVersion = "1.1.2"
)

var debug bool
var disableRetransmission bool
var test bool
var logStd bool
var version bool
var pathdb string
var hostname string

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&test, "test", false, "test enable")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdbs/gwiotdb", "path to db")
	flag.StringVar(&hostname, "hostname", "", "test hostname")
	flag.BoolVar(&disableRetransmission, "disableRetransmission", false, "disable retransmission")
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

	getENV()

	rootContext := actor.NewActorSystem().Root
	pubsub.Init(rootContext)

	// var provider *provider
	// var err error
	// if !disablePersistence {
	provider, err := newProvider(pathdb, 10)
	if err != nil {
		log.Fatalln(err)
	}
	// }

	remote := business.NewRemote(test)
	if disableRetransmission {
		remote.DisableReplay(true)
	}

	// decider := func(reason interface{}) actor.Directive {
	// 	fmt.Println("handling failure for child")
	// 	return actor.StopDirective
	// }
	// supervisor := actor.NewOneForOneStrategy(10, 1000, decider)

	propsRemote := actor.PropsFromProducer(func() actor.Actor { return remote }).
		WithReceiverMiddleware(persistence.Using(provider))
		// WithSupervisor(supervisor)

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

func getENV() {
	if len(business.Keycloakurl) <= 0 {
		business.Keycloakurl = os.Getenv("KEYCLOAK_URL_DEVICES")
	}
	if len(business.Redirecturl) <= 0 {
		business.Redirecturl = os.Getenv("REDIRECT_URL_DEVICES")
	}
	if len(business.RemoteMqttBrokerURL) <= 0 {
		business.RemoteMqttBrokerURL = os.Getenv("BROKER_URL_DEVICES")
	}
	if len(business.Realm) <= 0 {
		business.Realm = os.Getenv("REALM_DEVICES")
	}
	if len(business.ClientSecret) <= 0 {
		business.ClientSecret = os.Getenv("CLIENTSECRET_DEVICES")
	}
	if len(business.Clientid) <= 0 {
		business.Clientid = os.Getenv("CLIENTID_DEVICES")
	}

	fmt.Printf("keycloakurl: %s\n", business.Keycloakurl)
	fmt.Printf("redirecturl: %s\n", business.Redirecturl)
	fmt.Printf("remoteMqttBrokerURL: %s\n", business.RemoteMqttBrokerURL)
	fmt.Printf("realm: %s\n", business.Realm)
	fmt.Printf("clientSecret: %s\n", business.ClientSecret)
	fmt.Printf("clientid: %s\n", business.Clientid)
}
