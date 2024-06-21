package renatsio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/coreos/go-oidc"
	"github.com/dumacp/go-gwiot/internal/localevents"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/events"
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

	INSTANCE_ID = "natsio-actor"
)

// RemoteActor remote actor
type NatsActor struct {
	lastReconnect time.Time
	url           string
	orgID         string
	groupName     string
	jwtConf       *JwtConf
	conn          *nats.Conn
	js            nats.JetStreamContext
	tokenSource   oauth2.TokenSource
	userInfo      *oidc.UserInfo
	contxt        context.Context
	evs           *eventstream.EventStream
	subs          map[string]*eventstream.Subscription
	cancel        func()
	test          bool
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

func subscribe(ctx actor.Context, evs *eventstream.EventStream) *eventstream.Subscription {
	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()
	pid := ctx.Sender()

	return evs.Subscribe(func(evt interface{}) {
		ctxroot.RequestWithCustomSender(pid, evt, self)

	})
}

// Receive function
func (a *NatsActor) Receive(ctx actor.Context) {
	fmt.Printf("message in actor: %s, msg type: %T, msg: %q\n", ctx.Self().GetId(), ctx.Message(), ctx.Message())
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())
		fmt.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())

		if a.evs == nil {
			a.evs = new(eventstream.EventStream)
		}
		if a.subs == nil {
			a.subs = make(map[string]*eventstream.Subscription, 0)
		}

		propsLocalEvents := actor.PropsFromFunc((&localevents.Actor{}).Receive)
		if _, err := ctx.SpawnNamed(propsLocalEvents, "localevents-actor"); err != nil {
			fmt.Printf("create local events actor error: %s", err)
			logs.LogError.Printf("create local events actor error: %s", err)
		}

		if err := func() error {
			contxt, cancel := context.WithCancel(context.TODO())
			a.contxt = contxt
			a.cancel = cancel

			go tickrReconnect(contxt, ctx)

			var err error
			if a.jwtConf != nil {
				fmt.Printf("jwtConf: %s\n", a.jwtConf)
				tks, userInfo, err := TokenSource(a.jwtConf.User, a.jwtConf.Pass, a.jwtConf.KeycloakURL, a.jwtConf.Realm, a.jwtConf.ClientID, a.jwtConf.ClientSecret)
				if err != nil {
					return err
				}
				a.userInfo = userInfo
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
			logs.LogWarn.Printf("connect nats error: %s", err)
			// a.evs.Publish(&Disconnected{Error: err})
		}
		logs.LogInfo.Printf("Started, actor, pid: %v\n", ctx.Self())
	case *Connection:
		if ctx.Sender() == nil {
			break
		}
		if sub, ok := a.subs[ctx.Sender().GetId()]; ok {
			a.evs.Unsubscribe(sub)
			delete(a.subs, ctx.Sender().GetId())
		}
		a.subs[ctx.Sender().GetId()] = subscribe(ctx, a.evs)
		if a.conn == nil || !a.conn.IsConnected() {
			ctx.Respond(&Disconnected{Error: fmt.Errorf("not connection")})
			break
		}
		ctx.Respond(&ConnectionResponse{
			Conn: a.conn,
		})
	case *Reconnect:
		if a.conn != nil && (a.conn.IsConnected() || a.conn.IsReconnecting()) {
			break
		}
		if err := func() error {
			t1 := a.lastReconnect
			logs.LogBuild.Printf("last connect at -> %s", t1)
			if t1.Before(time.Now().Add(-30 * time.Second)) {
				logs.LogInfo.Printf("try RECONNECTING")
				fmt.Println("  *****************     RECONNECT   *********************")
				a.lastReconnect = time.Now()
				var err error
				if a.jwtConf != nil {
					fmt.Printf("jwtConf: %s\n", a.jwtConf)
					tks, userInfo, err := TokenSource(a.jwtConf.User, a.jwtConf.Pass, a.jwtConf.KeycloakURL, a.jwtConf.Realm, a.jwtConf.ClientID, a.jwtConf.ClientSecret)
					if err != nil {
						return err
					}
					a.userInfo = userInfo
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
				a.evs.Publish(&ConnectionResponse{
					Conn: a.conn,
				})
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Printf("connect nats error: %s", err)
			a.evs.Publish(&Disconnected{Error: err})
		}
	case *localevents.GetGroupName:
		ctx.Respond(&localevents.GetGroupNameResponse{GroupName: a.getGorupName()})
	case *localevents.GetOrgID:
		ctx.Respond(&localevents.GetOrgIDResponse{OrgID: a.getOrgID()})
	case *GetGroupName:
		ctx.Respond(&GetGroupNameResponse{GroupName: a.getGorupName()})
	case *GetOrgID:
		ctx.Respond(&GetOrgIDResponse{OrgID: a.getOrgID()})

		// if len(a.orgID) > 0 {
		// 	ctx.Respond(&GetOrgIDResponse{OrgID: a.orgID})
		// } else if a.userInfo == nil {
		// 	// orgId := ""
		// 	ctx.Respond(&GetOrgIDResponse{OrgID: ""})
		// } else {
		// 	// fmt.Printf("claims: %v\n", a.userInfo.Claims)
		// 	claims := make(map[string]interface{})
		// 	if err := a.userInfo.Claims(&claims); err != nil {
		// 		ctx.Respond(&GetOrgIDResponse{OrgID: ""})
		// 	} else {
		// 		fmt.Printf("claims: %v\n", claims)
		// 		if v, ok := claims["group_name"]; ok {
		// 			if v, ok := v.(string); ok {
		// 				sp := strings.Split(v, "_")
		// 				if len(sp) < 2 {
		// 					ctx.Respond(&GetOrgIDResponse{OrgID: ""})
		// 				} else {
		// 					ctx.Respond(&GetOrgIDResponse{OrgID: sp[1]})
		// 				}
		// 			}
		// 		} else {
		// 			ctx.Respond(&GetOrgIDResponse{OrgID: ""})
		// 		}
		// 	}
		// }
	case *events.Message[string, int]:
		if a.conn == nil || a.js == nil {
			break
		}
		orgId := a.getOrgID()
		groupName := strings.Split(a.getGorupName(), "_")[0]
		hostname := utils.Hostname()
		msg.GroupName(groupName).OrganizationId(orgId).DeviceSerial(hostname)
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("marshal error: %s\n", err)
			break
		}
		fmt.Printf("message to send (topic: %q): %s\n", fmt.Sprintf("Events.%s.%s", hostname, msg.Type()), string(data))
		publish(a.conn, a.js, fmt.Sprintf("Events.%s", msg.Type()), data, map[string]string{"org_id": orgId, "snDevice": hostname})
	case *gwiotmsg.Ping:
		if ctx.Sender() != nil {
			ctx.Respond(&gwiotmsg.Pong{})
		}
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
