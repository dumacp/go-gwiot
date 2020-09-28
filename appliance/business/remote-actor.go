package business

import (
	"fmt"
	"sync"
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
	lastBackMSG *messages.RemoteSnapshot
	ctx         actor.Context
	pidKeycloak *actor.PID
	client      mqtt.Client
	token       *oauth2.Token
	queueMSG    *queue
	// subscriptions map[string]*actor.PID
}

type queue struct {
	msgs []*messages.RemoteMSG
	mux  sync.Mutex
}

func newQueue() *queue {
	return &queue{
		msgs: make([]*messages.RemoteMSG, 0),
		mux:  sync.Mutex{},
	}
}

func (q *queue) put(msg *messages.RemoteMSG) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.msgs = append(q.msgs, msg)
}

func (q *queue) pop() *messages.RemoteMSG {
	q.mux.Lock()
	defer q.mux.Unlock()
	v := q.msgs[len(q.msgs)-1]
	q.msgs = q.msgs[:len(q.msgs)-1]
	return v
}

func (q *queue) deletedata() {
	q.mux.Lock()
	defer q.mux.Unlock()
	if q.msgs != nil {
		q.msgs = make([]*messages.RemoteMSG, 0)
	}
}

func (q *queue) truncate(n uint) {
	q.mux.Lock()
	defer q.mux.Unlock()
	if q.msgs != nil && len(q.msgs) >= int(n) {
		q.msgs = q.msgs[n:]
	}
}

func (q *queue) getdata() []*messages.RemoteMSG {
	q.mux.Lock()
	defer q.mux.Unlock()
	return q.msgs
}

func (q *queue) lendata() int {
	q.mux.Lock()
	defer q.mux.Unlock()
	return len(q.msgs)
}

func (q *queue) last() *messages.RemoteMSG {
	q.mux.Lock()
	defer q.mux.Unlock()
	return q.msgs[len(q.msgs)-1]
}

//NewRemote new remote actor
func NewRemote() *RemoteActor {
	r := &RemoteActor{}
	r.queueMSG = newQueue()
	return r
}

type reconnectRemote struct{}
type resendMSG struct{}

//Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())

		ps.queueMSG.deletedata()

		request := ctx.RequestFuture(ctx.Parent(), &messages.KeycloakAddressRequest{}, 6*time.Second)
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
			req := ctx.RequestFuture(ps.pidKeycloak, &messages.TokenRequest{}, 6*time.Second)
			if err := req.Wait(); err != nil {
				logs.LogError.Printf("request JWT error -> %s", err)
				break
			}
			res, err := req.Result()
			if err != nil {
				logs.LogError.Printf("request JWT error -> %s", err)
				break
			}
			if ps.token, ok = res.(*oauth2.Token); !ok {
				logs.LogError.Printf("request JWT error -> %s", err)
				break
			}
			logs.LogBuild.Printf("new jwt -> %v", ps.token)

		}
		ps.client = client(ps.token)
		if err := connect(ps.client); err != nil {
			time.Sleep(6 * time.Second)
			logs.LogError.Printf("connect error -> %s", err)
			break
		}
	case *reconnectRemote:
		panic(msg)
	case *oauth2.Token:
		ps.token = msg
		logs.LogBuild.Printf("new jwt token -> %+v", msg)
		if ps.client != nil {
			ps.client.Disconnect(300)
		}
		ps.client = client(ps.token)
		if err := connect(ps.client); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Printf("connect error -> %s", err)
			break
		}

		if ps.queueMSG.lendata() > 0 {
			ctx.Send(ctx.Self(), &resendMSG{})
		}

	case *resendMSG:
		if ps.queueMSG.lendata() <= 0 {
			break
		}
		topic := fmt.Sprintf(remoteQueueEvents, Hostname())

		lenBefore := ps.queueMSG.lendata()
		snap := &messages.RemoteSnapshot{
			TimeStamp: time.Now().Unix(),
			LastMSG:   ps.queueMSG.getdata()[0],
		}
		for _, msg := range ps.queueMSG.getdata() {
			logs.LogBuild.Printf("resend: %v, remote message -> %s", true, msg.GetData())
			if ok, err := sendMSG(ps.client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %q", err, msg.GetData())
				if ok {
					if msg.Retry < 3 {
						ctx.Send(ctx.Self(), &reconnectRemote{})
					} else {
						logs.LogError.Printf("publish error, last retry -> %s, message -> %q", err, msg.GetData())
					}
				}
			}
			ps.queueMSG.truncate(1)
		}
		if ps.queueMSG.lendata() > 0 && ps.queueMSG.lendata() < lenBefore {
			snap.LastMSG = ps.queueMSG.getdata()[0]
		}
		if ps.queueMSG.lendata() < lenBefore {
			ps.PersistSnapshot(snap)
			for _, v := range ps.queueMSG.getdata() {
				ps.PersistReceive(v)
			}
		}
	case *messages.RemoteMSG:
		logs.LogBuild.Printf("recovering: %v, remote message -> %s", ps.Recovering(), msg.GetData())

		if ps.pidKeycloak == nil {
			ctx.Send(ctx.Self(), &reconnectRemote{})
			if !ps.Recovering() {
				ps.PersistReceive(msg)
			}
			break
		}
		if ps.token == nil {
			ctx.Send(ctx.Self(), &reconnectRemote{})
			if !ps.Recovering() {
				ps.PersistReceive(msg)
			}
			break
		}
		if ps.token.Expiry.Before(time.Now()) {
			logs.LogBuild.Printf("before jwt token -> %+v", ps.token)
			ctx.Send(ctx.Self(), &reconnectRemote{})
			if !ps.Recovering() {
				ps.PersistReceive(msg)
			}
			break
		}

		topic := fmt.Sprintf(remoteQueueEvents, Hostname())
		// ps.lastMSG = msg

		go func(ctx actor.Context, client mqtt.Client, topic string, recovery bool, msg *messages.RemoteMSG) {
			if ok, err := sendMSG(client, topic, msg.GetData()); err != nil {
				logs.LogError.Printf("publish error -> %s, message -> %s", err, msg.GetData())
				if !recovery && ok && msg.Retry < 3 {
					ctx.Send(ctx.Self(), &reconnectRemote{})
				}
				if msg.Retry >= 3 {
					logs.LogError.Printf("publish error, last retry -> %s, message -> %q", err, msg.GetData())
				}
				if recovery && msg.Retry < 3 {
					msg.Retry++
					ps.queueMSG.put(msg)
				}
				if !recovery {
					ps.PersistReceive(msg)
				}
			}
		}(ctx, ps.client, topic, ps.Recovering(), msg)

	case *messages.RemoteSnapshot:
		logs.LogInfo.Printf("recover snapshot -> %v, last messages -> %s", time.Unix(msg.TimeStamp, 0), msg.GetLastMSG())
		ps.lastBackMSG = msg

	case *persistence.ReplayComplete:
		logs.LogBuild.Println("Replay Complete")

		if ps.queueMSG.lendata() > 0 && ps.lastBackMSG != nil {
			if ps.queueMSG.getdata()[0].TimeStamp <= ps.lastBackMSG.LastMSG.TimeStamp {
				break
			}
		}
		if ps.queueMSG.lendata() > 0 {
			ps.lastMSG = ps.queueMSG.getdata()[0]
		}

		logs.LogBuild.Println("Replay Complete 2")
		if ps.queueMSG.lendata() <= 0 {
			ps.PersistSnapshot(&messages.RemoteSnapshot{
				TimeStamp: time.Now().Unix(),
				LastMSG:   ps.lastMSG,
			})
			break
		}
		logs.LogBuild.Println("Replay Complete 3")
		if ps.queueMSG.lendata() > 0 {
			ps.PersistSnapshot(&messages.RemoteSnapshot{
				TimeStamp: time.Now().Unix(),
				LastMSG:   ps.queueMSG.last(),
			})
			for _, v := range ps.queueMSG.getdata() {
				ps.PersistReceive(v)
			}
			break
		}

	case *persistence.RequestSnapshot:
		logs.LogBuild.Println("Request Snapshot")

	case *actor.Stopping:
		ps.client.Disconnect(600)
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

//sendMSG return (response?, error)
func sendMSG(client mqtt.Client, topic string, msg []byte) (bool, error) {

	if !client.IsConnectionOpen() {
		return true, fmt.Errorf("connection is not open")
	}
	tk := client.Publish(topic, 1, false, msg)

	for range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if tk.WaitTimeout(600 * time.Millisecond) {
			return true, nil
		}
		if tk.Error() != nil {
			// logs.LogError.Printf("publish error -> %s, message -> %q", tk.Error(), msg)
			return true, tk.Error()
		}
	}

	return false, fmt.Errorf("message dont send")
}

func (ps *RemoteActor) tickreSend() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if ps.Recovering() {
			continue
		}
		if ps.queueMSG.lendata() > 0 {
			ps.ctx.Send(ps.ctx.Self(), &resendMSG{})
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
	opt.SetConnectTimeout(10 * time.Second)
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
