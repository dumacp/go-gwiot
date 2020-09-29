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
	remoteMqttBrokerURL   = "wss://fleet-mqtt.nebulae.com.co/mqtt"
	remoteQueueEvents     = "/devices/%s/events"
	remoteQueueEventsTest = "TEST/%s"
	protocolVersion       = 4  // corresponds to mqtt 3.1.1
	minimumBackoffTime    = 1  // initial backoff time in seconds
	maximumBackoffTime    = 32 // maximum backoff time in seconds
)

// //Gateway interface
// type Gateway interface {
// 	Receive(ctx actor.Context)
// 	// Publish(topic string, msg []byte)
// }

//RemoteActor remote actor
type RemoteActor struct {
	persistence.Mixin
	lastMSG       *messages.RemoteMSG
	lastCacheMSG  *messages.RemoteMSG
	lastBackMSG   *messages.RemoteSnapshot
	ctx           actor.Context
	pidKeycloak   *actor.PID
	client        mqtt.Client
	token         *oauth2.Token
	lastReconnect time.Time
	// queueMSG    *queue
	// subscriptions map[string]*actor.PID
}

// type queue struct {
// 	msgs []*messages.RemoteMSG
// 	mux  sync.Mutex
// }

// func newQueue() *queue {
// 	return &queue{
// 		msgs: make([]*messages.RemoteMSG, 0),
// 		mux:  sync.Mutex{},
// 	}
// }

// func (q *queue) put(msg *messages.RemoteMSG) {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	q.msgs = append(q.msgs, msg)
// }

// func (q *queue) pop() *messages.RemoteMSG {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	v := q.msgs[len(q.msgs)-1]
// 	q.msgs = q.msgs[:len(q.msgs)-1]
// 	return v
// }

// func (q *queue) deletedata() {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	if q.msgs != nil {
// 		q.msgs = make([]*messages.RemoteMSG, 0)
// 	}
// }

// func (q *queue) truncate(n uint) {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	if q.msgs != nil && len(q.msgs) >= int(n) {
// 		q.msgs = q.msgs[n:]
// 	}
// }

// func (q *queue) shift() *messages.RemoteMSG {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	if q.msgs == nil {
// 		return nil
// 	}
// 	if len(q.msgs) <= 0 {
// 		return nil
// 	}
// 	data := q.msgs[0]
// 	if len(q.msgs) == 1 {
// 		q.msgs = make([]*messages.RemoteMSG, 0)
// 	} else {
// 		q.msgs = q.msgs[1:]
// 	}
// 	return data
// }

// func (q *queue) getdata() []*messages.RemoteMSG {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	return q.msgs
// }

// func (q *queue) lendata() int {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	return len(q.msgs)
// }

// func (q *queue) last() *messages.RemoteMSG {
// 	q.mux.Lock()
// 	defer q.mux.Unlock()
// 	return q.msgs[len(q.msgs)-1]
// }

//NewRemote new remote actor
func NewRemote() *RemoteActor {
	r := &RemoteActor{}
	r.lastBackMSG = &messages.RemoteSnapshot{TimeStamp: 0, LastMSG: nil}
	go r.tickrReconnect()
	return r
}

type reconnectRemote struct{}

type resendMSG struct{}

func (ps *RemoteActor) connect() error {
	if ps.client != nil && ps.client.IsConnectionOpen() {
		return nil
	}
	ctx := ps.ctx
	request := ctx.RequestFuture(ctx.Parent(), &messages.KeycloakAddressRequest{}, 6*time.Second)
	if err := request.Wait(); err != nil {
		return fmt.Errorf("request keycloak actor address error -> %s", err)
	}

	res, err := request.Result()
	if err != nil {
		return fmt.Errorf("request keycloak actor address error -> %s", err)
	}

	var ok bool
	if ps.pidKeycloak, ok = res.(*actor.PID); !ok {
		return fmt.Errorf("request keycloak actor address error -> %s", err)
	}
	logs.LogBuild.Printf("remote keycloak -> %v", ps.pidKeycloak)

	if ps.client != nil {
		ps.client.Disconnect(300)
	}
	if ps.token == nil || ps.token.Expiry.Before(time.Now()) {
		req := ctx.RequestFuture(ps.pidKeycloak, &messages.TokenRequest{}, 6*time.Second)
		if err := req.Wait(); err != nil {
			return fmt.Errorf("request JWT error -> %s", err)
		}
		res, err := req.Result()
		if err != nil {
			return fmt.Errorf("request JWT error -> %s", err)
		}
		if ps.token, ok = res.(*oauth2.Token); !ok {
			return fmt.Errorf("request JWT error -> %s", err)
		}
		logs.LogBuild.Printf("new jwt -> %v", ps.token)

	}
	ps.client = client(ps.token)
	if err := connect(ps.client); err != nil {
		time.Sleep(6 * time.Second)
		return fmt.Errorf("connect error -> %s", err)
	}
	return nil
}

//Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	ps.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		// logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())

		// ps.queueMSG.deletedata()

		if err := ps.connect(); err != nil {
			logs.LogError.Println(err)
		}
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case *reconnectRemote:
		t1 := ps.lastReconnect
		logs.LogBuild.Printf("last connect at -> %s", t1)
		if time.Now().After(t1.Add(20 * time.Second)) {
			ps.lastReconnect = time.Now()
			if err := ps.connect(); err != nil {
				logs.LogError.Println(err)
			} else {
				logs.LogBuild.Printf("Request recovery, lastChahe messages -> %s, lastBackup -> %v", time.Unix(ps.lastCacheMSG.GetTimeStamp(), 0), time.Unix(ps.lastBackMSG.GetTimeStamp(), 0))

				if ps.lastCacheMSG == nil || ps.lastCacheMSG.GetTimeStamp() > ps.lastBackMSG.GetTimeStamp() {
					panic(msg)
				}
			}
			// ctx.Request(ctx.Parent(), &ErrorRemote{})
		}
	case *oauth2.Token:
		ps.token = msg
		// logs.LogBuild.Printf("new jwt token -> %+v", msg)
		// if ps.client != nil {
		// 	ps.client.Disconnect(300)
		// }
		// ps.client = client(ps.token)
		// if err := connect(ps.client); err != nil {
		// 	time.Sleep(3 * time.Second)
		// 	logs.LogError.Printf("connect error -> %s", err)
		// 	break
		// }

		// if ps.queueMSG.lendata() > 0 {
		// 	ctx.Send(ctx.Self(), &resendMSG{})
		// }

	case *resendMSG:

	case *messages.RemoteMSG:
		logs.LogBuild.Printf("recovering: %v, remote message -> %s", ps.Recovering(), time.Unix(msg.GetTimeStamp(), 0))
		if !ps.Recovering() {
			logs.LogBuild.Printf("persist message -> %v", time.Unix(msg.TimeStamp, 0))
			ps.PersistReceive(msg)
		} else {
			ps.lastCacheMSG = msg
		}

		// if ps.token.Expiry.Before(time.Now()) {
		// 	logs.LogBuild.Printf("before jwt token -> %+v", ps.token)
		// 	if !ps.Recovering() {
		// 		ctx.Send(ctx.Self(), &reconnectRemote{})
		// 	}
		// 	// ps.queueMSG.put(msg)
		// 	break
		// }

		if ps.client == nil || !ps.client.IsConnectionOpen() {
			t1 := ps.lastReconnect
			logs.LogBuild.Printf("last connect at -> %s", t1)
			if time.Now().After(t1.Add(20 * time.Second)) {
				ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
			}
			break
		}

		topic := fmt.Sprintf(remoteQueueEvents, Hostname())
		// ps.lastMSG = msg

		if _, err := sendMSG(ps.client, topic, msg.GetData()); err != nil {
			logs.LogError.Printf("publish error -> %s, message -> %s", err, msg.GetData())

			if !ps.Recovering() {
				t1 := ps.lastReconnect
				if time.Now().After(t1.Add(20 * time.Second)) {
					ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
				}
			}
		} else {
			// ps.queueMSG.put(msg)
			ps.lastMSG = msg
		}

	case *messages.RemoteSnapshot:
		logs.LogInfo.Printf("******************\nrecover snapshot -> %v, last messages -> %s\n******************", time.Unix(msg.GetTimeStamp(), 0), msg.GetLastMSG())
		*ps.lastBackMSG = *msg

	case *persistence.ReplayComplete:
		logs.LogBuild.Println("Replay Complete")
		if ps.lastMSG == nil {
			break
		}
		logs.LogBuild.Println("Replay Complete 1")
		if ps.lastBackMSG.GetTimeStamp() < ps.lastMSG.GetTimeStamp() {
			ctx.Send(ctx.Self(), &persistence.RequestSnapshot{})
		}
		logs.LogBuild.Println("Replay Complete 2")

	case *persistence.RequestSnapshot:
		if ps.lastMSG == nil {
			break
		}
		if ps.lastCacheMSG != nil && ps.lastCacheMSG.GetTimeStamp() > ps.lastMSG.GetTimeStamp() {
			break
		}
		logs.LogBuild.Printf("Request Snapshot -> %v, last messages -> %s", time.Unix(ps.lastBackMSG.TimeStamp, 0), ps.lastBackMSG.GetLastMSG())
		snap := &messages.RemoteSnapshot{}

		if ps.client != nil && !ps.client.IsConnectionOpen() {
			break
		}

		if ps.lastMSG != nil {
			if ps.lastBackMSG.TimeStamp < ps.lastMSG.TimeStamp {
				logs.LogBuild.Printf("Request Snapshot, last messages -> %s, lastBackup -> %v", time.Unix(ps.lastMSG.GetTimeStamp(), 0), time.Unix(ps.lastBackMSG.GetTimeStamp(), 0))
				snap.TimeStamp = ps.lastMSG.GetTimeStamp()
				ps.PersistSnapshot(snap)
				logs.LogBuild.Printf("Request Snapshot, lastBackup -> %v", time.Unix(snap.GetTimeStamp(), 0))
				*ps.lastBackMSG = *snap
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

//sendMSG return (response?, error)
func sendMSG(client mqtt.Client, topic string, msg []byte) (bool, error) {

	if client != nil && !client.IsConnectionOpen() {
		return true, fmt.Errorf("connection is not open")
	}
	tk := client.Publish(topic, 1, false, msg)

	for range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if tk.WaitTimeout(300 * time.Millisecond) {
			return true, nil
		}
		if tk.Error() != nil {
			// logs.LogError.Printf("publish error -> %s, message -> %q", tk.Error(), msg)
			return true, tk.Error()
		}
	}

	return false, fmt.Errorf("message dont send")
}

func (ps *RemoteActor) tickrReconnect() {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if ps.Recovering() {
			continue
		}
		if ps.client == nil || !ps.client.IsConnectionOpen() {
			t1 := ps.lastReconnect
			if time.Now().After(t1.Add(20 * time.Second)) {
				ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
			}
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
