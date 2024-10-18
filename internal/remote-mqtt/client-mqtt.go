package remqtt

import (
	"fmt"
	"time"

	"github.com/dumacp/go-gwiot/internal/utils"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/oauth2"
)

func client(tk *oauth2.Token, url, user, pass string) mqtt.Client {
	tlsconfig := utils.LoadLocalCert(utils.LocalCertDir)

	opt := mqtt.NewClientOptions()
	opt.AddBroker(url)
	opt.SetConnectRetry(true)
	opt.SetAutoReconnect(false)
	opt.SetConnectRetryInterval(10 * time.Second)
	if tk != nil {
		opt.SetPassword(tk.AccessToken)
		opt.SetUsername("unused")
	} else if len(user) > 0 {
		opt.SetPassword(pass)
		opt.SetUsername(user)
	}
	opt.SetProtocolVersion(protocolVersion)
	opt.SetOrderMatters(true)
	opt.SetTLSConfig(tlsconfig)

	opt.SetClientID(fmt.Sprintf("%s-%d", utils.Hostname(), time.Now().Unix()))
	opt.SetKeepAlive(30 * time.Second)
	opt.SetConnectTimeout(10 * time.Second)
	fmt.Printf("ops mqtt: %v\n", opt)
	client := mqtt.NewClient(opt)
	return client
}

func clientWithJwt(tk *oauth2.Token, url string) mqtt.Client {
	return client(tk, url, "", "")
}

func clientWithUserPass(url, user, pass string) mqtt.Client {
	return client(nil, url, user, pass)
}

func clientWithoutAuth(url string) mqtt.Client {
	return client(nil, url, "", "")
}

func connect(c mqtt.Client) error {
	tk := c.Connect()
	if !tk.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connect wait error")
	}
	if err := tk.Error(); err != nil {
		return err
	}
	return nil
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

	fmt.Printf("data to send in mqtt (%s): %s\n", topic, data)
	tk := client.Publish(topicSend, 0, false, data)

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
