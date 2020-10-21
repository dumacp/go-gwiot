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

//RemoteActor remote actor
type RemoteActor struct {
	persistence.Mixin
	test          bool
	lastMSG       *messages.RemoteMSG
	lastCacheMSG  *messages.RemoteMSG
	lastBackMSG   *messages.RemoteSnapshot
	ctx           actor.Context
	pidKeycloak   *actor.PID
	client        mqtt.Client
	token         *oauth2.Token
	lastReconnect time.Time
}

//NewRemote new remote actor
func NewRemote(test bool) *RemoteActor {
	r := &RemoteActor{}
	r.test = test
	r.lastBackMSG = &messages.RemoteSnapshot{TimeStamp: 0, LastMSG: nil}
	go r.tickrReconnect()
	return r
}

type reconnectRemote struct{}

type resendMSG struct{}

func (ps *RemoteActor) connect() error {
	if ps.client != nil {
		ps.client.Disconnect(300)
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
	if ps.token == nil {
		return fmt.Errorf("token is empty %v", nil)
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
		if ps.client != nil && ps.client.IsConnectionOpen() {
			break
		}
		logs.LogBuild.Printf("  *****************     RECONNECT   *********************")
		logs.LogBuild.Printf("  *****************     RECONNECT   *********************")

		logs.LogBuild.Printf("last connect at -> %s", t1)
		if time.Now().After(t1.Add(20 * time.Second)) {
			ps.lastReconnect = time.Now()
			if err := ps.connect(); err != nil {
				logs.LogError.Println(err)
				ps.client.Disconnect(300)
			} else {
				logs.LogInfo.Printf("RECONNECT SUCESSFULL")
				logs.LogBuild.Printf("Request recovery, lastChahe messages -> %s, lastBackup -> %v", time.Unix(ps.lastCacheMSG.GetTimeStamp(), 0), time.Unix(ps.lastBackMSG.GetTimeStamp(), 0))

				if ps.lastCacheMSG == nil || ps.lastCacheMSG.GetTimeStamp() > ps.lastBackMSG.GetTimeStamp() {
					panic(msg)
				}
			}
		}
	case *oauth2.Token:
		ps.token = msg

	case *resendMSG:

	case *messages.RemoteMSG:
		logs.LogBuild.Printf("recovering: %v, remote message -> %s", ps.Recovering(), time.Unix(msg.GetTimeStamp(), 0))
		if !ps.Recovering() {
			logs.LogBuild.Printf("persist message -> %v", time.Unix(msg.TimeStamp, 0))
			ps.lastCacheMSG = msg
			ps.PersistReceive(msg)
		}

		if ps.client == nil || !ps.client.IsConnectionOpen() {
			t1 := ps.lastReconnect
			logs.LogBuild.Printf("  *****************     dont connect!!!!   *********************")
			logs.LogBuild.Printf("last connect at -> %s", t1)
			if time.Now().After(t1.Add(20 * time.Second)) {
				ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
			}
			break
		}

		topic := fmt.Sprintf(remoteQueueEvents, Hostname())

		if _, err := sendMSG(ps.client, topic, msg.GetData(), ps.test); err != nil {
			logs.LogError.Printf("publish error -> %s, message -> %s", err, msg.GetData())

			if !ps.Recovering() {
				ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
			}
		} else {
			ps.lastMSG = msg
		}

	case *messages.RemoteSnapshot:
		logs.LogInfo.Printf("recover snapshot at -> %s", time.Unix(msg.GetTimeStamp(), 0))
		*ps.lastBackMSG = *msg

	case *persistence.ReplayComplete:
		logs.LogBuild.Println("Replay Complete")
		if ps.lastMSG == nil {
			break
		}
		logs.LogBuild.Printf("Replay Request Snapshot, backup -> %v, last messages -> %s", time.Unix(ps.lastBackMSG.TimeStamp, 0), time.Unix(ps.lastMSG.TimeStamp, 0))
		if ps.lastBackMSG.GetTimeStamp() <= ps.lastMSG.GetTimeStamp() {
			ps.execSnapshot()
		}

	case *persistence.RequestSnapshot:
		logs.LogBuild.Println("Request snapshot init")
		if ps.lastMSG == nil {
			break
		}
		if ps.lastCacheMSG != nil && ps.lastCacheMSG.GetTimeStamp() <= ps.lastMSG.GetTimeStamp() {
			break
		}
		ps.execSnapshot()

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
func sendMSG(client mqtt.Client, topic string, msg []byte, test bool) (bool, error) {

	if client != nil && !client.IsConnectionOpen() {
		client.Disconnect(300)
		return true, fmt.Errorf("connection is not open")
	}
	var topicSend string
	if test {
		topicSend = fmt.Sprintf(remoteQueueEventsTest, Hostname())
	} else {
		topicSend = topic
	}
	tk := client.Publish(topicSend, 0, false, msg)

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

func (ps *RemoteActor) execSnapshot() {
	logs.LogBuild.Printf("Request Snapshot -> %v, last messages -> %s", time.Unix(ps.lastBackMSG.TimeStamp, 0), ps.lastBackMSG.GetLastMSG())
	snap := &messages.RemoteSnapshot{}

	if ps.client != nil {
		if !ps.client.IsConnectionOpen() {
			ps.client.Disconnect(300)
			return
		}
	} else {
		return
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
	opt.SetAutoReconnect(false)
	opt.SetConnectRetryInterval(10 * time.Second)
	opt.SetPassword(tk.AccessToken)
	opt.SetUsername("unused")
	opt.SetProtocolVersion(protocolVersion)
	opt.SetTLSConfig(tlsconfig)

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
