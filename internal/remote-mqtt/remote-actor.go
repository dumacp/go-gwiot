package remqtt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/database"
	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/messages"
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

	dbpath             = "/SD/boltdbs/gwiotdb"
	databaseName       = "replayeventsdb"
	collectionUsosData = "events"
)

// RemoteActor remote actor
type RemoteActor struct {
	test           bool
	ctx            actor.Context
	pidKeycloak    *actor.PID
	client         mqtt.Client
	token          *oauth2.Token
	lastReconnect  time.Time
	lastSendedMsg  time.Time
	retryDays      int
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

func (ps *RemoteActor) RetryDaysReplay(days int) {
	ps.retryDays = days
}

type reconnectRemote struct{}
type verifyReplay struct{}

func (ps *RemoteActor) connect() error {
	if ps.client != nil {
		ps.client.Disconnect(300)
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
	ps.client = client(ps.token)
	if err := connect(ps.client); err != nil {
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

		if db, err := database.Open(ctx.ActorSystem().Root, dbpath); err != nil {
			if !ps.disableReplay {
				time.Sleep(3 * time.Second)
				panic(fmt.Sprintf("database open error: %s", err))
			}
		} else {
			ps.db = database.NewService(db)
			time.Sleep(1 * time.Second)

			if ps.db != nil && !ps.isDatabaseOpen {
				if err := ps.db.Open(); err != nil {
					logs.LogError.Println("database is closed")
				} else {
					ps.isDatabaseOpen = true
				}
			}
		}

		select {
		case _, ok := <-ps.quit:
			if ok {
				close(ps.quit)
				time.Sleep(1 * time.Second)
			}
		default:
			if ps.quit != nil {
				close(ps.quit)
				time.Sleep(1 * time.Second)
			}
		}
		ps.quit = make(chan int)
		go ps.tickrReconnect(ps.quit)

		if err := ps.connect(); err != nil {
			logs.LogError.Println(err)
		}
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case **actor.Stopping:
		logs.LogInfo.Printf("Stopping, actor, pid: %v\n", ctx.Self())
		close(ps.quit)

	case *verifyReplay:
		if err := func() error {

			if ps.db != nil && !ps.isDatabaseOpen {
				if err := ps.db.Open(); err != nil {
					logs.LogError.Println("database is closed")
				} else {
					ps.isDatabaseOpen = true
				}
			}
			if ps.db == nil || !ps.isDatabaseOpen {
				return errors.New("database is closed")
			}

			if ps.client == nil || !ps.client.IsConnectionOpen() {
				return errors.New("verify Replay cancel, client mqtt remote is nil")
			}

			topic := fmt.Sprintf(remoteQueueEvents, utils.Hostname())

			count := 0
			query := func(id string, el []byte) bool {
				if ps.retryDays > 0 {
					if ref, err := strconv.ParseInt(id, 10, 64); err == nil {
						if time.Since(time.UnixMilli(ref/1000)) < time.Duration(ps.retryDays)*time.Hour*24 {

						}
					}

				}
				diff_time := time.Since(ps.lastReconnect)
				if diff_time < 80*time.Millisecond && diff_time > 0 {
					time.Sleep(diff_time)
				}
				if _, err := sendMSG(ps.client, topic, el, ps.test); err != nil {
					ps.client.Disconnect(300)
					logs.LogWarn.Printf("re-send transaction: %s, errror: %s", id, err)
					return false
				}
				logs.LogWarn.Printf("re-send event (id: %s)", id)
				if count > 30 {
					logs.LogWarn.Println("re-send, count limit")
					go func() {
						time.Sleep(1800 * time.Millisecond)
						ctx.Send(ctx.Self(), &verifyReplay{})
					}()
					return false
				}
				count++
				ps.lastSendedMsg = time.Now()
				// TODO if wait response develop accumulative ids
				ps.db.DeleteWithoutResponse(id, databaseName, collectionUsosData)
				return true
			}
			err := ps.db.Query(databaseName, collectionUsosData, "", false, 30*time.Second, query)
			if err != nil {
				if !ps.disableReplay {
					go func() {
						time.Sleep(1800 * time.Millisecond)
						ctx.Send(ctx.Self(), &verifyReplay{})
					}()
				}
				return err
			}
			return nil
		}(); err != nil {
			logs.LogError.Printf("re-send data error: %s", err)
		}
	case *reconnectRemote:
		t1 := ps.lastReconnect
		if ps.client != nil && ps.client.IsConnectionOpen() {
			break
		}
		logs.LogInfo.Printf("  *****************     RECONNECT   *********************")
		fmt.Println("  *****************     RECONNECT   *********************")

		logs.LogBuild.Printf("last connect at -> %s", t1)
		if t1.Before(time.Now().Add(-30 * time.Second)) {
			ps.lastReconnect = time.Now()
			if err := ps.connect(); err != nil {
				logs.LogError.Println(err)
				if ps.client != nil {
					ps.client.Disconnect(300)
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
			fmt.Printf("new data to send: %q\n", data)

			if ps.client == nil || !ps.client.IsConnectionOpen() {
				return fmt.Errorf("not connection")
			}

			topic := fmt.Sprintf(remoteQueueEvents, utils.Hostname())

			diff_time := time.Since(ps.lastReconnect)
			if diff_time < 100*time.Millisecond && diff_time > 0 {
				time.Sleep(diff_time)
			}
			if _, err := sendMSG(ps.client, topic, data, ps.test); err != nil {
				ps.client.Disconnect(300)
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
				date := time.Now().Format("02-01-2005")
				collection := fmt.Sprintf("%s_%s", collectionUsosData, date)
				if _, err := ps.db.Update(uid, data, databaseName, collection); err != nil {
					logs.LogError.Printf("storage data (id: %s) %q error: %s", uid, data, err)
				}
			}
			if ps.lastReconnect.Before(time.Now().Add(-40 * time.Second)) {
				ctx.Send(ctx.Self(), &reconnectRemote{})
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

// sendMSG return (response?, error)
func sendMSG(client mqtt.Client, topic string, data []byte, test bool) (bool, error) {

	if client == nil || !client.IsConnectionOpen() {
		return true, fmt.Errorf("connection is not open")
	}
	var topicSend string
	if test {
		topicSend = fmt.Sprintf(remoteQueueEventsTest, utils.Hostname())
	} else {
		topicSend = topic
	}

	logs.LogBuild.Printf("data to send: %s", data)
	tk := client.Publish(topicSend, 1, false, data)

	for range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if !tk.WaitTimeout(300 * time.Millisecond) {
			continue
		}
		if tk.Error() != nil {
			return true, tk.Error()
		} else {
			return true, nil
		}
	}

	return false, fmt.Errorf("message dont send")
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
			if (ps.client == nil || !ps.client.IsConnectionOpen()) && ps.ctx != nil {
				t1 := ps.lastReconnect
				if t1.Before(time.Now().Add(-60 * time.Second)) {
					ps.ctx.Send(ps.ctx.Self(), &reconnectRemote{})
				}
			}
		case <-tickVerify.C:
			if (ps.client == nil || !ps.client.IsConnectionOpen()) && ps.ctx != nil {
				ps.ctx.Send(ps.ctx.Self(), &verifyReplay{})
			}
		case <-quit:
			return
		}
	}
}
