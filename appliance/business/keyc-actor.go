package business

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	"golang.org/x/oauth2"
)

//KeycloakActor keycloak actor
type KeycloakActor struct {
	username    string
	password    string
	httpContext context.Context
	tokenSource oauth2.TokenSource
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
		tk, err := tokenRequest(act.tokenSource, act.username, act.password)
		if err != nil {
			logs.LogWarn.Printf("token request error -> %s", err)
			break
		}
		ctx.Send(ctx.Sender(), tk)

	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)

	}

}

func tokenRequest(cachets oauth2.TokenSource, username, password string) (*oauth2.Token, error) {
	if cachets == nil {
		config := newKeyConfig()
		httpContext, _ := newHTTPContext(context.Background())
		ts, err := tokenSorce(httpContext, config, username, password)
		if err != nil {
			logs.LogError.Panicf("error getting token -> %s", err)
		}
		tk, err := ts.Token()
		if err != nil {
			logs.LogWarn.Printf("token error -> %s", err)
			return nil, err
		}
		cachets = ts
		return tk, nil
	}

	tk, err := cachets.Token()
	if err != nil {
		logs.LogWarn.Printf("token error -> %s", err)
		return nil, err
	}

	config := newKeyConfig()
	httpContext, _ := newHTTPContext(context.Background())
	ts, err := tokenSorce(httpContext, config, username, password)
	if err != nil {
		logs.LogError.Panicf("error getting token -> %s", err)
		return nil, err
	}
	tk, err = ts.Token()
	if err != nil {
		logs.LogWarn.Printf("token error -> %s", err)
	}
	cachets = ts

	return tk, nil
}
