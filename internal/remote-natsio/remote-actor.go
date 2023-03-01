package renatsio

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/database"
	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/messages"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

const (
	remoteQueueEvents     = "Events.Offline"
	remoteQueueEventsTest = "TEST/%s"
	protocolVersion       = 4  // corresponds to mqtt 3.1.1
	minimumBackoffTime    = 1  // initial backoff time in seconds
	maximumBackoffTime    = 32 // maximum backoff time in seconds

	dbpath             = "/SD/boltdbs/gwiotdb"
	databaseName       = "replayeventsdb"
	collectionUsosData = "events"
)

// RemoteActor remote actor
type RemoteActor struct {
	test           bool
	ctx            actor.Context
	pidKeycloak    *actor.PID
	conn           *nats.Conn
	token          *oauth2.Token
	lastReconnect  time.Time
	lastSendedMsg  time.Time
	disableReplay  bool
	isDatabaseOpen bool
	db             database.DBservice
	quit           chan int
}

// NewRemote new remote actor
func NewRemote(test bool) *RemoteActor {
	r := &RemoteActor{}
	r.test = test

	return r
}

func (ps *RemoteActor) DisableReplay(disable bool) {
	ps.disableReplay = disable
}

type reconnectRemote struct{}
type verifyReplay struct{}

func (ps *RemoteActor) connect() error {
	if ps.conn != nil && ps.conn.IsConnected() {
		ps.conn.Close()
	}
	if len(keyc.Keycloakurl) > 0 {
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
	}
	var err error

	ps.conn, err = connect(ps.token)
	if err != nil {
		time.Sleep(6 * time.Second)
		return fmt.Errorf("connect error -> %s", err)
	}
	return nil
}

// Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	ps.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())
		fmt.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())

		ps.quit = make(chan int)
		go ps.tickrReconnect(ps.quit)

		if err := ps.connect(); err != nil {
			logs.LogError.Println(err)
		}
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case **actor.Stopping:
		logs.LogInfo.Printf("Stopping, actor, pid: %v\n", ctx.Self())
		close(ps.quit)

	case *reconnectRemote:
		t1 := ps.lastReconnect
		if ps.conn != nil && ps.conn.IsConnected() {
			break
		}
		logs.LogInfo.Printf("  *****************     RECONNECT   *********************")
		fmt.Println("  *****************     RECONNECT   *********************")

		logs.LogBuild.Printf("last connect at -> %s", t1)
		if t1.Before(time.Now().Add(-30 * time.Second)) {
			ps.lastReconnect = time.Now()
			if err := ps.connect(); err != nil {
				logs.LogError.Println(err)
				if ps.conn != nil {
					ps.conn.Close()
				}
			} else {
				fmt.Printf("RECONNECT SUCESSFULL\n")
				logs.LogInfo.Printf("RECONNECT SUCESSFULL")
				fmt.Printf("lastSendedMsg: %s, disableReplay: %v\n", ps.lastSendedMsg, ps.disableReplay)

				if !ps.disableReplay {
					ctx.Send(ctx.Self(), &verifyReplay{})
				}
			}
		}
	case *oauth2.Token:
		ps.token = msg

	case *messages.RemoteMSG2:
		data := prepareMSG(msg)
		if err := func(data []byte) error {
			fmt.Printf("new data to send: %s\n", data)

			if ps.conn == nil || !ps.conn.IsConnected() {
				return fmt.Errorf("not connection")
			}

			topic := remoteQueueEvents

			diff_time := time.Since(ps.lastReconnect)
			if diff_time < 100*time.Millisecond && diff_time > 0 {
				time.Sleep(diff_time)
			}
			headers := map[string]string{"id": utils.Hostname()}
			if _, err := sendMSG(ps.conn, topic, data, headers, ps.test); err != nil {
				ps.conn.Close()
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.GetData())
			}
			ps.lastSendedMsg = time.Now()
			return nil
		}(data); err != nil {
			logs.LogError.Printf("send error: %s", err)
			if ps.db != nil && !ps.isDatabaseOpen {
				if err := ps.db.Open(); err != nil {
					logs.LogError.Println("database is closed")
				} else {
					ps.isDatabaseOpen = true
				}
			}

			if !ps.disableReplay && ps.db != nil && ps.isDatabaseOpen {
				logs.LogBuild.Printf("backup event: %s", data)
				uid := fmt.Sprintf("%d", time.Now().UnixNano())
				if _, err := ps.db.Update(uid, data, databaseName, collectionUsosData); err != nil {
					logs.LogError.Printf("storage data (id: %s) %q error: %s", uid, data, err)
				}
			}
			if ps.lastReconnect.Before(time.Now().Add(-40 * time.Second)) {
				ctx.Send(ctx.Self(), &reconnectRemote{})
			}
		}

	case *actor.Stopping:
		ps.conn.Close()
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")

	}
}

// sendMSG return (response?, error)
func prepareMSG(msg *messages.RemoteMSG2) []byte {

	var data []byte
	if msg.Version == 2 {
		data = []byte(fmt.Sprintf("{\"sDv\": %q, %s, %s}", utils.Hostname(), msg.State[1:len(msg.State)-1], msg.Events[1:len(msg.Events)-1]))
	} else {
		data = msg.GetData()
	}

	return data
}

func (ps *RemoteActor) tickrReconnect(quit chan int) {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		} else {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	tickVerify := time.NewTicker(90 * time.Second)
	defer tickVerify.Stop()

	for {
		select {
		case <-tick.C:
			if (ps.conn == nil || !ps.conn.IsConnected()) && ps.ctx != nil {
				t1 := ps.lastReconnect
				if t1.Before(time.Now().Add(-60 * time.Second)) {
					ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
				}
			}
		case <-tickVerify.C:
			if (ps.conn == nil || !ps.conn.IsConnected()) && ps.ctx != nil {
				ps.ctx.Send(ps.ctx.Self(), &verifyReplay{})
			}
		case <-quit:
			return
		}
	}
}
