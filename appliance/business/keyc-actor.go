package business

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	"github.com/dumacp/keycloak"
	"golang.org/x/oauth2"
)

//KeycloakActor keycloak actor
type KeycloakActor struct {
	username    string
	password    string
	httpContext context.Context
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	keyc        keycloak.Keycloak
	httContext  context.Context
}

//NewKeycActor create keycloak actor
func NewKeycActor(usernmae, password string) *KeycloakActor {
	act := &KeycloakActor{username: usernmae, password: password}
	return act
}

//Receive function Receive
func (act *KeycloakActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
	case *messages.TokenRequest:
		logs.LogBuild.Printf("request message -> %s", msg)
		var tk *oauth2.Token
		var err error
		if act.tokenSource != nil {
			tk, err = act.tokenSource.Token()
			if err == nil {
				ctx.Send(ctx.Sender(), tk)
				break
			}
			logs.LogWarn.Printf("token request error -> %s", err)
		}

		if act.httContext == nil {
			act.httpContext = newHTTPContext(context.Background(), act.httpClient)
		}
		if act.keyc == nil {
			config := newKeyConfig()
			act.keyc, err = keycServer(act.httpContext, config)
			if err != nil {
				logs.LogError.Printf("error getting token -> %s", err)
				break
			}
		}

		ts, err := tokenRequest(act.httpContext, act.keyc, act.username, act.password)
		if err != nil {
			logs.LogError.Printf("token request error -> %s", err)
			break
		}
		act.tokenSource = ts
		ctx.Send(ctx.Sender(), tk)

	case *messages.GroupIDRequest:
		var err error
		if act.httContext == nil {
			act.httpContext = newHTTPContext(context.Background(), act.httpClient)
		}
		if act.keyc == nil {
			config := newKeyConfig()
			act.keyc, err = keycServer(act.httpContext, config)
			if err != nil {
				logs.LogError.Printf("error getting token -> %s", err)
				break
			}
		}
		if act.tokenSource != nil {
			_, err = act.tokenSource.Token()
			if err != nil {
				act.tokenSource, err = tokenRequest(act.httpContext, act.keyc, act.username, act.password)
				if err != nil {
					logs.LogError.Printf("error getting token -> %s", err)
					break
				}
			}
		} else {
			act.tokenSource, err = tokenRequest(act.httpContext, act.keyc, act.username, act.password)
			if err != nil {
				logs.LogError.Printf("error getting userInfo -> %s", err)
				break
			}
		}

		userInfo, err := act.keyc.UserInfo(act.httpContext, act.tokenSource)
		if err != nil {
			logs.LogError.Printf("error getting userInfo -> %s", err)
			break
		}
		logs.LogBuild.Printf("userInfo: %#v", userInfo)
		groupid, ok := userInfo["group_name"]
		if !ok {
			logs.LogError.Printf("no groupID property xxx")
			break
		}
		ctx.Send(ctx.Sender(), &messages.GroupIDResponse{Data: []byte(fmt.Sprintf("%v", groupid))})

	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)

	}

}

func tokenRequest(ctx context.Context, keyc keycloak.Keycloak, username, password string) (oauth2.TokenSource, error) {

	ts, err := tokenSorce(ctx, keyc, username, password)
	if err != nil {
		return nil, fmt.Errorf("error getting token -> %s", err)
	}

	return ts, nil
}
