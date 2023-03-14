package renatsio

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/reclient"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

const (
// remoteQueueEvents     = "Events.Offline.%s"
// remoteQueueEventsTest = "TEST/%s"
// protocolVersion       = 4  // corresponds to mqtt 3.1.1
// minimumBackoffTime    = 1  // initial backoff time in seconds
// maximumBackoffTime    = 32 // maximum backoff time in seconds

// dbpath             = "/SD/boltdbs/gwiotdb"
// databaseName       = "replayeventsdb"
// collectionUsosData = "events"
)

// RemoteActor remote actor
type NatsActor struct {
	lastReconnect time.Time
	lastSendedMsg time.Time
	url           string
	jwtConf       *JwtConf
	conn          *nats.Conn
	js            nats.JetStreamContext
	tokenSource   oauth2.TokenSource
	contxt        context.Context
	cancel        func()
	test          bool
	// disableReplay bool
}

// NewRemote new remote actor
func NewClientNatsio(url string, jwtConf *JwtConf, test bool) *NatsActor {
	r := &NatsActor{}
	r.test = test
	r.url = url
	r.jwtConf = jwtConf
	return r
}

type JwtConf struct {
	User         string
	Pass         string
	Realm        string
	ClientID     string
	ClientSecret string
	KeycloakURL  string
}

// Receive function
func (a *NatsActor) Receive(ctx actor.Context) {
	fmt.Printf("message in actor: %s, msg type: %T, msg: %q\n", ctx.Self().GetId(), ctx.Message(), ctx.Message())
	switch msg := ctx.Message().(type) {

	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())
		fmt.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())

		if err := func() error {
			contxt, cancel := context.WithCancel(context.TODO())
			a.contxt = contxt
			a.cancel = cancel

			go tickrReconnect(contxt, ctx)

			var err error
			if a.jwtConf != nil {
				fmt.Printf("jwtConf: %s\n", a.jwtConf)
				tks, err := TokenSource(a.jwtConf.User, a.jwtConf.Pass, a.jwtConf.KeycloakURL, a.jwtConf.Realm, a.jwtConf.ClientID, a.jwtConf.ClientSecret)
				if err != nil {
					return err
				}
				tk, err := tks.Token()
				if err != nil {
					return err
				}
				a.tokenSource = tks
				a.conn, err = connectWithJwt(a.url, tk)
				if err != nil {
					return err
				}
				a.js, err = a.conn.JetStream()
				if err != nil {
					return err
				}
			} else {
				a.conn, err = clientWithoutAuth(a.url)
				if err != nil {
					return err
				}
				a.js, err = a.conn.JetStream()
				if err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
		}
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case *Reconnect:
		if a.conn != nil && (a.conn.IsConnected() || a.conn.IsReconnecting()) {
			break
		}
		if err := func() error {
			t1 := a.lastReconnect
			logs.LogInfo.Printf("  *****************     RECONNECT   *********************")
			fmt.Println("  *****************     RECONNECT   *********************")
			logs.LogBuild.Printf("last connect at -> %s", t1)
			if t1.Before(time.Now().Add(-30 * time.Second)) {
				a.lastReconnect = time.Now()
				var err error
				if a.jwtConf != nil {
					fmt.Printf("jwtConf: %s\n", a.jwtConf)
					tks, err := TokenSource(a.jwtConf.User, a.jwtConf.Pass, a.jwtConf.KeycloakURL, a.jwtConf.Realm, a.jwtConf.ClientID, a.jwtConf.ClientSecret)
					if err != nil {
						return err
					}
					tk, err := tks.Token()
					if err != nil {
						return err
					}
					a.tokenSource = tks
					a.conn, err = connectWithJwt(a.url, tk)
					if err != nil {
						return err
					}
					a.js, err = a.conn.JetStream()
					if err != nil {
						return err
					}
				} else {
					a.conn, err = clientWithoutAuth(a.url)
					if err != nil {
						return err
					}
					a.js, err = a.conn.JetStream()
					if err != nil {
						return err
					}
				}
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
		}
	case *gwiotmsg.Ping:
		if ctx.Sender() != nil {
			ctx.Respond(&gwiotmsg.Pong{})
		}
	case *gwiotmsg.Event:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.Subject
			headers := map[string]string{"id": utils.Hostname()}
			if err := publish(a.conn, a.js, topic, data, headers); err != nil {
				// a.conn.Close()
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			a.lastSendedMsg = time.Now()
			if ctx.Sender() != nil && msg.Ack {
				ctx.Respond(&gwiotmsg.Ack{})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil && msg.Ack {
				ctx.Respond(&gwiotmsg.Error{
					Error: err.Error(),
				})
			}
		}
	case *gwiotmsg.RequestPubSub:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.Subject
			headers := map[string]string{"id": utils.Hostname()}
			response, err := requestPubSub(a.conn, topic, data, headers, 3*time.Second)
			if err != nil {
				// a.conn.Close()
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			a.lastSendedMsg = time.Now()
			if ctx.Sender() != nil {
				meta, _ := response.Metadata()
				fmt.Printf("metadata in response: %v\n", meta)
				ctx.Respond(&gwiotmsg.Response{
					Data: response.Data,
				})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&reclient.MsgError{
					Error: err,
				})
			}
		}
	case *gwiotmsg.Request:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.Subject
			headers := map[string]string{"id": utils.Hostname()}
			err := request(ctx, a.conn, a.js, topic, msg.Reply, data, headers, time.Duration(msg.GetTimeout())*time.Second)
			if err != nil {
				// a.conn.Close()
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			a.lastSendedMsg = time.Now()

			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&gwiotmsg.Error{
					Error: err.Error(),
				})
			}
		}
	case *gwiotmsg.SubcriptionSubject:
		subs, err := subscription(ctx, a.conn, a.js, msg.GetSubject())
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		go func() {
			<-a.contxt.Done()
			if err := subs.Unsubscribe(); err != nil {
				logs.LogWarn.Println(err)
			}
		}()
	case *gwiotmsg.WatchKeyValue:
		subs, err := wathcKV(ctx, a.conn, a.js, msg.GetBucket(), msg.GetKey())
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		go func() {
			<-a.contxt.Done()
			if err := subs.Stop(); err != nil {
				logs.LogWarn.Println(err)
			}
		}()
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
		if a.conn != nil {
			a.conn.Close()
		}
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

func tickrReconnect(contxt context.Context, ctx actor.Context) {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		} else {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()

	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	tickVerify := time.NewTicker(90 * time.Second)
	defer tickVerify.Stop()

	for {
		select {
		case <-tick.C:
			ctxroot.Send(self, &Reconnect{})
		case <-tickVerify.C:
		case <-contxt.Done():
			return
		}
	}
}
