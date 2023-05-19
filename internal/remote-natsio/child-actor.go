package renatsio

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type ChildNats struct {
	parentId        string
	pidRemoteParent *actor.PID
	conn            *nats.Conn
	js              nats.JetStreamContext
	contxt          context.Context
	subscriptions   map[string]RemoteSubscription
	cancel          func()
}

type RemoteSubscription struct {
	Sender  *actor.PID
	Message any
}

func NewChildNatsio(parentId string) *ChildNats {
	a := &ChildNats{}
	a.parentId = parentId
	a.subscriptions = make(map[string]RemoteSubscription)
	return a
}

func (a *ChildNats) Receive(ctx actor.Context) {

	logs.LogBuild.Printf("Message arrived in %s: %s, %T, %s",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started new internal client %q, address: %s (%s)",
			ctx.Self().GetId(), ctx.Self().GetAddress(), ctx.Parent())

		success := false
		var pid *actor.PID
		for range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
			pid = actor.NewPID(ctx.Self().GetAddress(), fmt.Sprintf("%s/%s", a.parentId, INSTANCE_ID))
			fmt.Printf("////////////////// pid: %s\n", pid)
			if err := ctx.RequestFuture(pid, &gwiotmsg.Ping{}, 500*time.Millisecond).Wait(); err != nil {
				logs.LogWarn.Printf("actor %q is not ready (error: %s)", pid.GetId(), err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			success = true
			break
		}
		if !success {
			logs.LogWarn.Panicf("actor %q is not ready", pid.GetId())
		}
		ctx.Request(pid, &Connection{})
		contxt, cancel := context.WithCancel(context.TODO())
		a.contxt = contxt
		a.cancel = cancel
		go tick(contxt, ctx, 30*time.Second)
	case *tickMsg:
		if a.conn == nil || a.js == nil {
			break
		}
		if len(a.subscriptions) > 0 {
			for _, v := range a.subscriptions {
				switch msg := v.Message.(type) {
				case *gwiotmsg.SubcriptionSubject:
					ctx.RequestWithCustomSender(ctx.Self(), msg, v.Sender)
				case *gwiotmsg.WatchKeyValue:
					ctx.RequestWithCustomSender(ctx.Self(), msg, v.Sender)
				}
			}
		}
	case *ConnectionResponse:
		a.conn = msg.Conn
		js, err := a.conn.JetStream()
		if err != nil {
			time.Sleep(3 * time.Second)
			ctx.Poison(ctx.Self())
			logs.LogWarn.Printf("natsio connection error: %s", err)
			a.js = nil
			break
		}
		a.js = js
		if a.pidRemoteParent != nil {
			ctx.Send(a.pidRemoteParent, &gwiotmsg.Connected{})
		}
	case *Disconnected:
		a.conn = nil
		if a.pidRemoteParent != nil {
			ctx.Send(a.pidRemoteParent, &gwiotmsg.Disconnected{
				Error: msg.Error.Error(),
			})
		}
	case *gwiotmsg.StatusConnRequest:
		if ctx.Sender() == nil {
			break
		}
		if a.conn != nil && a.js != nil {
			ctx.Respond(&gwiotmsg.Disconnected{
				Error: "not connected",
			})
		} else {
			ctx.Respond(&gwiotmsg.Connected{})
		}
	case *gwiotmsg.Ping:
		if ctx.Sender() == nil {
			break
		}
		ctx.Respond(&gwiotmsg.Pong{})
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
		a.pidRemoteParent = ctx.Sender()
		var uids string
		uid, err := uuid.NewRandom()
		if err != nil {
			uids = fmt.Sprintf("%d", time.Now().UnixNano())
		} else {
			uids = uid.String()
		}
		a.subscriptions[uids] = RemoteSubscription{
			Sender:  ctx.Sender(),
			Message: msg,
		}
		subs, err := subscription(ctx.ActorSystem().Root, ctx.Sender(), a.conn, a.js, msg.GetSubject())
		if err != nil {
			logs.LogWarn.Println(err)
			// if ctx.Sender() != nil {
			// 	ctx.Respond(&gwiotmsg.Error{
			// 		Error: err.Error(),
			// 	})
			// }
			break
		}
		delete(a.subscriptions, uids)
		// if ctx.Sender() != nil {
		// 	ctx.Respond(&gwiotmsg.Ack{})
		// }
		go func() {
			subs := subs
			<-a.contxt.Done()
			if err := subs.Unsubscribe(); err != nil {
				logs.LogWarn.Println(err)
			}
		}()
	case *gwiotmsg.WatchKeyValue:
		if ctx.Sender() == nil {
			break
		}
		a.pidRemoteParent = ctx.Sender()
		uids := fmt.Sprintf("%s-%s-%s", ctx.Sender().GetId(), msg.GetBucket(), msg.GetKey())
		a.subscriptions[uids] = RemoteSubscription{
			Sender:  ctx.Sender(),
			Message: msg,
		}
		subs, err := wathcKV(ctx.ActorSystem().Root, ctx.Sender(), a.conn, a.js, msg.GetBucket(), msg.GetKey(), msg.GetIncludeHistory())
		if err != nil {
			logs.LogWarn.Printf("watchKeyValue error: %s", err)
			// if ctx.Sender() != nil {
			// 	ctx.Respond(&gwiotmsg.Error{
			// 		Error: err.Error(),
			// 	})
			// }
			break
		}
		delete(a.subscriptions, uids)
		// if ctx.Sender() != nil {
		// 	ctx.Respond(&gwiotmsg.Ack{})
		// }
		go func() {
			subs := subs
			<-a.contxt.Done()
			fmt.Printf("stopping watch: %v\n", subs)
			// if err := subs.Unsubscribe(); err != nil {
			if err := subs.Stop(); err != nil {
				logs.LogWarn.Println(err)
			}
			fmt.Printf("stopped watch: %v\n", subs)
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

type tickMsg struct{}

func tick(contxt context.Context, ctx actor.Context, timeout time.Duration) {

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()

	t_0 := time.NewTimer(1 * time.Second)
	defer t_0.Stop()
	t_1 := time.NewTicker(timeout)
	defer t_1.Stop()

	for {
		select {
		case <-contxt.Done():
			return
		case <-t_0.C:
			ctxroot.Request(self, &tickMsg{})
		case <-t_1.C:
			ctxroot.Request(self, &tickMsg{})
		}
	}
}
