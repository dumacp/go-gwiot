package renatsio

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/nats-io/nats.go"
)

func subscription(ctx actor.Context, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject string, options ...nats.SubOpt) (*nats.Subscription, error) {

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

func Subscription(ctx actor.Context, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject string,
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

func DurableSubscription(ctx actor.Context, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, subject, durableName string,
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
	fmt.Printf("/////////////// streamInfo: %v\n", si)

	opts := make([]nats.WatchOpt, 0)

	opts = append(opts, nats.AddIdleHeartbeat(30*time.Second))
	opts = append(opts, nats.MetaOnly())

	return kv.Keys(opts...)

}

func listHistoryKv(conn *nats.Conn, js nats.JetStreamContext, bucket, key string) ([]uint64, error) {

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
	fmt.Printf("/////////////// streamInfo: %v\n", si)

	opts := make([]nats.WatchOpt, 0)

	opts = append(opts, nats.AddIdleHeartbeat(30*time.Second))
	opts = append(opts, nats.MetaOnly())

	entries, err := kv.History(key, opts...)
	if err != nil {
		return nil, err
	}

	revisions := make([]uint64, 0)

	for _, entry := range entries {
		revisions = append(revisions, entry.Revision())
	}

	return revisions, nil

}

// func wathcKV(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, bucket, key string) (*nats.Subscription, error) {
func wathcKV(contxt context.Context, ctx actor.Context, sender *actor.PID, conn *nats.Conn, js nats.JetStreamContext, bucket, key string, rev uint64, history bool) (nats.KeyWatcher, error) {

	if conn == nil || !conn.IsConnected() || js == nil {
		return nil, fmt.Errorf("connection is not open (%v) (%v) (%v)", conn, js, func() bool { return conn != nil && conn.IsConnected() }())
	}

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return nil, err
	}

	si, err := js.StreamInfo(fmt.Sprintf("KV_%s", bucket))
	if err != nil {
		return nil, err
	}
	fmt.Printf("/////////////// watch streamInfo: %v\n", si)

	opts := make([]nats.WatchOpt, 0)

	opts = append(opts, nats.AddIdleHeartbeat(60*time.Second))
	opts = append(opts, nats.MetaOnly())
	if contxt != nil {
		opts = append(opts, nats.Context(contxt))
	} else {
		opts = append(opts, nats.Context(context.Background()))
	}
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
		// for v := range watcher.Updates() {
		revfor := rev
		for {
			select {
			case <-watcher.Context().Done():
				fmt.Println("watcher context done")
				return
			case v, ok := <-watcher.Updates():
				if !ok {
					fmt.Println("watcher updates channel closed")
					return
				}
				if v == nil {
					fmt.Println("update nil!!!!!!!!!!!!!!!")
					continue
				}
				fmt.Printf("update: %v\n", v)
				if revfor > 0 && v.Revision() <= revfor {
					continue
				}
				revfor = v.Revision()
				update, err := kv.GetRevision(v.Key(), v.Revision())
				if err != nil {
					fmt.Printf("update (key=%s,rev=%d) error: %s\n", v.Key(), v.Revision(), err)
					continue
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
		}
	}()
	return watcher, err
}
