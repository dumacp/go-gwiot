package renatsio

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
)

type ChildNats struct {
	pidMain *actor.PID
	js      nats.JetStreamContext
	contxt  context.Context
	conn    *nats.Conn
	cancel  func()
}

func NewChildNatsio(pid *actor.PID) *ChildNats {
	return &ChildNats{pidMain: pid}
}

func (a *ChildNats) Receive(ctx actor.Context) {

	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started new internal client %q", ctx.Self().GetId())
		if a.pidMain != nil {
			ctx.Request(a.pidMain, &Connection{})
		}
		contxt, cancel := context.WithCancel(context.TODO())
		a.contxt = contxt
		a.cancel = cancel
	case *ConnectionResponse:
		a.conn = msg.Conn
		js, err := a.conn.JetStream()
		if err != nil {
			time.Sleep(3 * time.Second)
			ctx.Poison(ctx.Self())
			logs.LogWarn.Printf("natsio connection error: %s", err)
			break
		}
		a.js = js
	case *gwiotmsg.Event:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.Subject
			headers := map[string]string{"id": utils.Hostname()}
			if err := publish(a.conn, a.js, topic, data, headers); err != nil {
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
			if ctx.Sender() != nil {
				ctx.Respond(&gwiotmsg.Ack{})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			if ctx.Sender() != nil {
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
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}
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
				ctx.Respond(&gwiotmsg.Error{
					Error: err.Error(),
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
				return fmt.Errorf("publish error -> %s, message -> %s", err, msg.Data)
			}

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
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")
	}

}
