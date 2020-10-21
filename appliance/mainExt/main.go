package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/dumacp/go-modem/appliance/business/controlext"
	"github.com/dumacp/go-modem/appliance/business/messages"
	"github.com/dumacp/go-modem/appliance/business/nmea"
	"github.com/dumacp/go-modem/appliance/business/nmeatcp"
)

var debug bool
var mqtt bool
var port int
var ipTest string
var apnConn string

var timeout int
var baudRate int
var portNmea string
var distanceMin int

const (
	ipTestInitial = "8.8.8.8"
)

func init() {
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.BoolVar(&mqtt, "mqtt", false, "[DEPRECATED] send messages to local broker.")
	flag.IntVar(&port, "port", 8082, "port actor in remote mode")
	flag.IntVar(&timeout, "timeout", 30, "timeout to capture frames.")
	flag.IntVar(&baudRate, "baudRate", 115200, "baud rate to capture nmea's frames.")
	flag.StringVar(&portNmea, "portNmea", "/dev/ttyUSB1", "device serial to read.")
	flag.IntVar(&distanceMin, "distance", 30, "minimun distance traveled before to send")
}

func main() {

	flag.Parse()
	initLogs(debug)

	remote.Start(fmt.Sprintf("127.0.0.1:%v", port),
		remote.WithAdvertisedAddress(fmt.Sprintf("localhost:%v", port)))

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

	pidCheck := &actor.PID{}

	props := actor.PropsFromFunc(func(c actor.Context) {
		switch msg := c.Message().(type) {
		case *actor.Started:
			propsNmea := actor.PropsFromFunc(nmeatcp.NewNmeaActor(
				debug,
				portNmea,
				baudRate,
				timeout,
				distanceMin,
			).Receive)
			propsCheck := actor.PropsFromFunc(controlext.NewCheckModemActor(debug).Receive)
			pidNmea, err := c.SpawnNamed(propsNmea, "nmeaGPS")
			if err != nil {
				errlog.Panic(err)
			}
			pidCheck, err = c.SpawnNamed(propsCheck, "checkmodem")
			if err != nil {
				errlog.Panic(err)
			}
			c.Watch(pidNmea)
			c.Watch(pidCheck)
			c.Send(pidNmea, &nmea.AddressModem{
				Addr: pidCheck.GetAddress(),
				ID:   pidCheck.GetId(),
			})
			// c.Send(pidCheck, &messages.ModemCheck{
			// 	Addr: "8.8.8.8",
			// 	Apn:  "",
			// })

		case *actor.Terminated:
			errlog.Printf("actor terminated: %s", msg.Who.GetId())
		}
	})

	_, err := rootContext.SpawnNamed(props, "modem")
	if err != nil {
		errlog.Fatalln(err)
	}

	// funcModemAddr := func(msg *messRecepcionist.SubscribeAdvertising) {
	// 	log.Printf("receive Advertising: %v\n", msg)
	// 	msgAddr := &messages.ModemAddr{
	// 		Addr: msg.Addr,
	// 		Id:   msg.Id,
	// 	}
	// 	rootContext.Send(pid, msgAddr)
	// }

	// serviceRecp, err := receptionist.NewReceptionist("127.0.0.1:8081")
	// if err != nil {
	// 	log.Println(err)
	// } else {
	// 	log.Println("START Receptionist")
	// 	serviceRecp.Register("netmodem", funcModemAddr, []string{"ctrlmodem"})
	// }

	// time.Sleep(2 * time.Second)
	// rootContext.Send(pid, &messages.ModemCheck{
	// 	Addr: "127.0.0.1",
	// 	Apn:  "",
	// })
	// rootContext.Send(pid, &messages.TestIP{Addr: "127.0.0.1"})

	testIP, err1 := getTestIP()
	apn, err2 := getAPN()
	if err1 == nil || err2 == nil {
		rootContext.Send(pidCheck, &messages.ModemCheck{
			Addr: testIP,
			Apn:  apn,
		})
	}

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for {
		select {
		case v := <-finish:
			errlog.Println(v)
			return
		}
	}

}

func getTestIP() (string, error) {
	testIP := os.Getenv("TEST_IP")
	if len(testIP) <= 0 {
		return "", fmt.Errorf("TEST_IP not found")
	}
	if strings.Contains(ipTest, testIP) {
		return "", fmt.Errorf("already testIP")
	}
	ipTest = testIP
	return testIP, nil
}

func getAPN() (string, error) {
	apn := os.Getenv("APN")
	if len(apn) <= 0 {
		return "", fmt.Errorf("APN not found")
	}
	if strings.Contains(apn, apnConn) {
		return "", fmt.Errorf("already APN")
	}
	apnConn = apn
	return apn, nil
}
