package remqtt

import (
	"fmt"
	"time"

	"github.com/dumacp/go-gwiot/internal/utils"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/oauth2"
)

func client(tk *oauth2.Token) mqtt.Client {
	tlsconfig := utils.LoadLocalCert(utils.LocalCertDir)

	opt := mqtt.NewClientOptions()
	opt.AddBroker(utils.RemoteBrokerURL)
	opt.SetConnectRetry(true)
	opt.SetAutoReconnect(false)
	opt.SetConnectRetryInterval(10 * time.Second)
	if tk != nil {
		opt.SetPassword(tk.AccessToken)
		opt.SetUsername("unused")
	}
	opt.SetProtocolVersion(protocolVersion)
	opt.SetOrderMatters(true)
	opt.SetTLSConfig(tlsconfig)

	opt.SetClientID(fmt.Sprintf("%s-%d", utils.Hostname(), time.Now().Unix()))
	opt.SetKeepAlive(30 * time.Second)
	opt.SetConnectTimeout(10 * time.Second)
	client := mqtt.NewClient(opt)
	return client
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
