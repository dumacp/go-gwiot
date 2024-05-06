package renatsio

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

const (
// subEvent = "Events.Offline"
)

func connect(urli string, tk *oauth2.Token) (*nats.Conn, error) {

	uri, err := url.Parse(urli)
	if err != nil {
		return nil, err
	}

	tlsconfig := utils.LoadLocalCert(utils.LocalCertDir)

	opts := make([]nats.Option, 0)
	if strings.Contains(uri.Scheme, "https") || strings.Contains(uri.Scheme, "wss") {
		opts = append(opts, nats.Secure(tlsconfig))
	}
	if tk != nil {

		opts = append(opts, nats.SetJwtBearer(func() string { return tk.AccessToken }))
	}

	dialer := &CustomDialer{
		Dialer: &net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		delay:   1 * time.Second,
		skipTLS: false,
	}

	newURL := uri.Scheme + "://" + uri.Host

	fmt.Printf("url: %s, path: %s\n", newURL, uri.Path)

	opts = append(opts, nats.SetCustomDialer(dialer))
	opts = append(opts, nats.Timeout(20*time.Second))
	opts = append(opts, nats.Compression(true))
	if len(uri.Path) > 0 {
		opts = append(opts, nats.ProxyPath(uri.Path))
	}

	conn, err := nats.Connect(newURL, opts...)
	if err != nil {
		fmt.Printf("%T, %s", err, err)
		return nil, err
	}

	return conn, nil
}

func connectWithJwt(url string, tk *oauth2.Token) (*nats.Conn, error) {
	return connect(url, tk)
}

func clientWithoutAuth(url string) (*nats.Conn, error) {
	return connect(url, nil)
}

func jsclient(conn *nats.Conn) (nats.JetStreamContext, error) {

	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}
	return js, nil
}

func kv(conn *nats.Conn, js nats.JetStreamContext, bucket string) (nats.KeyValue, error) {

	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("connection is not open")
	}

	return js.KeyValue(bucket)
}

// publish return (response?, error)
func publish(conn *nats.Conn, js nats.JetStreamContext, topic string, data []byte, headers map[string]string) error {

	if conn == nil || !conn.IsConnected() || js == nil {
		return fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)
	msg := nats.NewMsg(topic)
	msg.Header = make(nats.Header)
	msg.Data = data
	for k, v := range headers {
		msg.Header.Add(k, v)
	}

	var err error
	for range []int{0, 1, 2} {
		var ack *nats.PubAck
		ack, err = js.PublishMsg(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("msg ack: %v\n", ack)
		break
	}

	if err != nil {
		return fmt.Errorf("message dont send: %s", err)
	}

	return nil
}

// publishPubSub return (response?, error)
func publishPubSub(conn *nats.Conn, topic string, data []byte, headers map[string]string) error {

	if conn == nil || !conn.IsConnected() {
		return fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)
	msg := nats.NewMsg(topic)
	msg.Header = make(nats.Header)
	msg.Data = data
	for k, v := range headers {
		msg.Header.Add(k, v)
	}

	var err error
	for range []int{0, 1, 2} {
		err = conn.PublishMsg(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		break
	}

	if err != nil {
		return fmt.Errorf("message dont send: %s", err)
	}

	return nil
}

// requestPubSub return (response?, error)
func requestPubSub(conn *nats.Conn, topic string, data []byte, headers map[string]string, timeout time.Duration) (*nats.Msg, error) {

	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)
	msg := nats.NewMsg(topic)
	msg.Header = make(nats.Header)
	msg.Data = data
	for k, v := range headers {
		msg.Header.Add(k, v)
	}

	ctxChild, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	response, err := conn.RequestMsgWithContext(ctxChild, msg)
	if err != nil {
		return nil, fmt.Errorf("message dont send: %s", err)
	}

	return response, nil
}

// request return (response?, error)
func request(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, subject, reply string, data []byte, headers map[string]string, timeout time.Duration) error {

	if conn == nil || !conn.IsConnected() || js == nil {
		return fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)
	msg := nats.NewMsg(subject)
	msg.Header = make(nats.Header)
	msg.Data = data
	for k, v := range headers {
		msg.Header.Add(k, v)
	}

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()
	sender := ctx.Sender()

	contxt, cancel := context.WithCancel(context.TODO())

	ch := make(chan *nats.Msg)
	go func() {
		defer cancel()
		select {
		case <-contxt.Done():
		case msg, ok := <-ch:
			if !ok {
				fmt.Println("channel close in \"request\"")
				break
			}
			headers := make(map[string]string)
			if msg.Header != nil {
				for k, v := range msg.Header {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}
			}
			data := make([]byte, len(msg.Data))
			copy(data, msg.Data)
			ctxroot.RequestWithCustomSender(sender, &gwiotmsg.SubscriptionMessage{
				Subject: msg.Subject,
				Reply:   msg.Reply,
				Headers: headers,
				Data:    data,
			}, self)
			if err := msg.Sub.Unsubscribe(); err != nil {
				fmt.Println(err)
			}
			if err := msg.Ack(); err != nil {
				fmt.Println(err)
			}

		}
	}()

	subs, err := js.ChanSubscribe(reply, ch, nats.DeliverNew())
	if err != nil {
		close(ch)
		return fmt.Errorf("message dont send: %s", err)
	}

	go func(timeout time.Duration) {

		if timeout <= 0 {
			timeout = 1 * time.Second
		}

		tout := time.NewTimer(timeout)
		defer tout.Stop()

		select {
		case <-tout.C:
			if err := subs.Unsubscribe(); err != nil {
				fmt.Printf("unsubscribe error: %s\n", err)
			}
			fmt.Printf("timeout in subscription with reply: %s\n", reply)
		case <-contxt.Done():
		}
	}(timeout)

	for range []int{0, 1, 2} {
		// var ack *nats.PubAck
		if _, err = js.PublishMsg(msg); err != nil {
			fmt.Println(err)
			continue
		}
		// fmt.Printf("msg ack: %v\n", ack)
		break
	}
	if err != nil {
		return fmt.Errorf("message dont send: %s", err)
	}

	return nil
}

// updateKeyValue return (response?, error)
func updateKeyValue(conn *nats.Conn, kv nats.KeyValue, key string, data []byte, lastRevision uint64) (uint64, error) {

	if conn == nil || !conn.IsConnected() {
		return 0, fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)

	var rev uint64
	var err error
	for range []int{0, 1, 2} {
		rev, err = kv.Update(key, data, lastRevision)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}

	if err != nil {
		return 0, fmt.Errorf("message dont send: %s", err)
	}

	return rev, nil
}

// putKeyValue return (response?, error)
func putKeyValue(conn *nats.Conn, kv nats.KeyValue, key string, data []byte) (uint64, error) {

	if conn == nil || !conn.IsConnected() {
		return 0, fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)

	rev, err := kv.Put(key, data)

	if err != nil {
		return 0, fmt.Errorf("message dont send: %s", err)
	}

	return rev, nil
}

func getKV(ctx actor.Context, conn *nats.Conn, js nats.JetStreamContext, bucket, key string, rev uint64) error {

	if conn == nil || !conn.IsConnected() {
		return fmt.Errorf("connection is not open")
	}

	kv, err := js.KeyValue(bucket)
	if err != nil {
		return err
	}

	entry, err := kv.GetRevision(key, uint64(rev))
	if err != nil {
		return err
	}

	ctx.Respond(&gwiotmsg.KvEntryMessage{
		Bucket: entry.Bucket(),
		Key:    entry.Key(),
		Rev:    entry.Revision(),
		Delta:  entry.Delta(),
		Op:     uint32(entry.Operation()),
		Data:   entry.Value(),
	})

	return nil

}

func ack(conn *nats.Conn, js nats.JetStreamContext, subject, reply string, data []byte, headers map[string]string, timeout time.Duration) error {

	if conn == nil || !conn.IsConnected() {
		return fmt.Errorf("connection is not open")
	}

	logs.LogBuild.Printf("data to send: %s", data)

	return nil
}
