package renatsio

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/utils"
	"golang.org/x/oauth2"
)

func TokenSource(user, pass, keycloakUrl, realm, clientid, clientsecret string) (oauth2.TokenSource, error) {

	ctx_ := context.TODO()
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	tlsConfig := utils.LoadLocalCert(utils.LocalCertDir)

	cl := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       tlsConfig,
			Dial:                  (dialer).Dial,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	ctx := context.WithValue(ctx_, oauth2.HTTPClient, cl)
	return keyc.TokenSource(ctx, user, pass, keycloakUrl, realm, clientid, clientsecret)
}
