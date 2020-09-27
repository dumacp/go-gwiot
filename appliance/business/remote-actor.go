package business

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/persistence"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	"github.com/dumacp/utils"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/oauth2"
)

const (
	remoteMqttBrokerURL = "wss://fleet-mqtt.nebulae.com.co/mqtt"
	remoteQueueEvents   = "/devices/%s/events"
	protocolVersion     = 4  // corresponds to mqtt 3.1.1
	minimumBackoffTime  = 1  // initial backoff time in seconds
	maximumBackoffTime  = 32 // maximum backoff time in seconds
)

// //Gateway interface
// type Gateway interface {
// 	Receive(ctx actor.Context)
// 	// Publish(topic string, msg []byte)
// }

//RemoteActor remote actor
type RemoteActor struct {
	persistence.Mixin
	lastMSG *messages.RemoteMSG
	// lastBackMSG *messages.RemoteMSG
	ctx         actor.Context
	pidKeycloak *actor.PID
	client      mqtt.Client
	token       *oauth2.Token
	queueMSG    []*messages.RemoteMSG
	// subscriptions map[string]*actor.PID
}

func (ps *RemoteActor) putMSG(msg *messages.RemoteMSG) {
	if ps.queueMSG != nil && len(ps.queueMSG) > 100 {
		lastMSGs := ps.queueMSG[50:]
		ps.queueMSG = lastMSGs
	}
	ps.queueMSG = append(ps.queueMSG, msg)
}

func (ps *RemoteActor) deleteMSG(n uint) {
	if ps.queueMSG != nil && len(ps.queueMSG) >= int(n) {
		lastMSGs := ps.queueMSG[n:]
		ps.queueMSG = lastMSGs
	}
}

//NewRemote new remote actor
func NewRemote() *RemoteActor {
	r := &RemoteActor{}
	r.queueMSG = make([]*messages.RemoteMSG, 0)
	return r
}

type reconnectRemote struct{}
type resendMSG struct{}

//Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())

		request := ctx.RequestFuture(ctx.Parent(), &messages.KeycloakAddressRequest{}, 3*time.Second)
		if err := request.Wait(); err != nil {
			logs.LogError.Printf("request keycloak actor address error -> %s", err)
			break
		}

		res, err := request.Result()
		if err != nil {
			logs.LogError.Printf("request keycloak actor address error -> %s", err)
			break
		}

		var ok bool
		if ps.pidKeycloak, ok = res.(*actor.PID); !ok {
			logs.LogError.Printf("request keycloak actor address error -> %s", err)
			break
		}
		logs.LogBuild.Printf("remote keycloak -> %v", ps.pidKeycloak)

		if ps.client != nil {
			ps.client.Disconnect(300)
		}
		if ps.token == nil || ps.token.Expiry.Before(time.Now()) {
			ctx.Request(ps.pidKeycloak, &messages.TokenRequest{})
			time.Sleep(6 * time.Second)
			break
		}
		ps.client = client(ps.token)
		if err := connect(ps.client); err != nil {
			time.Sleep(6 * time.Second)
			logs.LogError.Panicf("connect error -> %s", err)
		}

	case *oauth2.Token:
		ps.token = msg
		logs.LogBuild.Printf("new jwt token -> %+v", msg)
		if ps.client != nil {
			ps.client.Disconnect(300)
		}
		ps.client = client(ps.token)
		if err := connect(ps.client); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicf("connect error -> %s", err)
		}

		if len(ps.queueMSG) > 0 {
			ctx.Send(ctx.Self(), &resendMSG{})
		}

	case *resendMSG:
		if len(ps.queueMSG) <= 0 {
			break
		}
		topic := fmt.Sprintf(remoteQueueEvents, Hostname())

		for i, msg := range ps.queueMSG {
			if err := sendMSG(ps.client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				ctx.Send(ctx.Self(), &reconnectRemote{})
				break
			}
			ps.deleteMSG(1)
			if i > 32 {
				ps.PersistSnapshot(&messages.RemoteSnapshot{
					TimeStamp: time.Now().Unix(),
					LastMSG:   ps.queueMSG,
				})
				break
			}
		}
	case *messages.RemoteMSG:
		logs.LogBuild.Printf("remote message -> %s", msg.GetData())
		if ps.pidKeycloak == nil {
			ctx.Send(ctx.Self(), &actor.Started{})
			ps.queueMSG = append(ps.queueMSG, msg)
			ps.PersistReceive(msg)
			break
		}
		if ps.token == nil {
			ctx.Request(ps.pidKeycloak, &messages.TokenRequest{})
			time.Sleep(6 * time.Second)
			ps.queueMSG = append(ps.queueMSG, msg)
			ps.PersistReceive(msg)
			break
		}
		if ps.token.Expiry.Before(time.Now()) {
			logs.LogBuild.Printf("before jwt token -> %+v", ps.token)
			ctx.Request(ps.pidKeycloak, &messages.TokenRequest{})
			time.Sleep(6 * time.Second)
			ps.queueMSG = append(ps.queueMSG, msg)
			ps.PersistReceive(msg)
			break
		}

		topic := fmt.Sprintf(remoteQueueEvents, Hostname())
		ps.lastMSG = msg

		go func(ctx actor.Context, client mqtt.Client, topic string, msg *messages.RemoteMSG) {
			if err := sendMSG(client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				ctx.Send(ctx.Self(), &reconnectRemote{})
				ps.queueMSG = append(ps.queueMSG, msg)
				ps.PersistReceive(msg)
			}
		}(ctx, ps.client, topic, msg)

	case *reconnectRemote:
		panic(msg)
	case *messages.RemoteSnapshot:
		logs.LogInfo.Printf("recover snapshot -> %v, %s", time.Unix(msg.TimeStamp, 0), msg.GetLastMSG())
		if msg.GetLastMSG() == nil {
			break
		}
		topic := fmt.Sprintf(remoteQueueEvents, Hostname())
		if ps.queueMSG == nil {
			ps.queueMSG = msg.GetLastMSG()
		} else {
			ps.queueMSG = append(ps.queueMSG, msg.GetLastMSG()...)
		}
		for i, msg := range ps.queueMSG {
			if err := sendMSG(ps.client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				ctx.Send(ctx.Self(), &reconnectRemote{})
				break
			}
			ps.deleteMSG(1)
			if i > 32 {
				ps.PersistSnapshot(&messages.RemoteSnapshot{
					TimeStamp: time.Now().Unix(),
					LastMSG:   ps.queueMSG,
				})
				break
			}
		}
	case *persistence.ReplayComplete:
		if ps.queueMSG != nil && len(ps.queueMSG) > 32 {
			ps.PersistSnapshot(&messages.RemoteSnapshot{
				TimeStamp: time.Now().Unix(),
				LastMSG:   ps.queueMSG[:32],
			})
			break
		}
	case *persistence.RequestSnapshot:
		if ps.queueMSG != nil && len(ps.queueMSG) > 32 {
			ps.PersistSnapshot(&messages.RemoteSnapshot{
				TimeStamp: time.Now().Unix(),
				LastMSG:   ps.queueMSG[:32],
			})
			break
		}

		ps.PersistSnapshot(ps.lastMSG)
	case *actor.Stopping:
		ps.client.Disconnect(600)
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

func sendMSG(client mqtt.Client, topic string, msg []byte) error {
	tk := client.Publish(topic, 1, false, msg)

	for range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if tk.WaitTimeout(1000 * time.Millisecond) {
			return nil
		}
		if tk.Error() != nil {
			// logs.LogError.Printf("publish error -> %s, message -> %q", tk.Error(), msg.GetData())
			return tk.Error()
		}
	}

	return fmt.Errorf("message dont send")
}

func (ps *RemoteActor) tickreSend() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if len(ps.queueMSG) > 0 {
			ps.ctx.Send(ps.ctx.Self(), &resendMSG{})
		} else {

			logs.LogBuild.Printf("new snapshot -> %v", ps.lastMSG)
			ps.PersistSnapshot(&messages.RemoteSnapshot{
				TimeStamp: time.Now().Unix(),
				LastMSG:   make([]*messages.RemoteMSG, 0),
			})
			// ps.lastBackMSG = ps.lastMSG

		}
	}
}

func client(tk *oauth2.Token) mqtt.Client {
	_, tlsconfig := utils.LoadLocalCert(localCertDir)
	opt := mqtt.NewClientOptions()
	opt.AddBroker(remoteMqttBrokerURL)
	opt.SetConnectRetry(true)
	opt.SetConnectRetryInterval(10 * time.Second)
	opt.SetPassword(tk.AccessToken)
	opt.SetUsername("unused")
	opt.SetProtocolVersion(protocolVersion)
	opt.SetTLSConfig(tlsconfig)

	// opt.SetAutoReconnect(true)
	opt.SetClientID(fmt.Sprintf("%s-%d", Hostname(), time.Now().Unix()))
	opt.SetKeepAlive(30 * time.Second)
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
