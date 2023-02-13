package renatsio

import (
	"fmt"

	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

const (
	subEvent = "Events.Offline"
)

func connect(tk *oauth2.Token) (*nats.Conn, error) {
	tlsconfig := utils.LoadLocalCert(utils.LocalCertDir)

	optTls := nats.Secure(tlsconfig)

	optJwt := nats.SetJwtBearer(func() string { return tk.AccessToken })

	conn, err := nats.Connect(utils.RemoteBrokerURL, optTls, optJwt)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func client(conn *nats.Conn) (nats.JetStreamContext, error) {

	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}
	// stream := nats.StreamConfig{
	// 	Name:        "Events",
	// 	Subjects:    []string{subEvent},
	// 	Description: "stream events",
	// 	Retention:   nats.LimitsPolicy,
	// 	Discard:     nats.DiscardNew,
	// 	// Storage:     nats.MemoryStorage,
	// 	// MaxMsgsPerSubject: 5,
	// 	// MaxBytes: 256,
	// 	MaxMsgs: 10,
	// }

	// info, err := js.AddStream(&stream)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Printf("stream info: %+v\n", info)

	return js, nil
}

//sendMSG return (response?, error)
func sendMSG(conn *nats.Conn, topic string, data []byte, headers map[string]string, test bool) (bool, error) {

	if conn == nil || !conn.IsConnected() {
		return true, fmt.Errorf("connection is not open")
	}
	client, err := client(conn)
	if err != nil {
		return false, err
	}
	var topicSend string
	if test {
		topicSend = fmt.Sprintf(remoteQueueEventsTest, utils.Hostname())
	} else {
		topicSend = topic
	}

	logs.LogBuild.Printf("data to send: %s", data)
	msg := nats.NewMsg(topicSend)
	msg.Header = make(nats.Header)
	msg.Data = data
	for k, v := range headers {
		msg.Header.Add(k, v)
	}

	for range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		var ack *nats.PubAck
		ack, err = client.PublishMsg(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("msg ack: %v\n", ack)
		break
	}

	if err != nil {
		return false, fmt.Errorf("message dont send: %s", err)
	}

	return true, nil
}
