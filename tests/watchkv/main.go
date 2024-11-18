package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-gwiot/pkg/gwiot"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {

	sys := actor.NewActorSystem()

	ctxroot := sys.Root

	rconfig := remote.Configure("127.0.0.1", 8199)
	r := remote.NewRemote(sys, rconfig)
	r.Start()

	// client mqtt
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID("test-kv")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalln(token.Error())
	}

	// funcSubscribe
	funcSubs := func(topicReply string, pid *actor.PID, parse func([]byte) interface{}) error {
		// mqtt subscriptions
		client.Subscribe(topicReply, 0, func(client mqtt.Client, msg mqtt.Message) {
			if msg.Payload() == nil {
				return
			}
			if ev := parse(msg.Payload()); ev != nil {
				ctxroot.Send(pid, ev)
			}
		})
		return nil
	}

	// funcPublish
	funcPub := func(topicDiscovery string, msg []byte) {
		// mqtt publish
		client.Publish(topicDiscovery, 0, false, msg)
	}

	// create child actor
	actorDiscv := gwiot.NewDiscoveryActor("testTopicReply", funcSubs, funcPub)
	propsDiscv := actor.PropsFromProducer(func() actor.Actor {
		return actorDiscv
	})

	// nast actor
	remoteAddress := ""
	kind := ""
	type connectMsg struct{}
	var pidChild *actor.PID
	propsNats := actor.PropsFromFunc(func(ctx actor.Context) {
		fmt.Printf("message in actor main Actor (%s): msg type: %T, msg: %q\n", ctx.Self().GetId(), ctx.Message(), ctx.Message())
		switch msg := ctx.Message().(type) {
		case *actor.Started:
			pidDiscv, err := ctx.SpawnNamed(propsDiscv, "testsDiscovery")
			if err != nil {
				log.Fatalf("error spawn actor: %s", err)
			}
			ctx.Send(pidDiscv, &gwiotmsg.Discovery{})
		case *actor.Stopping:
			if pidChild != nil {
				ctx.PoisonFuture(pidChild).Wait()
			}
		case *gwiotmsg.DiscoveryResponse:
			log.Printf("response: %v", ctx.Message())
			remoteAddress = fmt.Sprintf("%s:%d", msg.GetHost(), msg.GetPort())
			kind = msg.GetKind()
			ctx.Send(ctx.Self(), &connectMsg{})
		case *connectMsg:
			if len(remoteAddress) <= 0 {
				break
			}
			if kind == "" {
				break
			}
			fmt.Println("connectMsg 1")
			r := remote.GetRemote(ctx.ActorSystem())
			fmt.Println("connectMsg 2")
			pidResponse, err := r.SpawnNamed(remoteAddress, "nast-test-clietn", gwiot.KIND_NAME, 3*time.Second)
			if err != nil {
				logs.LogWarn.Printf("remote activation nast error: %s", err)
				remoteAddress = ""
				break
			}
			fmt.Println("connectMsg 3")

			pidChild = pidResponse.GetPid()
			ctx.Watch(pidChild)
			ctx.Request(pidChild, &gwiotmsg.WatchKeyValue{
				Bucket: "Status",
				Key:    "test",
			})
		case *gwiotmsg.WatchMessage:
			log.Printf("watch message: %v", msg)
		case *actor.Terminated:
			fmt.Printf("terminated: %v\n", msg)
		}
	})

	pidNats, err := ctxroot.SpawnNamed(propsNats, "testsNats")
	if err != nil {
		log.Fatalf("error spawn actor: %s", err)
	}

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	<-finish
	if pidNats != nil {
		log.Print("finish")
		ctxroot.PoisonFuture(pidNats).Wait()
	}
	logs.LogInfo.Printf("Stoping gwiot")

}
