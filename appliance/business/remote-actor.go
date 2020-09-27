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
	lastMSG     *messages.RemoteMSG
	ctx         actor.Context
	pidKeycloak *actor.PID
	client      mqtt.Client
	token       *oauth2.Token
	queueMSG    []*messages.RemoteMSG
	// subscriptions map[string]*actor.PID
}

func (ps *RemoteActor) putMSG(msg *messages.RemoteMSG) {
	if ps.queueMSG != nil && len(ps.queueMSG) > 200 {
		lastMSGs := ps.queueMSG[100:]
		ps.queueMSG = lastMSGs
	}
	ps.queueMSG = append(ps.queueMSG, msg)
}

func (ps *RemoteActor) deleteMSG() {
	if ps.queueMSG != nil && len(ps.queueMSG) > 0 {
		lastMSGs := ps.queueMSG[1:]
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
		for _, msg := range ps.queueMSG {
			if err := sendMSG(ps.client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				ctx.Send(ctx.Self(), &reconnectRemote{})
				break
			}
			ps.deleteMSG()
		}
	case *messages.RemoteMSG:
		logs.LogBuild.Printf("remote message -> %s", msg.GetData())
		if ps.pidKeycloak == nil {
			ctx.Send(ctx.Self(), &actor.Started{})
			ps.queueMSG = append(ps.queueMSG, msg)
			break
		}
		if ps.token == nil {
			ctx.Request(ps.pidKeycloak, &messages.TokenRequest{})
			time.Sleep(6 * time.Second)
			ps.queueMSG = append(ps.queueMSG, msg)
			break
		}
		if ps.token != nil {
			if ps.token.Expiry.Before(time.Now()) {
				logs.LogBuild.Printf("before jwt token -> %+v", ps.token)
				ctx.Request(ps.pidKeycloak, &messages.TokenRequest{})
				time.Sleep(6 * time.Second)
				ps.queueMSG = append(ps.queueMSG, msg)
				break
			}
		}
		topic := fmt.Sprintf(remoteQueueEvents, Hostname())
		ps.lastMSG = msg

		go func(ctx actor.Context, client mqtt.Client, topic string, msg *messages.RemoteMSG) {
			if err := sendMSG(client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				ctx.Send(ctx.Self(), &reconnectRemote{})
				ps.PersistReceive(msg)
			}
		}(ctx, ps.client, topic, msg)

		// go func() {
		// 	tk := ps.client.Publish(topic, 1, false, msg.GetData())

		// 	for range []int{0, 1, 2} {
		// 		if tk.WaitTimeout(600 * time.Second) {
		// 			return
		// 		}
		// 		if tk.Error() != nil {
		// 			logs.LogError.Printf("publish error -> %s, message -> %q", tk.Error(), msg.GetData())
		// 			ctx.Send(ctx.Self(), &reconnectRemote{})
		// 			break
		// 		}
		// 	}
		// 	ps.PersistReceive(msg)
		// }()
	case *reconnectRemote:
		panic(msg)
	case *persistence.ReplayComplete:
		if ps.lastMSG != nil {
			ps.PersistSnapshot(ps.lastMSG)
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

// func (ps *RemoteActor) Started(ctx actor.Context) {
// 	ps.ctx = ctx
// 	switch ctx.Message().(type) {
// 	}
// }

// func (ps *RemoteActor) Stopped(ctx actor.Context) {
// }

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

//InitConnection func to init remote connection
// func (r *RemoteConn) InitConnection() (mqtt.Client, error) {

// 	r.tokenSubscribe = nil

// 	if r.ts == nil {

// 		keyc, err := keycloak.NewConfig(r.ctx, r.configKeyc)
// 		if err != nil {
// 			return nil, err
// 		}
// 		tk, err := keyc.TokenRequest(r.ctx, r.username, r.username)
// 		if err != nil {
// 			return nil, err
// 		}
// 		r.ts = keyc.TokenSource(r.ctx, tk)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	//clientID := fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s", projectID, region, registryID, gatewayID)
// 	opts := mqtt.NewClientOptions()
// 	opts.AddBroker(remoteMqttBrokerURL)
// 	opts.SetClientID(r.clientID)
// 	tk, err := r.ts.Token()
// 	if err != nil {
// 		r.ts = nil
// 		return nil, err
// 	}
// 	opts.SetPassword(tk.AccessToken)
// 	log.Println(tk.AccessToken)
// 	opts.SetUsername("unused")
// 	opts.SetCleanSession(false)
// 	opts.SetProtocolVersion(protocolVersion)
// 	opts.SetOnConnectHandler(OnConnect(r.Conn))
// 	opts.SetTLSConfig(r.tlsConfig)
// 	if len(r.subscriptions) > 0 {
// 		log.Printf("OnConnection subscribed!!")
// 		opts.SetDefaultPublishHandler(OnMessage(r.Conn))
// 	}
// 	opts.SetConnectionLostHandler(OnDisconnect(r.Conn))

// 	//TODO: ???
// 	// opts.SetAutoReconnect(true)

// 	client := mqtt.NewClient(opts)
// 	if token := client.Connect(); token.Wait() && token.Error() != nil {
// 		log.Println("Failed to connect client")
// 		return nil, token.Error()
// 	}
// 	if len(r.subscriptions) > 0 {
// 		for range []int{1, 2, 3, 4, 5} {
// 			time.Sleep(1 * time.Second)
// 			if r.tokenSubscribe == nil {
// 				continue
// 			}
// 			break
// 		}
// 	}
// 	return client, nil
// }
