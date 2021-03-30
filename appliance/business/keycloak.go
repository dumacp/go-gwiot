package business

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	"github.com/dumacp/keycloak"
	"github.com/dumacp/utils"
	"golang.org/x/oauth2"
)

const (
	localCertDir = "/usr/local/share/ca-certificates/"
	// ClientSecret = "da9bbc28-01d8-43af-8c8a-fb0654937231"

	pathenvfile = "/usr/include/serial-dev"
)

var (
	Clientid     = "devices2"
	ClientSecret = "b73479a3-225b-4b96-ad65-22edd82623a3"
	Redirecturl  = "https://fleet-mqtt.nebulae.com.co/"
	Keycloakurl  = "https://fleet.nebulae.com.co/auth"
	Realm        = "DEVICES"
)

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

func newHTTPContext(ctx context.Context) (context.Context, *tls.Config) {
	client, tlsconfig := utils.LoadLocalCert(localCertDir)
	ctx = keycloak.NewClientContext(ctx, client)
	return ctx, tlsconfig
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
