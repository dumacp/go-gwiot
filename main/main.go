package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/app"
	"github.com/dumacp/go-gwiot/internal/env"
	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/pubsub"
	"github.com/dumacp/go-gwiot/internal/remote"
	remqtt "github.com/dumacp/go-gwiot/internal/remote-mqtt"
	renatsio "github.com/dumacp/go-gwiot/internal/remote-natsio"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/google/uuid"
)

const (
	showVersion = "1.2.14"
)

var debug bool
var disableRetransmission bool
var test bool
var logStd bool
var version bool
var pathdb string
var hostname string
var isnats bool
var retryDays int

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&test, "test", false, "test enable")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&pathdb, "pathdb", "/SD/boltdbs/gwiotdb", "path to db")
	flag.StringVar(&hostname, "hostname", "", "test hostname")
	flag.BoolVar(&disableRetransmission, "disableRetransmission", false, "disable retransmission")
	flag.IntVar(&retryDays, "retryDays", 2, "days to retransmission (0 for all days)")
	// flag.BoolVar(&isnats, "isNatsIO", false, "coonect to nats.io broker")
	flag.StringVar(&utils.RemoteBrokerURL, "remoteBrokerURL", "", "example: \"wss://fleet-mqtt.nebulae.com.co/mqtt\", remote broker url")
	flag.StringVar(&utils.ExternalRemoteBrokerURL, "externalBrokerURL", "", "external remote broker url")
	flag.StringVar(&utils.ExternalRemoteBrokerUser, "externalBrokerUser", "", "external remote broker user")
	flag.StringVar(&utils.ExternalRemoteBrokerPass, "externalBrokerPass", "", "external remote broker pass")
	flag.StringVar(&utils.ExternalRemoteBrokerTopic, "externalBrokerTopic", "", "external remote broker topic")

}

func main() {

	flag.Parse()

	if len(hostname) > 0 {
		utils.SetHostname(hostname)
	}

	if version {
		fmt.Printf("version: %s\n", showVersion)
		os.Exit(2)
	}
	initLogs(debug, logStd)

	getENV()

	uuid.SetNodeID([]byte(utils.Hostname()))

	rootContext := actor.NewActorSystem().Root
	pubsub.Init(rootContext)

	var propsRemote *actor.Props

	if !isnats {
		conf := remqtt.JwtConf{
			User:         utils.Hostname(),
			Pass:         utils.Hostname(),
			Realm:        keyc.Realm,
			ClientID:     keyc.Clientid,
			ClientSecret: keyc.ClientSecret,
			KeycloakURL:  keyc.Keycloakurl,
		}

		client := remqtt.NewRemote(test, &conf)

		propsClient := actor.PropsFromProducer(func() actor.Actor { return client })
		reclient := remote.NewRemote(propsClient)
		if disableRetransmission {
			reclient.DisableReplay(true)
		}
		if retryDays > 0 {
			reclient.RetryDaysReplay(retryDays)
		}

		propsRemote = actor.PropsFromProducer(func() actor.Actor { return reclient })

	} else {
		reclient := renatsio.NewRemote(test)
		if disableRetransmission {
			reclient.DisableReplay(true)
		}
		propsRemote = actor.PropsFromProducer(func() actor.Actor { return reclient })
	}

	propsApp := actor.PropsFromProducer(func() actor.Actor { return app.NewApp(propsRemote) })
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
	if len(keyc.Keycloakurl) <= 0 {
		if len(os.Getenv("KEYCLOAK_URL_DEVICES")) > 0 {
			keyc.Keycloakurl = os.Getenv("KEYCLOAK_URL_DEVICES")
		} else {
			keyc.Keycloakurl = env.Keycloakurl_
		}
	}
	if len(keyc.Redirecturl) <= 0 {

		if len(os.Getenv("REDIRECT_URL_DEVICES")) > 0 {
			keyc.Redirecturl = os.Getenv("REDIRECT_URL_DEVICES")
		} else {
			keyc.Redirecturl = env.Redirecturl_
		}
	}
	if len(utils.RemoteBrokerURL) <= 0 {
		if len(os.Getenv("BROKER_URL_DEVICES")) > 0 {
			utils.RemoteBrokerURL = os.Getenv("BROKER_URL_DEVICES")
		} else {
			utils.RemoteBrokerURL = env.RemoteBrokerURL_
		}
	}
	if len(keyc.Realm) <= 0 {
		if len(os.Getenv("REALM_DEVICES")) > 0 {
			keyc.Realm = os.Getenv("REALM_DEVICES")
		} else {
			keyc.Realm = env.Realm_
		}
	}
	if len(keyc.ClientSecret) <= 0 {
		if len(os.Getenv("CLIENTSECRET_DEVICES")) > 0 {
			keyc.ClientSecret = os.Getenv("CLIENTSECRET_DEVICES")
		} else {
			keyc.ClientSecret = env.ClientSecret_
		}
	}
	if len(keyc.Clientid) <= 0 {

		if len(os.Getenv("CLIENTID_DEVICES")) > 0 {
			keyc.Clientid = os.Getenv("CLIENTID_DEVICES")
		} else {
			keyc.Clientid = env.Clientid_
		}
	}

	if len(utils.ExternalRemoteBrokerURL) <= 0 {
		if len(os.Getenv("BROKER_URL_EXTERNAL")) > 0 {
			utils.ExternalRemoteBrokerURL = os.Getenv("BROKER_URL_EXTERNAL")
		} else {
			utils.ExternalRemoteBrokerURL = env.ExternalRemoteBrokerURL_
		}
	}
	if len(utils.ExternalRemoteBrokerUser) <= 0 {
		if len(os.Getenv("BROKER_USER_EXTERNAL")) > 0 {
			utils.ExternalRemoteBrokerUser = os.Getenv("BROKER_USER_EXTERNAL")
		}
	}
	if len(utils.ExternalRemoteBrokerPass) <= 0 {
		if len(os.Getenv("BROKER_PASS_EXTERNAL")) > 0 {
			utils.ExternalRemoteBrokerPass = os.Getenv("BROKER_PASS_EXTERNAL")
		}
	}
	if len(utils.ExternalRemoteBrokerTopic) <= 0 {
		if len(os.Getenv("BROKER_TOPIC_EXTERNAL")) > 0 {
			utils.ExternalRemoteBrokerTopic = os.Getenv("BROKER_TOPIC_EXTERNAL")
		} else {
			utils.ExternalRemoteBrokerTopic = env.ExternalRemoteBrokerTopic_
		}
	}
	if len(utils.Url) <= 0 {
		if len(os.Getenv("APPFARE_URL")) > 0 {
			utils.Url = os.Getenv("APPFARE_URL")
		} else {
			utils.Url = env.Urlin_
		}
	}
	if len(utils.User) <= 0 {
		if len(os.Getenv("APPFARE_USER")) > 0 {
			utils.User = os.Getenv("APPFARE_USER")
		} else {
			utils.User = env.User_
		}
	}
	if len(utils.PassCode) <= 0 {
		if len(os.Getenv("APPFARE_PASSCODE")) > 0 {
			utils.PassCode = os.Getenv("APPFARE_PASSCODE")
		} else {
			utils.PassCode = env.PaccCode_
		}
	}

	fmt.Printf("keycloakurl: %s\n", keyc.Keycloakurl)
	fmt.Printf("redirecturl: %s\n", keyc.Redirecturl)
	fmt.Printf("remoteMqttBrokerURL: %s\n", utils.RemoteBrokerURL)
	fmt.Printf("realm: %s\n", keyc.Realm)
	fmt.Printf("clientSecret: %s\n", keyc.ClientSecret)
	fmt.Printf("clientid: %s\n", keyc.Clientid)
}
