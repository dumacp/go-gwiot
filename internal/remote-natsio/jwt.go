package renatsio

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/utils"
	"golang.org/x/oauth2"
)

func Oauth2Config(ctx context.Context, url, realm, clientid, clientsecret string) (*oauth2.Config, error) {
	return keyc.Oauth2Config(ctx, url, realm, clientid, clientsecret)
}

func ContextWithHTTPClient(ctx context.Context) context.Context {
	tlsConfig, _ := utils.LoadLocalCert()

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	cl := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       tlsConfig,
			Dial:                  (dialer).Dial,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 3 * time.Second,
		},
	}

	return context.WithValue(ctx, oauth2.HTTPClient, cl)
}

func TokenSource(ctx context.Context, config *oauth2.Config, url, realm, user, pass string) (oauth2.TokenSource, error) {

	// dialer := &net.Dialer{
	// 	Timeout:   30 * time.Second,
	// 	KeepAlive: 30 * time.Second,
	// }

	// tlsConfig, _ := utils.LoadLocalCert()

	// cl := &http.Client{
	// 	Transport: &http.Transport{
	// 		TLSClientConfig:       tlsConfig,
	// 		Dial:                  (dialer).Dial,
	// 		TLSHandshakeTimeout:   20 * time.Second,
	// 		ExpectContinueTimeout: 3 * time.Second,
	// 	},
	// }

	// ctxx := context.WithValue(ctx, oauth2.HTTPClient, cl)
	return keyc.TokenSource(ctx, config, url, realm, user, pass)
}

func UserInfo(ctx context.Context, ts oauth2.TokenSource, url, realm string) (*oidc.UserInfo, error) {
	return keyc.UserInfo(ctx, ts, url, realm)
}
