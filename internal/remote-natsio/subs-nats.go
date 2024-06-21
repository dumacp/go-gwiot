package renatsio

import (
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/nats-io/nats.go"
)

func subscription(ctx *actor.RootContext, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject string, options ...nats.SubOpt) (*nats.Subscription, error) {

	if conn == nil || !conn.IsConnected() || js == nil {
		return nil, fmt.Errorf("connection is not open")
	}

	// ctxroot := ctx.ActorSystem().Root
	// sender := ctx.Sender()

	subs, err := js.Subscribe(subject, func(msg *nats.Msg) {

		headers := make(map[string]string)
		for k, v := range msg.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		if err := ctx.RequestFuture(sender, &gwiotmsg.SubscriptionMessage{
			Subject: msg.Subject,
			Reply:   msg.Reply,
			Headers: headers,
			Data:    msg.Data,
		}, 1*time.Second).Wait(); err == nil {
			if err := msg.Ack(); err != nil {
				fmt.Println(err)
			}
		}
	})
	return subs, err
}

func Subscription(ctx *actor.RootContext, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject string,
	startSeq uint64, startTime time.Time, maxDeliver, maxAckPending uint, deliverPolicy gwiotmsg.DeliverPolicy) (*nats.Subscription, error) {

	policy := func() nats.SubOpt {
		switch deliverPolicy {
		case gwiotmsg.DeliverPolicy_DeliverAll:
			return nats.DeliverAll()
		case gwiotmsg.DeliverPolicy_DeliverLast:
			return nats.DeliverLast()
		case gwiotmsg.DeliverPolicy_DeliverNew:
			return nats.DeliverLast()
		case gwiotmsg.DeliverPolicy_DeliverByStartTime:
			return nats.StartTime(startTime)
		case gwiotmsg.DeliverPolicy_DeliverByStartSeq:
			return nats.StartSequence(startSeq)
		default:
			return nats.DeliverAll()
		}
	}()

	return subscription(ctx, sender, conn, js, subject,
		nats.MaxAckPending(int(maxAckPending)),
		nats.MaxDeliver(int(maxDeliver)),
		policy,
	)
}

func DurableSubscription(ctx *actor.RootContext, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject, durableName string,
	startSeq uint64, startTime time.Time, maxDeliver, maxAckPending uint, deliverPolicy gwiotmsg.DeliverPolicy) (*nats.Subscription, error) {

	policy := func() nats.SubOpt {
		switch deliverPolicy {
		case gwiotmsg.DeliverPolicy_DeliverAll:
			return nats.DeliverAll()
		case gwiotmsg.DeliverPolicy_DeliverLast:
			return nats.DeliverLast()
		case gwiotmsg.DeliverPolicy_DeliverNew:
			return nats.DeliverLast()
		case gwiotmsg.DeliverPolicy_DeliverByStartTime:
			return nats.StartTime(startTime)
		case gwiotmsg.DeliverPolicy_DeliverByStartSeq:
			return nats.StartSequence(startSeq)
		default:
			return nats.DeliverAll()
		}
	}()

	return subscription(ctx, sender, conn, js, subject,
		nats.MaxAckPending(int(maxAckPending)),
		nats.MaxDeliver(int(maxDeliver)),
		policy,
		nats.Durable(durableName),
	)
}

// func listKV(conn *nats.Conn, js nats.JetStreamContext, bucket) (*nats.Subscription, error) {
func listKV(conn *nats.Conn, js nats.JetStreamContext, bucket string) ([]string, error) {

	if conn == nil || !conn.IsConnected() || js == nil {
		return nil, fmt.Errorf("connection is not open (%v) (%v) (%v)", conn, js, func() bool { return conn != nil && conn.IsConnected() }())
	}

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return nil, err
	}

	si, err := js.StreamInfo(fmt.Sprintf("KV_%s", bucket))
	if err != nil {
		return nil, err
	}
	fmt.Printf("/////////////////////// streamInfo: %v\n", si)

	opts := make([]nats.WatchOpt, 0)

	opts = append(opts, nats.AddIdleHeartbeat(30*time.Second))
	opts = append(opts, nats.MetaOnly())

	return kv.Keys(opts...)

}

// func wathcKV(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, bucket, key string) (*nats.Subscription, error) {
func wathcKV(ctx *actor.RootContext, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, bucket, key string, rev uint64, history bool) (nats.KeyWatcher, error) {

	if conn == nil || !conn.IsConnected() || js == nil {
		return nil, fmt.Errorf("connection is not open (%v) (%v) (%v)", conn, js, func() bool { return conn != nil && conn.IsConnected() }())
	}

	// ctxroot := ctx.ActorSystem().Root
	// self := ctx.Self()
	// sender := ctx.Sender()

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return nil, err
	}

	si, err := js.StreamInfo(fmt.Sprintf("KV_%s", bucket))
	if err != nil {
		return nil, err
	}
	fmt.Printf("/////////////////////// watch streamInfo: %v\n", si)

	// sub, err := js.Subscribe(fmt.Sprintf("$KV.FMS-DEV-ROUTES.%s", key), func(msg *nats.Msg) {
	// 	fmt.Printf("PUB DATA: %s (%v)\n", msg.Data, msg.Header)
	// 	ctxroot.RequestWithCustomSender(sender, &gwiotmsg.WatchMessage{
	// 		KvEntryMessage: &gwiotmsg.KvEntryMessage{
	// 			Bucket: bucket,
	// 			Key:    key,
	// 			Rev:    0,
	// 			Delta:  0,
	// 			Op:     0,
	// 			Data:   msg.Data,
	// 		},
	// 	}, self)
	// 	msg.Ack()
	// },
	// 	nats.DeliverLastPerSubject(),
	// 	nats.OrderedConsumer(),
	// 	nats.IdleHeartbeat(30*time.Second),
	// )
	// if err != nil {
	// 	return nil, err
	// }
	// return sub, nil

	opts := make([]nats.WatchOpt, 0)

	opts = append(opts, nats.AddIdleHeartbeat(60*time.Second))
	opts = append(opts, nats.MetaOnly())
	if history {
		opts = append(opts, nats.IncludeHistory())
	}

	if len(key) <= 0 {
		kv.Keys(opts...)
	}

	watcher, err := kv.Watch(key, opts...)
	if err != nil {
		return nil, err
	}

	go func() {
		for v := range watcher.Updates() {
			if v == nil {
				fmt.Println("update nil!!!!!!!!!!!!!!!")
				continue
			}
			fmt.Printf("update: %v\n", v)
			if rev > 0 && v.Revision() <= rev {
				continue
			}
			update, err := kv.GetRevision(v.Key(), v.Revision())
			if err != nil {
				fmt.Printf("update (key=%s,rev=%d) error: %s\n", v.Key(), v.Revision(), err)
				continue
			}

			ctx.Request(sender, &gwiotmsg.WatchMessage{
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
	}()
	return watcher, err
}
