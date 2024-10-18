package keyc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/keycloak"
	"golang.org/x/oauth2"
)

var (
	Clientid     string
	ClientSecret string
	Redirecturl  string
	Keycloakurl  string
	Realm        string
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

func newHTTPContext(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		tlsConfig, _ := utils.LoadLocalCert()
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
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

func Oauth2Config(ctx context.Context, url, realm, clientid, clientsecret string) (*oauth2.Config, error) {

	issuer := fmt.Sprintf("%s/realms/%s", url, realm)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	config := &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: clientsecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  url,
		Scopes:       []string{oidc.ScopeOpenID},
	}
	return config, nil
}

func TokenSource(ctx context.Context, config *oauth2.Config, url, realm, username, password string) (oauth2.TokenSource, error) {

	// config := &oauth2.Config{
	// 	ClientID:     clientid,
	// 	ClientSecret: clientsecret,
	// 	Endpoint:     provider.Endpoint(),
	// 	RedirectURL:  url,
	// 	Scopes:       []string{oidc.ScopeOpenID},
	// }

	tk, err := config.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, err
	}

	ts := config.TokenSource(ctx, tk)

	return ts, nil

}

func UserInfo(ctx context.Context, ts oauth2.TokenSource, url, realm string) (*oidc.UserInfo, error) {
	issuer := fmt.Sprintf("%s/realms/%s", url, realm)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	userInfo, err := provider.UserInfo(ctx, ts)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}
