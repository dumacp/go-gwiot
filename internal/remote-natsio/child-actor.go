package renatsio

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/asynkron/protoactor-go/router"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
)

type ChildNats struct {
	parentId           string
	orgId              string
	pidGwiot           *actor.PID
	pidRemoteParent    *actor.PID
	conn               *nats.Conn
	js                 nats.JetStreamContext
	contxt             context.Context
	subscriptions      map[string]*RemoteSubscription
	subWatchers        map[string]nats.KeyWatcher
	subSubcripters     map[string]*nats.Subscription
	subsGroupBroadcast map[string]*actor.PID
	cancel             func()
}

type RemoteSubscription struct {
	Sender  *actor.PID
	Message any
	Retry   int
}

func NewChildNatsio(parentId string) func() actor.Actor {
	return func() actor.Actor {
		a := &ChildNats{}
		a.parentId = parentId
		a.subscriptions = make(map[string]*RemoteSubscription)
		a.subWatchers = make(map[string]nats.KeyWatcher)
		a.subSubcripters = make(map[string]*nats.Subscription)
		a.subsGroupBroadcast = make(map[string]*actor.PID)
		return a
	}
}

func (a *ChildNats) Receive(ctx actor.Context) {

	// fmt.Println("///////////////////////////////////////////////////////////////////////////////")

	fmt.Printf("Message arrived in %s: %s, %T, %s\n",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started new internal client %q, address: %s (%s)",
			ctx.Self().GetId(), ctx.Self().GetAddress(), ctx.Parent())

		success := false
		var pid *actor.PID
		for range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
			pid = actor.NewPID(ctx.Self().GetAddress(), a.parentId)
			fmt.Printf("/////////////// pid: %s\n", pid)
			if err := ctx.RequestFuture(pid, &gwiotmsg.Ping{}, 500*time.Millisecond).Wait(); err != nil {
				logs.LogWarn.Printf("actor %q is not ready (error: %s)", pid.GetId(), err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			success = true
			break
		}
		if !success {
			time.Sleep(3 * time.Second)
			logs.LogWarn.Panicf("actor %q is not ready", pid.GetId())
		}
		a.pidGwiot = pid
		ctx.Watch(a.pidGwiot)
		ctx.Request(pid, &Connection{})
		contxt, cancel := context.WithCancel(context.TODO())
		a.contxt = contxt
		a.cancel = cancel
		go tick(contxt, ctx, 180*time.Second)
	case *tickMsg:
		if a.conn == nil || a.js == nil {
			if a.pidGwiot != nil {
				ctx.Request(a.pidGwiot, &Connection{})
			}
			break
		}
		if len(a.subscriptions) > 0 {
			for k, v := range a.subscriptions {
				fmt.Printf("re subscription: %s, %T\n", k, v.Message)
				switch msg := v.Message.(type) {
				case *gwiotmsg.SubcriptionSubject:
					ctx.RequestWithCustomSender(ctx.Self(), msg, v.Sender)
				case *gwiotmsg.WatchKeyValue:
					ctx.RequestWithCustomSender(ctx.Self(), msg, v.Sender)
				}
				// delete(a.subscriptions, k)
			}
		}
	case *ConnectionResponse:
		a.conn = msg.Conn
		opts := make([]nats.JSOpt, 0)
		if a.contxt != nil {
			opts = append(opts, nats.Context(a.contxt))
		}
		js, err := a.conn.JetStream(opts...)
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
		if a.conn == nil || a.js == nil {
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
	case *gwiotmsg.HttpPostRequest:
		if ctx.Sender() == nil {
			break
		}
		if a.pidGwiot == nil {
			ctx.Respond(&gwiotmsg.Error{
				Error: "gwiot actor not found",
			})
			break
		}
		if res, err := ctx.RequestFuture(a.pidGwiot, msg, 10*time.Second).Result(); err != nil {
			// fmt.Printf("error request http ___: %s\n", err)
			ctx.Respond(&gwiotmsg.HttpPostResponse{
				Error: err.Error(),
			})
		} else if jwtRes, ok := res.(*gwiotmsg.HttpPostResponse); ok {
			ctx.Respond(jwtRes)
		} else {
			ctx.Respond(&gwiotmsg.HttpPostResponse{
				Error: fmt.Sprintf("error response: %T", res),
			})
		}
	case *gwiotmsg.HttpGetRequest:
		fmt.Printf("child http get request: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidGwiot == nil {
			ctx.Respond(&gwiotmsg.Error{
				Error: "gwiot actor not found",
			})
			break
		}
		if res, err := ctx.RequestFuture(a.pidGwiot, msg, 10*time.Second).Result(); err != nil {
			// fmt.Printf("error request http ___: %s\n", err)
			ctx.Respond(&gwiotmsg.HttpGetResponse{
				Error: err.Error(),
			})
		} else if jwtRes, ok := res.(*gwiotmsg.HttpGetResponse); ok {
			ctx.Respond(jwtRes)
		} else {
			ctx.Respond(&gwiotmsg.HttpGetResponse{
				Error: fmt.Sprintf("error response: %T", res),
			})
		}
	case *gwiotmsg.KvEntryMessage:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.GetBucket()
			key := msg.GetKey()
			kv, err := a.js.KeyValue(topic)
			if err != nil {
				return fmt.Errorf("update kv %q error: %s", topic, err)
			}
			var rev uint64
			if msg.Rev > 0 {
				rev, err = updateKeyValue(a.conn, kv, key, data, msg.Rev)
				if err != nil {
					return fmt.Errorf("update kv %q error: %s", topic, err)
				}
			} else {
				rev, err = putKeyValue(a.conn, kv, key, data)
				if err != nil {
					return fmt.Errorf("update kv %q error: %s", topic, err)
				}
			}

			if ctx.Sender() != nil {
				ctx.Respond(&gwiotmsg.AckKv{
					Rev: rev,
					Ids: msg.Id,
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
	case *gwiotmsg.EventPubSub:
		if err := func() error {
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			topic := msg.Subject
			headers := map[string]string{"id": utils.Hostname()}
			if err := publishPubSub(a.conn, topic, data, headers); err != nil {
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
		if ctx.Sender() == nil {
			break
		}
		if a.pidRemoteParent == nil || len(a.pidRemoteParent.GetId()) <= 0 || a.pidRemoteParent.GetId() != ctx.Sender().GetId() {
			a.pidRemoteParent = ctx.Sender()
			ctx.Watch(ctx.Sender())
		}
		// var uids string
		// uid, err := uuid.NewRandom()
		// if err != nil {
		// 	uids = fmt.Sprintf("%d", time.Now().UnixNano())
		// } else {
		// 	uids = uid.String()
		// }
		// a.subscriptions[uids] = RemoteSubscription{
		// 	Sender:  ctx.Sender(),
		// 	Message: msg,
		// }
		uids := msg.GetSubject()
		vRouter, ok := a.subscriptions[uids]
		if !ok {
			// Check if we already have a broadcast group for this subject
			subers, exists := a.subsGroupBroadcast[uids]
			if !exists {
				propsWriteChannel := router.NewBroadcastGroup()
				var err error
				subers, err = ctx.SpawnNamed(propsWriteChannel, fmt.Sprintf("subsGroup_%s", uids))
				if err != nil {
					logs.LogError.Printf("error spawn watcher (%q): %s", uids, err)
					if ctx.Sender() != nil {
						ctx.Respond(&gwiotmsg.Error{
							Error: err.Error(),
						})
					}
					break
				}
				a.subsGroupBroadcast[uids] = subers
			}
			vRouter = &RemoteSubscription{
				Sender:  ctx.Sender(), // ✅ Guardar el sender original, no el BroadcastGroup
				Message: msg,
			}
			a.subscriptions[uids] = vRouter
			ctx.Send(subers, &router.AddRoutee{
				PID: ctx.Sender(),
			})
		} else {
			ctx.Send(vRouter.Sender, &router.AddRoutee{
				PID: ctx.Sender(),
			})
		}
		if _, ok := a.subSubcripters[uids]; ok {
			// if err := w.Unsubscribe(); err != nil {
			// 	logs.LogWarn.Printf("subscripter %q error: %s", uids, err)
			// }
			// delete(a.subSubcripters, uids)
			delete(a.subscriptions, uids)
			break
		}
		subs, err := subscription(ctx, vRouter.Sender, a.conn, a.js, msg.GetSubject())
		if err != nil {
			logs.LogWarn.Println(err)
			// if ctx.Sender() != nil {
			// 	ctx.Respond(&gwiotmsg.Error{
			// 		Error: err.Error(),
			// 	})
			// }
			break
		}
		a.subSubcripters[uids] = subs
		delete(a.subscriptions, uids)
		// if ctx.Sender() != nil {
		// 	ctx.Respond(&gwiotmsg.Ack{})
		// }
		go func() {
			// subs := subs
			<-a.contxt.Done()
			if err := subs.Unsubscribe(); err != nil {
				logs.LogWarn.Println(err)
			}
		}()
	case *gwiotmsg.ListHistoryRevKv:
		if ctx.Sender() == nil {
			break
		}
		a.pidRemoteParent = ctx.Sender()
		bucket := a.addPrefix(ctx, msg.GetBucket())

		revs, err := listHistoryKv(a.conn, a.js, bucket, msg.Key)
		if err != nil {
			logs.LogWarn.Printf("listHistoryKV key: %s (%s), err: %s", msg.Key, bucket, err)
			ctx.Respond(&gwiotmsg.Error{
				Error: err.Error(),
			})
			break
		}
		ctx.Respond(&gwiotmsg.HistoryRevKv{
			Revisions: revs,
			Bucket:    bucket,
			Key:       msg.Key,
		})

	case *gwiotmsg.ListKeysBucket:
		if ctx.Sender() == nil {
			break
		}
		if len(msg.GetBucket()) <= 0 {
			ctx.Respond(&gwiotmsg.Error{
				Error: "bucket is empty",
			})
			break
		}

		a.pidRemoteParent = ctx.Sender()
		bucket := a.addPrefix(ctx, msg.GetBucket())

		keys, err := listKV(a.conn, a.js, bucket)
		fmt.Printf("keys (%q): %v, %T\n", keys, bucket, keys)
		ctx.Respond(&gwiotmsg.KeysBucket{
			Keys: keys,
			Error: func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		})
	case *gwiotmsg.GetKeyValue:
		if ctx.Sender() == nil {
			break
		}
		if len(msg.Key) <= 0 || len(msg.Bucket) <= 0 {
			ctx.Respond(&gwiotmsg.Error{
				Error: "key or bucket is empty",
			})
			break
		}
		a.pidRemoteParent = ctx.Sender()
		bucket := a.addPrefix(ctx, msg.GetBucket())

		entries, err := getKV(a.conn, a.js, bucket, msg.Key, msg.Rev, msg.GetIncludeHistory())
		if err != nil {
			logs.LogWarn.Printf("getKV key: %s (%s), err: %s", msg.Key, bucket, err)
			ctx.Respond(&gwiotmsg.Error{
				Error: err.Error(),
			})
			break
		}
		// fmt.Printf("keys (%q): %v, %T\n", keys, bucket, keys)
		result := make([]*gwiotmsg.KvEntryMessage, 0)
		for _, entry := range entries {
			result = append(result, &gwiotmsg.KvEntryMessage{
				Bucket: entry.Bucket(),
				Key:    entry.Key(),
				Rev:    entry.Revision(),
				Delta:  entry.Delta(),
				Op:     uint32(entry.Operation()),
				Data:   entry.Value(),
			})
		}
		if len(result) == 0 {
			ctx.Respond(&gwiotmsg.Error{
				Error: "without data",
			})
		} else if len(result) == 1 {
			ctx.Respond(result[0])
		} else {
			ctx.Respond(&gwiotmsg.KvEntriesMessage{
				Entries: result,
			})
		}
	case *gwiotmsg.WatchKeyValue:
		// fmt.Println("///////////////////////////////////////////////////////////////////////////////")
		// fmt.Printf("%T (%s): /////////////////////////////////////////////////////////////////////////// \n", msg, msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidRemoteParent == nil || len(a.pidRemoteParent.GetId()) <= 0 || a.pidRemoteParent.GetId() != ctx.Sender().GetId() {
			a.pidRemoteParent = ctx.Sender()
			ctx.Watch(ctx.Sender())
		}
		bucket := a.addPrefix(ctx, msg.GetBucket())

		// uids := fmt.Sprintf("%s-%s-%s", ctx.Sender().GetId(), bucket, msg.GetKey())
		// a.subscriptions[uids] = RemoteSubscription{
		// 	Sender:  ctx.Sender(),
		// 	Message: msg,
		// }
		uids := fmt.Sprintf("%s-%s", bucket, msg.GetKey())
		vRouter, ok := a.subscriptions[uids]
		if !ok {
			watchers, exists := a.subsGroupBroadcast[uids]
			if !exists {
				propsWriteChannel := router.NewBroadcastGroup()
				var err error
				watchers, err = ctx.SpawnNamed(propsWriteChannel, fmt.Sprintf("watcherGroup_%s", uids))
				if err != nil {
					logs.LogError.Printf("error spawn watcher (%q): %s", uids, err)
					if ctx.Sender() != nil {
						ctx.Respond(&gwiotmsg.Error{
							Error: err.Error(),
						})
					}
					break
				}
				a.subsGroupBroadcast[uids] = watchers
			}
			vRouter = &RemoteSubscription{
				Sender:  ctx.Sender(), // ✅ Guardar el sender original, no el BroadcastGroup
				Message: msg,
			}
			a.subscriptions[uids] = vRouter
			ctx.Send(watchers, &router.AddRoutee{
				PID: ctx.Sender(),
			})
		} else {
			ctx.Send(vRouter.Sender, &router.AddRoutee{
				PID: ctx.Sender(),
			})
		}
		if _, ok := a.subWatchers[uids]; ok {
			// if err := w.Stop(); err != nil {
			// 	logs.LogWarn.Printf("watcher %q error: %s", uids, err)
			// }
			// // Wait for the watcher context to be done to ensure proper cleanup
			// if w.Context() != nil {
			// 	select {
			// 	case <-w.Context().Done():
			// 		logs.LogInfo.Printf("watcher %q context properly closed", uids)
			// 	case <-time.After(5 * time.Second):
			// 		logs.LogWarn.Printf("watcher %q context close timeout", uids)
			// 	}
			// }
			// delete(a.subWatchers, uids)
			if a.conn == nil || !a.conn.IsConnected() || a.js == nil {
				logs.LogWarn.Printf("connection is not open (%v) (%v) (%v)", a.conn, a.js, func() bool { return a.conn != nil && a.conn.IsConnected() }())
				break
			}
			kv, err := a.js.KeyValue(bucket)
			if err != nil {
				logs.LogWarn.Printf("get key value (%q) error: %s", bucket, err)
			}
			if update, err := kv.Get(msg.GetKey()); err != nil {
				logs.LogWarn.Printf("get key value (key = %q) error: %s", msg.GetKey(), err)
			} else {
				ctx.Request(ctx.Sender(), &gwiotmsg.WatchMessage{
					KvEntryMessage: &gwiotmsg.KvEntryMessage{
						Bucket: update.Bucket(),
						Key:    update.Key(),
						Rev:    update.Revision(),
						Delta:  update.Delta(),
						Op:     uint32(update.Operation()),
						Data:   update.Value(),
					},
				})
			}
			delete(a.subscriptions, uids)
			break
		}
		subs, err := wathcKV(a.contxt, ctx, vRouter.Sender, a.conn, a.js, bucket, msg.GetKey(), msg.GetRev(), msg.GetIncludeHistory())
		if err != nil {
			logs.LogWarn.Printf("watchKeyValue error: %s", err)
			// if ctx.Sender() != nil {
			// 	ctx.Respond(&gwiotmsg.Error{
			// 		Error: err.Error(),
			// 	})
			// }
			break
		}
		a.subWatchers[uids] = subs
		delete(a.subscriptions, uids)
		// if ctx.Sender() != nil {
		// 	ctx.Respond(&gwiotmsg.Ack{})
		// }
		go func() {
			// subs := subs
			<-a.contxt.Done()
			fmt.Printf("stopping watch: %v\n", subs)
			// if err := subs.Unsubscribe(); err != nil {
			if err := subs.Stop(); err != nil {
				logs.LogWarn.Println(err)
			}
			// Wait for the watcher context to be done to ensure proper cleanup
			if subs.Context() != nil {
				select {
				case <-subs.Context().Done():
					logs.LogInfo.Printf("watcher %q context properly closed", uids)
				case <-time.After(3 * time.Second):
					logs.LogWarn.Printf("watcher %q context close timeout", uids)
				}
			}
			fmt.Printf("stopped watch: %v\n", subs)
		}()
	case *remote.EndpointTerminatedEvent:
		fmt.Printf("endpoint terminated \"%s\" (%s)\n", ctx.Self().GetId(), ctx.Parent())
	case *actor.Terminated:
		fmt.Printf("terminated  who: %q (self: %q) (parent: %q)\n", msg.GetWho().GetId(), ctx.Self().GetId(), ctx.Parent())
		if a.pidRemoteParent != nil && a.pidRemoteParent.GetId() == msg.GetWho().GetId() {
			a.pidRemoteParent = nil
			ctx.PoisonFuture(ctx.Self()).Wait()
		}
		if a.pidGwiot != nil && a.pidGwiot.GetId() == msg.GetWho().GetId() {
			a.pidGwiot = nil
			ctx.PoisonFuture(ctx.Self()).Wait()
		}
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")
	case *Resting:
		logs.LogError.Println("pidGwiot Resting, actor is about to rest")
		if a.cancel != nil {
			a.cancel()
		}
		ctx.PoisonFuture(ctx.Self()).Wait()
	default:
		fmt.Printf("unknown message in child natsio actor: %T, %s\n", msg, msg)
		if ctx.Sender() != nil {
			fmt.Printf("unknown message in child natsio actor: %T, %s, sender: %s\n", msg, msg, ctx.Sender().GetId())
		}
	}

}

type tickMsg struct{}

func tick(contxt context.Context, ctx actor.Context, timeout time.Duration) {

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()

	t_0 := time.NewTimer(3 * time.Second)
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
