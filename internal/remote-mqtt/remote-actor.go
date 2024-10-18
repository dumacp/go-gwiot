package remqtt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/parameters"
	"github.com/dumacp/go-gwiot/internal/remote"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-logs/pkg/logs"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/oauth2"
)

const (
	remoteQueueEvents     = "/devices/%s/events"
	remoteQueueEventsTest = "TEST/%s"
	protocolVersion       = 4  // corresponds to mqtt 3.1.1
	minimumBackoffTime    = 1  // initial backoff time in seconds
	maximumBackoffTime    = 32 // maximum backoff time in seconds

)

type JwtConf struct {
	User         string
	Pass         string
	Realm        string
	ClientID     string
	ClientSecret string
	KeycloakURL  string
}

// RemoteActor remote actor
type RemoteActor struct {
	ctx            actor.Context
	client         mqtt.Client
	clientExternal mqtt.Client
	tokenSource    oauth2.TokenSource
	lastSendedMsg  time.Time
	lastRetry      time.Time
	jwtConf        *JwtConf
	placa          string
	externalBroker string
	externalUser   string
	externalPass   string
	externalTopic  string
	cancel         func()
	test           bool
	retryFlag      bool
	// disableReplay bool
}

// NewRemote new remote actor
func NewRemote(test bool, conf *JwtConf) *RemoteActor {
	r := &RemoteActor{}
	r.test = test
	if conf != nil {
		r.jwtConf = conf
	}

	return r
}

// func (ps *RemoteActor) PID() actor.PID {
// 	ps.retryDays = days
// }

type reconnectRemote struct{}
type reconnectExternalRemote struct{}
type verifyReplay struct{}
type verifyRetry struct{}
type MsgTick struct{}

// Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	ps.ctx = ctx
	// logs.LogBuild.Printf("message arrived to %q, msg: %s (%T)", ctx.Self().GetId(), ctx.Message(), ctx.Message())

	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())
		fmt.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())

		contxt, cancel := context.WithCancel(context.TODO())
		ps.cancel = cancel
		go tick(contxt, ctx, 30*time.Second)

		if ps.jwtConf != nil {
			fmt.Printf("jwtConf: %s\n", ps.jwtConf)
			tks, err := TokenSource(ps.jwtConf.User, ps.jwtConf.Pass, ps.jwtConf.KeycloakURL, ps.jwtConf.Realm, ps.jwtConf.ClientID, ps.jwtConf.ClientSecret)
			if err != nil {
				logs.LogWarn.Println(err)
				break
			}
			tk, err := tks.Token()
			if err != nil {
				logs.LogWarn.Println(err)
				break
			}
			ps.tokenSource = tks
			ps.client = clientWithJwt(tk, utils.RemoteBrokerURL)
		} else {
			ps.client = clientWithoutAuth(utils.RemoteBrokerURL)
		}

		if err := connect(ps.client); err != nil {
			logs.LogError.Println(err)
		} else {
			logs.LogInfo.Printf("  *****************     CONNECTED   *********************")
			fmt.Println("  *****************     CONNECTED   *********************")
			if ctx.Parent() != nil {
				ctx.Request(ctx.Parent(), &remote.MsgReconnect{})
			}
		}
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case *actor.Stopping:
		logs.LogInfo.Printf("Stopping, actor, pid: %v\n", ctx.Self())
		if ps.cancel != nil {
			ps.cancel()
		}
	case *MsgTick:
		if ps.client == nil || !ps.client.IsConnectionOpen() {
			ctx.Send(ctx.Self(), &reconnectRemote{})
		}
		if len(ps.externalBroker) > 0 {
			if ps.clientExternal == nil || !ps.clientExternal.IsConnectionOpen() {
				ctx.Send(ctx.Self(), &reconnectExternalRemote{})
			}
		}
	case *verifyRetry:
		if !ps.retryFlag || time.Since(ps.lastRetry) < 6*time.Second {
			break
		}
		ctx.Send(ctx.Self(), &verifyReplay{})

	case *reconnectRemote:
		if err := func() error {
			if ps.client != nil && ps.client.IsConnectionOpen() {
				return nil
			}
			logs.LogInfo.Printf("  *****************     RECONNECT   *********************")
			fmt.Println("  *****************     RECONNECT   *********************")
			if ps.jwtConf != nil {
				var tk *oauth2.Token
				if ps.tokenSource != nil {
					token, err := ps.tokenSource.Token()
					if err == nil {
						tk = token
					}
				}
				if tk == nil {
					tks, err := TokenSource(ps.jwtConf.User, ps.jwtConf.Pass, ps.jwtConf.KeycloakURL, ps.jwtConf.Realm, ps.jwtConf.ClientID, ps.jwtConf.ClientSecret)
					if err != nil {
						return err
					}
					tk, err = tks.Token()
					if err != nil {
						return err
					}
					ps.tokenSource = tks
				}
				ps.client = clientWithJwt(tk, utils.RemoteBrokerURL)
			} else {
				ps.client = clientWithoutAuth(utils.RemoteBrokerURL)
			}
			if err := connect(ps.client); err != nil {
				return err
			} else {
				fmt.Printf("RECONNECT SUCESSFULL\n")
				logs.LogInfo.Printf("RECONNECT SUCESSFULL")
				fmt.Printf("lastSendedMsg: %s\n", ps.lastSendedMsg)
				if ctx.Parent() != nil {
					ctx.Request(ctx.Parent(), &remote.MsgReconnect{})
				}
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
			if ps.client != nil {
				ps.client.Disconnect(300)
			}
		}
	case *reconnectExternalRemote:
		if err := func() error {
			if ps.clientExternal != nil && ps.client.IsConnectionOpen() {
				return nil
			}
			if len(ps.externalBroker) <= 0 {
				return fmt.Errorf("external broker url is empty")
			}
			fmt.Printf("external client  RECONNECT (%q)\n", ps.externalBroker)
			if len(ps.externalUser) > 0 {
				ps.clientExternal = clientWithUserPass(ps.externalBroker, ps.externalUser, ps.externalPass)
			} else {
				ps.clientExternal = clientWithoutAuth(ps.externalBroker)
			}
			if err := connect(ps.clientExternal); err != nil {
				return err
			} else {
				fmt.Printf("external client RECONNECT SUCESSFULL (%q)\n", ps.externalBroker)
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
			if ps.client != nil {
				ps.client.Disconnect(300)
			}
		}
	case *remote.MsgSendData:
		if err := func() error {
			topic := fmt.Sprintf(remoteQueueEvents, utils.Hostname())
			if _, err := sendMSG(ps.client, topic, msg.Data, ps.test); err != nil {
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			if ctx.Sender() != nil {
				ctx.Respond(&remote.MsgAck{})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
				ctx.Respond(&remote.MsgError{
					Error: err,
				})
			}
		}
	case *parameters.PlatformParameters:
		if msg.Props != nil && len(msg.Props.DEV_PID) > 0 {
			ps.placa = msg.Props.DEV_PID
		}
		if msg.Props != nil && len(msg.Props.BROKER_URL_EXTERNAL) > 0 {
			ps.externalBroker = msg.Props.BROKER_URL_EXTERNAL
		}
		if msg.Props != nil && len(msg.Props.BROKER_USER_EXTERNAL) > 0 {
			ps.externalUser = msg.Props.BROKER_USER_EXTERNAL
		}
		if msg.Props != nil && len(msg.Props.BROKER_PASS_EXTERNAL) > 0 {
			ps.externalPass = msg.Props.BROKER_PASS_EXTERNAL
		}
		if msg.Props != nil && len(msg.Props.BROKER_TOPIC_EXTERNAL) > 0 {
			ps.externalTopic = msg.Props.BROKER_TOPIC_EXTERNAL
		}
	case *remote.MsgExternalSendData:
		if err := func() error {
			if len(ps.externalTopic) <= 0 {
				return fmt.Errorf("external broker url is empty")
			}
			if len(ps.placa) <= 0 {
				ctx.Request(ctx.Parent(), &parameters.GetPlatformParameters{})
				return fmt.Errorf("placa is empty")
			}
			topic := fmt.Sprintf("%s/%s", utils.ExternalRemoteBrokerTopic, strings.ToUpper(ps.placa))
			if _, err := sendMSG(ps.clientExternal, topic, msg.Data, ps.test); err != nil {
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			if ctx.Sender() != nil {
				ctx.Respond(&remote.MsgAck{})
			}
			return nil
		}(); err != nil {
			fmt.Printf("external send error: %s\n", err)
			logs.LogError.Printf("external send error: %s", err)
		}
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

func tick(contxt context.Context, ctx actor.Context, timeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		} else {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()
	rootctx := ctx.ActorSystem().Root
	self := ctx.Self()
	t1 := time.NewTicker(timeout)
	defer t1.Stop()
	t2 := time.NewTicker(90 * time.Second)
	defer t2.Stop()
	t3 := time.NewTicker(3 * time.Second)
	defer t3.Stop()
	for {
		select {
		case <-t1.C:
			rootctx.Send(self, &MsgTick{})
		case <-t2.C:
			rootctx.Send(self, &verifyReplay{})
		case <-t3.C:
			rootctx.Send(self, &verifyRetry{})
		case <-contxt.Done():
			return
		}
	}
}
