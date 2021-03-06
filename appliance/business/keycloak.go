package business

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	"github.com/dumacp/keycloak"
	"github.com/dumacp/utils"
	"golang.org/x/oauth2"
)

const (
	localCertDir = "/usr/local/share/ca-certificates/"
	keycloakurl  = "https://fleet.nebulae.com.co/auth"
	clientid     = "devices3"
	clientSecret = "da9bbc28-01d8-43af-8c8a-fb0654937231"
	redirecturl  = "https://fleet-mqtt.nebulae.com.co/"
	realm        = "DEVICES"

	pathenvfile = "/usr/include/serial-dev"
)

func newKeyConfig() *keycloak.ServerConfig {

	return &keycloak.ServerConfig{
		Url:          keycloakurl,
		ClientID:     clientid,
		ClientSecret: clientSecret,
		RedirectUrl:  redirecturl,
		Realm:        realm,
	}
}

func newHTTPContext(ctx context.Context) (context.Context, *tls.Config) {
	client, tlsconfig := utils.LoadLocalCert(localCertDir)
	ctx = keycloak.NewClientContext(ctx, client)
	return ctx, tlsconfig
}

func tokenSorce(ctx context.Context, config *keycloak.ServerConfig, username, password string) (oauth2.TokenSource, error) {
	keyc, err := keycloak.NewConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	log.Println("1")
	tk, err := keyc.TokenRequest(ctx, username, password)
	if err != nil {
		return nil, err
	}
	log.Println("2")
	ts := keyc.TokenSource(ctx, tk)
	if ts == nil {
		return nil, fmt.Errorf("tokenSource nil")
	}
	log.Println("3")
	return ts, nil
}
