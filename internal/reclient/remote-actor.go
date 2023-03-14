package reclient

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/database"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

var (
	Dbpath             = "/SD/boltdbs/gwiotdb"
	DatabaseName       = "replayeventsdb"
	CollectionUsosData = "events"
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
	lastReconnect  time.Time
	lastSendedMsg  time.Time
	lastRetry      time.Time
	propsClient    *actor.Props
	pidClient      *actor.PID
	retryDays      int
	retryFlag      bool
	disableReplay  bool
	isDatabaseOpen bool
	isConnected    bool
	db             database.DBservice
	cancel         func()
}

// NewRemote new remote actor
func NewRemote(client *actor.Props) *RemoteActor {
	r := &RemoteActor{}
	r.propsClient = client

	return r
}

func (ps *RemoteActor) DisableReplay(disable bool) {
	ps.disableReplay = disable
}

func (ps *RemoteActor) RetryDaysReplay(days int) {
	ps.retryDays = days
}

type verifyReplay struct{}
type verifyRetry struct{}

// Receive function
func (ps *RemoteActor) Receive(ctx actor.Context) {
	ps.ctx = ctx

	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())
		fmt.Printf("Starting, actor, pid: %s\n", ctx.Self().GetId())

		if db, err := database.Open(ctx.ActorSystem().Root, Dbpath); err != nil {
			if !ps.disableReplay {
				time.Sleep(10 * time.Second)
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
		pid, err := ctx.SpawnNamed(ps.propsClient, "clientremote")
		if err != nil {
			time.Sleep(10 * time.Second)
			panic(fmt.Sprintf("database open error: %s", err))
		}
		ps.pidClient = pid
		contxt, cancel := context.WithCancel(context.TODO())
		ps.cancel = cancel
		go tick(contxt, ctx)

		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
	case *actor.Stopping:
		logs.LogInfo.Printf("Stopping, actor, pid: %v\n", ctx.Self())
		if ps.cancel != nil {
			ps.cancel()
		}
	case *verifyRetry:
		if !ps.retryFlag || time.Since(ps.lastRetry) < 6*time.Second {
			break
		}
		ctx.Send(ctx.Self(), &verifyReplay{})
	case *MsgReconnect:
		ctx.Send(ctx.Self(), &verifyReplay{})
		ps.isConnected = true
	case *verifyReplay:
		if err := func() error {
			if ps.retryFlag {
				ps.retryFlag = false
			}
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
			count := 0
			query := func(id string, el []byte) bool {
				if count > 30 {
					logs.LogWarn.Println("re-send, count limit")
					ps.retryFlag = true
					return false
				}
				count++
				tid, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					ps.db.DeleteWithoutResponse(id, DatabaseName, CollectionUsosData)
					return true
				}
				fmt.Printf("time since: %s\n", time.Since(time.UnixMilli(tid/1000_000)))
				if ps.retryDays > 0 && time.Since(time.UnixMilli(tid/1000_000)) > time.Duration(ps.retryDays)*24*time.Hour {
					collection := fmt.Sprintf("%s_%s", CollectionUsosData, time.UnixMilli(tid/1000_000).Format("20060102"))
					ps.db.UpdateAsync(id, el, DatabaseName, collection)
					fmt.Printf("storage data in %q(id: %s) %q\n", collection, id, el)
					ps.db.DeleteWithoutResponse(id, DatabaseName, CollectionUsosData)
					return true
				}
				if !ps.isConnected {
					logs.LogWarn.Println("client is not connected")
					return false
				}
				res, err := ctx.RequestFuture(ps.pidClient, &MsgSendData{
					Data: el,
				}, 5*time.Second).Result()
				if err != nil {
					logs.LogWarn.Printf("re-send transaction: %s, error_: %s", id, err)
					return false
				}
				if err, ok := res.(*MsgError); ok {
					logs.LogWarn.Printf("re-send transaction: %s, error__: %s", id, err)
					return false
				}
				logs.LogWarn.Printf("re-send event (id: %s)", id)

				ps.lastSendedMsg = time.Now()
				// TODO if wait response develop accumulative ids
				ps.db.DeleteWithoutResponse(id, DatabaseName, CollectionUsosData)
				return true
			}
			if err := ps.db.Query(DatabaseName, CollectionUsosData, "", false, 30*time.Second, query); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			logs.LogError.Printf("re-send data error: %s", err)
		}

	case *messages.RemoteMSG2:
		data := prepareMSG(msg)
		if err := func(data []byte) error {
			fmt.Printf("new data to send: %q\n", data)
			if !ps.isConnected {
				return fmt.Errorf("client is not connected")
			}
			diff_time := time.Since(ps.lastReconnect)
			if diff_time < 100*time.Millisecond && diff_time > 0 {
				time.Sleep(diff_time)
			}
			if err := ctx.RequestFuture(ps.pidClient, &MsgSendData{
				Data: data,
			}, 5*time.Second).Wait(); err != nil {
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.GetData())
			}
			ps.lastSendedMsg = time.Now()
			return nil
		}(data); err != nil {
			logs.LogWarn.Printf("send error: %s", err)
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
				if _, err := ps.db.Update(uid, data, DatabaseName, CollectionUsosData); err != nil {
					logs.LogError.Printf("storage data (id: %s) %q error: %s", uid, data, err)
				}
			}
		}
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

func tick(contxt context.Context, ctx actor.Context) {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		} else {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()
	rootctx := ctx.ActorSystem().Root
	self := ctx.Self()
	t1 := time.NewTimer(3 * time.Second)
	defer t1.Stop()
	t2 := time.NewTicker(90 * time.Second)
	defer t2.Stop()
	t3 := time.NewTicker(3 * time.Second)
	defer t3.Stop()
	for {
		select {
		case <-t1.C:
			rootctx.Send(self, &verifyReplay{})
		case <-t2.C:
			rootctx.Send(self, &verifyReplay{})
		case <-t3.C:
			rootctx.Send(self, &verifyRetry{})
		case <-contxt.Done():
			return
		}
	}
}
