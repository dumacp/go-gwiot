package renatsio

import (
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/nats-io/nats.go"
)

func subscription(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, subject string, options ...nats.SubOpt) (*nats.Subscription, error) {

	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("connection is not open")
	}

	ctxroot := ctx.ActorSystem().Root
	sender := ctx.Sender()

	subs, err := js.Subscribe(subject, func(msg *nats.Msg) {

		headers := make(map[string]string)
		for k, v := range msg.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		if err := ctxroot.RequestFuture(sender, &gwiotmsg.SubscriptionMessage{
			Subject: msg.Subject,
			Reply:   msg.Reply,
			Headers: headers,
			Data:    msg.Data,
		}, 1*time.Second).Wait(); err != nil {
			if err := msg.Ack(); err != nil {
				fmt.Println(err)
			}
		}
	})
	return subs, err
}

func Subscription(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, subject string,
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

	return subscription(ctx, conn, js, subject,
		nats.MaxAckPending(int(maxAckPending)),
		nats.MaxDeliver(int(maxDeliver)),
		policy,
	)
}

func DurableSubscription(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, subject, durableName string,
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

	return subscription(ctx, conn, js, subject,
		nats.MaxAckPending(int(maxAckPending)),
		nats.MaxDeliver(int(maxDeliver)),
		policy,
		nats.Durable(durableName),
	)
}

// func wathcKV(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, bucket, key string) (*nats.Subscription, error) {
func wathcKV(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, bucket, key string) (nats.KeyWatcher, error) {

	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("connection is not open")
	}

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()
	sender := ctx.Sender()

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return nil, err
	}

	si, err := js.StreamInfo(fmt.Sprintf("KV_%s", bucket))
	if err != nil {
		return nil, err
	}
	fmt.Printf("/////////////////////// streamInfo: %v\n", si)
	fmt.Printf("/////////////////////// streamInfo: %v\n", si)
	fmt.Printf("/////////////////////// streamInfo: %v\n", si)

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

	watcher, err := kv.Watch(key,
		nats.AddIdleHeartbeat(30*time.Second),
		nats.MetaOnly(),
	)
	if err != nil {
		return nil, err
	}

	go func() {
		for v := range watcher.Updates() {
			if v == nil {
				fmt.Println("update nil!!!!!!!!!!!!!!!")
				continue
			}
			update, err := kv.GetRevision(v.Key(), v.Revision())
			if err != nil {
				fmt.Printf("update (key=%s,rev=%d) error: %s\n", v.Key(), v.Revision(), err)
			}

			ctxroot.RequestWithCustomSender(sender, &gwiotmsg.WatchMessage{
				KvEntryMessage: &gwiotmsg.KvEntryMessage{
					Bucket: update.Bucket(),
					Key:    update.Key(),
					Rev:    update.Revision(),
					Delta:  update.Delta(),
					Op:     uint32(update.Operation()),
					Data:   update.Value(),
				},
			}, self)
		}
	}()
	return watcher, err
}
