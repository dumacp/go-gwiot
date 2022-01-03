package business

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/keycloak"
	"golang.org/x/oauth2"
)

const (
	localCertDir = "/usr/local/share/ca-certificates/"
	// ClientSecret = "da9bbc28-01d8-43af-8c8a-fb0654937231"

	pathenvfile = "/usr/include/serial-dev"
)

var (
	Clientid     string
	ClientSecret string
	Redirecturl  string
	Keycloakurl  string
	Realm        string
)

func init() {
	flag.StringVar(&Keycloakurl, "keycloakUrl", "", "example: \"https://fleet.nebulae.com.co/auth\", keycloak url")
	flag.StringVar(&Clientid, "clientID", "", "example: \"devices2\", clientid in realm")
	flag.StringVar(&ClientSecret, "clientSecret", "", "example: \"b73479a3-225b-4b96-ad65-22edd82623a3\", client secret")
	flag.StringVar(&Redirecturl, "redirectUrl", "", "example: \"https://fleet-mqtt.nebulae.com.co/\", redirecturl url")
	flag.StringVar(&Realm, "realm", "", "example: \"DEVICES\", realm name")
}

func newKeyConfig() *keycloak.ServerConfig {

	logs.LogBuild.Printf("keycloak url: %s", Keycloakurl)
	return &keycloak.ServerConfig{
		Url:          Keycloakurl,
		ClientID:     Clientid,
		ClientSecret: ClientSecret,
		RedirectUrl:  Redirecturl,
		Realm:        Realm,
	}
}

func newHTTPContext(ctx context.Context, client *http.Client) context.Context {
	// client, tlsconfig := utils.LoadLocalCert(localCertDir)
	if client == nil {
		tlsConfig := LoadLocalCert(localCertDir)
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
			Dial: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			// Dial: (&net.Dialer{
			// 	Timeout:   30 * time.Second,
			// 	KeepAlive: 60 * time.Second,
			// }).Dial,
			// TLSHandshakeTimeout:   10 * time.Second,
			// ResponseHeaderTimeout: 10 * time.Second,
			// ExpectContinueTimeout: 3 * time.Second,
		}
		client = &http.Client{Transport: tr}
		client.Timeout = 30 * time.Second
	}
	ctx = keycloak.NewClientContext(ctx, client)
	return ctx
}

func keycServer(ctx context.Context, config *keycloak.ServerConfig) (keycloak.Keycloak, error) {
	keyc, err := keycloak.NewConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return keyc, nil
}

func tokenSorce(ctx context.Context, keyc keycloak.Keycloak, username, password string) (oauth2.TokenSource, error) {
	// keyc, err := keycloak.NewConfig(ctx, config)
	// if err != nil {
	// 	return nil, err
	// }
	// log.Println("1")
	tk, err := keyc.TokenRequest(ctx, username, password)
	if err != nil {
		return nil, err
	}
	ts := keyc.TokenSource(ctx, tk)
	if ts == nil {
		return nil, fmt.Errorf("tokenSource nil")
	}
	return ts, nil
}
