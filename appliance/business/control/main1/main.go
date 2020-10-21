package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/dumacp/go-modem/appliance/business/control"
	"github.com/dumacp/go-modem/appliance/business/messages"
	"github.com/dumacp/omvz/receptionist"
	messRecepcionist "github.com/dumacp/omvz/receptionist/messages"
)

// var debug bool

func main() {

	remote.Start("127.0.0.1:8082", remote.WithAdvertisedAddress("localhost:8082"))
	rootContext := actor.EmptyRootContext

	// var msgChan chan string
	// pub, err := pubsub.NewConnection("go-netmodem")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer pub.Disconnect()
	// msgChan = make(chan string)
	// go pub.Publish("SYSTEM/ACTOR/modem", msgChan)
	// go func() {
	// 	for v := range pub.Err {
	// 		log.Println(v)
	// 	}
	// }()

	// pubsubMiddleware := func(next actor.SenderFunc) actor.SenderFunc {
	// 	return func(ctx actor.SenderContext, target *actor.PID, envelope *actor.MessageEnvelope) {
	// 		switch msg := envelope.Message.(type) {
	// 		case *messages.ModemReset:
	// 			msgS, err := json.Marshal(msg)
	// 			if err != nil {
	// 				break
	// 			}
	// 			select {
	// 			case msgChan <- string(msgS):
	// 			case <-time.After(3 * time.Second):
	// 			}
	// 		default:
	// 			next(ctx, target, envelope)
	// 		}
	// 	}
	// }
	props := actor.PropsFromProducer(control.NewCheckModemActor)
	pid, err := rootContext.SpawnNamed(props, "netmodem")
	if err != nil {
		log.Fatalln(err)
	}

	funcModemAddr := func(msg *messRecepcionist.SubscribeAdvertising) {
		log.Printf("receive Advertising: %v\n", msg)
		msgAddr := &messages.ModemAddr{
			Addr: msg.Addr,
			Id:   msg.Id,
		}
		rootContext.Send(pid, msgAddr)
	}

	serviceRecp, err := receptionist.NewReceptionist("127.0.0.1:8081")
	if err != nil {
		log.Println(err)
	} else {
		log.Println("START Receptionist")
		serviceRecp.Register("netmodem", funcModemAddr, []string{"ctrlmodem"})
	}

	time.Sleep(2 * time.Second)
	rootContext.Send(pid, &messages.ModemCheck{
		Addr: "127.0.0.1",
		Apn:  "",
	})
	// rootContext.Send(pid, &messages.TestIP{Addr: "127.0.0.1"})
	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)
	<-finish
}
