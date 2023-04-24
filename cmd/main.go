package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-gwiot/internal/pubsub"
	renatsio "github.com/dumacp/go-gwiot/internal/remote-natsio"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg/gwiot"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/google/uuid"
)

const (
	showVersion       = "1.2.12"
	ACTOR_INTSANCE_ID = "gwiot"
)

var (
	hostname     string
	remoteBroker string
	clientid     string
	clientSecret string
	redirecturl  string
	keycloakurl  string
	realm        string
	debug        bool
	test         bool
	logStd       bool
	version      bool
)

func init() {
	flag.BoolVar(&debug, "debug", false, "debug enable")
	flag.BoolVar(&test, "test", false, "test enable")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&hostname, "hostname", "", "test hostname")
	flag.StringVar(&remoteBroker, "remoteBroker", "", fmt.Sprintf("example: %q, remote broker url", remoteBroker_))
	flag.StringVar(&keycloakurl, "keycloakUrl", "", fmt.Sprintf("example: %q, keycloak url", keycloakurl_))
	flag.StringVar(&clientid, "clientID", "", fmt.Sprintf("example: %q, clientid in realm", clientid_))
	flag.StringVar(&clientSecret, "clientSecret", "", fmt.Sprintf("example: %q, client secret", clientSecret_))
	flag.StringVar(&redirecturl, "redirectUrl", "", fmt.Sprintf("example: %q, redirecturl url", redirecturl_))
	flag.StringVar(&realm, "realm", "", fmt.Sprintf("example: %q, realm name", realm_))
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

	sys := actor.NewActorSystem()
	pubsub.Init(sys.Root)

	portlocal := portlocal()
	rconfig := remote.Configure("127.0.0.1", portlocal)
	r := remote.NewRemote(sys, rconfig)
	prposNatsioclient := actor.PropsFromFunc(renatsio.NewChildNatsio(ACTOR_INTSANCE_ID).Receive)
	r.Register(gwiot.KIND_NAME, prposNatsioclient)
	r.Start()

	var (
		pidNatsio *actor.PID
	)

	rootContext := sys.Root

	<-time.After(3 * time.Second)

	props := actor.PropsFromFunc(
		func(ctx actor.Context) {
			fmt.Printf("message in actor: %s, msg type: %T, msg: %q\n", ctx.Self().GetId(), ctx.Message(), ctx.Message())
			switch msg := ctx.Message().(type) {
			case *actor.Started:
				logs.LogInfo.Printf("started actor %q", ctx.Self().GetId())
				if err := func() (err error) {
					var jwtConf *renatsio.JwtConf
					if len(keycloakurl) > 0 {
						jwtConf = &renatsio.JwtConf{
							KeycloakURL:  keycloakurl,
							Realm:        realm,
							User:         hostname,
							Pass:         hostname,
							ClientID:     clientid,
							ClientSecret: clientSecret,
						}
					}
					propsNatsio := actor.PropsFromFunc(renatsio.NewClientNatsio(remoteBroker, jwtConf, test).Receive)

					pidNatsio, err = ctx.SpawnNamed(propsNatsio, renatsio.INSTANCE_ID)
					if err != nil {
						return fmt.Errorf("create natsio actor error: %s", err)
					}

					if err := pubsub.Subscribe(gwiot.SUBS_TOPIC, ctx.Self(), Discover); err != nil {
						return fmt.Errorf("create discovery subscription pubsub error: %s", err)
					}

					return nil
				}(); err != nil {
					time.Sleep(3 * time.Second)
					logs.LogError.Panic(err)
				}
			case *gwiotmsg.Discovery:
				if len(msg.GetReply()) <= 0 {
					break
				}
				if pidNatsio == nil {
					break
				}
				remote.GetRemote(ctx.ActorSystem())

				host, port, err := ctx.ActorSystem().GetHostPort()
				if err != nil {
					logs.LogWarn.Printf("get remote address error: %s", err)
					break
				}
				mss := &gwiotmsg.DiscoveryResponse{
					Kind: gwiot.KIND_NAME,
					Host: host,
					Port: int32(port),
				}
				data, err := json.Marshal(mss)
				if err != nil {
					logs.LogWarn.Printf("get remote address error: %s", err)
					break
				}
				fmt.Printf("publish (%s): %q\n", msg.GetReply(), data)
				pubsub.Publish(msg.GetReply(), data)
			case *gwiotmsg.Subscribe:
			}
		})

	pidMain, err := rootContext.SpawnNamed(props, ACTOR_INTSANCE_ID)
	if err != nil {
		logs.LogError.Printf("create main actor error: %s", err)
	}

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)
	signal.Notify(finish, os.Interrupt)
	for range finish {
		if pidMain != nil {
			log.Print("finish Console")
			rootContext.Poison(pidMain)
		}
		logs.LogInfo.Printf("Stoping gwiot")
		time.Sleep(300 * time.Millisecond)
		log.Print("Finish")
		time.Sleep(600 * time.Millisecond)
		return
	}
}
