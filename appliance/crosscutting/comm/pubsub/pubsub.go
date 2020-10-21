package pubsub

import (
	"fmt"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-modem/appliance/business/messages"
	"github.com/dumacp/go-modem/appliance/crosscutting/logs"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	clientID       = "ignition"
	TopicAppliance = "appliance/modem"
	TopicEvents    = "EVENTS/modem"
	//TopicEvents           = TopicAppliance + "/events"
	TopicStart            = TopicAppliance + "/START"
	TopicRestart          = TopicAppliance + "/RESTART"
	TopicStop             = TopicAppliance + "/STOP"
	TopicStatus           = TopicAppliance + "/STATUS"
	TopicRequestInfoState = TopicAppliance + "/RequestInfoState"
)

// //Gateway interface
// type Gateway interface {
// 	Receive(ctx actor.Context)
// 	// Publish(topic string, msg []byte)
// }

type pubsubActor struct {
	ctx actor.Context
	// behavior      actor.Behavior
	state         messages.StatusResponse_StateType
	client        mqtt.Client
	mux           sync.Mutex
	subscriptions map[string]*subscribeMSG
}

var instance *pubsubActor
var once sync.Once

//getInstance create pubsub Gateway
func getInstance() *pubsubActor {

	once.Do(func() {
		instance = &pubsubActor{}
		instance.mux = sync.Mutex{}
		instance.subscriptions = make(map[string]*subscribeMSG)
		ctx := actor.EmptyRootContext
		props := actor.PropsFromFunc(instance.Receive)
		_, err := ctx.SpawnNamed(props, "pubsub-actor")
		if err != nil {
			logs.LogError.Panic(err)
		}
	})
	return instance
}

//Init init pubsub instance
func Init() error {
	defer time.Sleep(3 * time.Second)
	if getInstance() == nil {
		return fmt.Errorf("error instance")
	}
	return nil
}

type publishMSG struct {
	topic string
	msg   []byte
}

type subscribeMSG struct {
	pid   *actor.PID
	parse func([]byte) interface{}
}

//Publish function to publish messages in pubsub gateway
func Publish(topic string, msg []byte) {
	getInstance().ctx.Send(instance.ctx.Self(), &publishMSG{topic: topic, msg: msg})
}

//Subscribe subscribe to topics
func Subscribe(topic string, pid *actor.PID, parse func([]byte) interface{}) error {
	instance := getInstance()
	subs := &subscribeMSG{pid: pid, parse: parse}
	instance.mux.Lock()
	instance.subscriptions[topic] = subs
	instance.mux.Unlock()
	if !instance.client.IsConnected() {
		// instance.ctx.PoisonFuture(instance.ctx.Self()).Wait()
		return fmt.Errorf("pubsub is not connected")
	}
	logs.LogBuild.Printf("subscription in topic -> %q -> %#v", topic, subs)
	instance.subscribe(topic, subs)
	return nil
}

func (ps *pubsubActor) subscribe(topic string, subs *subscribeMSG) error {
	handler := func(client mqtt.Client, m mqtt.Message) {
		logs.LogBuild.Printf("local topic -> %q", m.Topic())
		// logs.LogBuild.Printf("local payload - > %s", m.Payload())
		m.Ack()
		msg := subs.parse(m.Payload())
		// logs.LogBuild.Printf("parse payload-> %s", msg)
		ps.ctx.Send(subs.pid, msg)
	}
	if tk := instance.client.Subscribe(topic, 1, handler); !tk.WaitTimeout(3 * time.Second) {
		if err := tk.Error(); err != nil {
			return err
		}
	}
	return nil
}

//Receive function
func (ps *pubsubActor) Receive(ctx actor.Context) {
	ps.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
		ps.client = client()
		if err := connect(ps.client); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}
		for k, v := range ps.subscriptions {
			ps.subscribe(k, v)
		}
	case *publishMSG:
		tk := ps.client.Publish(msg.topic, 0, false, msg.msg)
		if !tk.WaitTimeout(3 * time.Second) {
			if tk.Error() != nil {
				logs.LogError.Printf("end error: %s, with messages -> %v", msg)
			} else {
				logs.LogError.Printf("timeout error with message -> %v", msg)
			}
		}
	case *actor.Stopping:
		ps.client.Disconnect(600)
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

// func (ps *pubsubActor) Started(ctx actor.Context) {
// 	ps.ctx = ctx
// 	switch ctx.Message().(type) {
// 	}
// }

// func (ps *pubsubActor) Stopped(ctx actor.Context) {
// }

func client() mqtt.Client {
	opt := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1883")
	opt.SetAutoReconnect(true)
	opt.SetClientID(fmt.Sprintf("%s-%d", clientID, time.Now().Unix()))
	opt.SetKeepAlive(30 * time.Second)
	opt.SetConnectRetryInterval(10 * time.Second)
	client := mqtt.NewClient(opt)
	return client
}

func connect(c mqtt.Client) error {
	tk := c.Connect()
	if !tk.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connect wait error")
	}
	if err := tk.Error(); err != nil {
		return err
	}
	return nil
}
