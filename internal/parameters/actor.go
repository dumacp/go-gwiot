package parameters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-logs/pkg/logs"
)

type Actor struct {
	id         string
	timeout    time.Duration
	httpClient *http.Client
	params     *PlatformParameters
	cancel     func()
}

func NewActor(id string) *Actor {
	a := &Actor{id: id}
	return a
}

func (a *Actor) Receive(ctx actor.Context) {
	fmt.Printf("Message arrived in %q: %T, %s\n",
		ctx.Self().GetId(), ctx.Message(), ctx.Sender())
	// logs.LogBuild.Printf("Message arrived in paramActor: %s, %T, %s",
	// 	ctx.Message(), ctx.Message(), ctx.Sender())
	switch ctx.Message().(type) {
	case *actor.Started:
		timeout := 300 * time.Second
		contxt, cancel := context.WithCancel(context.Background())
		a.cancel = cancel

		go tick(contxt, ctx, timeout)
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
	case *GetPlatformParameters:
		if ctx.Sender() == nil {
			break
		}
		if a.params != nil {
			ctx.Respond(a.params)
		}
	case *RequestPlatformParameters:
		if a.params != nil {
			break
		}
		if err := func() error {
			url := fmt.Sprintf("%s/%s/%s", utils.Url, Uri, utils.Hostname())
			resp, err := utils.Get(a.httpClient, url, utils.User, utils.PassCode, nil)
			if err != nil {
				return err
			}
			fmt.Printf("Get url: %s\n", url)
			fmt.Printf("Get response, GetParameters: %s\n", resp)
			var result PlatformParameters
			if err := json.Unmarshal(resp, &result); err != nil {
				return err
			}
			if len(result.ID) <= 0 {
				return fmt.Errorf("params error: null entity response -> %v", &result)
			}
			if result.Props == nil || len(result.Props.DEV_PID) <= 0 {
				return fmt.Errorf("params error: null entity response -> %v", &result)
			}
			a.params = &result
			fmt.Printf("Get response, GetParameters: %v\n", &result)
			ctx.Send(ctx.Parent(), &result)
			return nil
		}(); err != nil {
			fmt.Printf("get params error: %s\n", err)
			logs.LogError.Printf("get params error: %s", err)
		}

	}
}

func tick(contxt context.Context, ctx actor.Context, timeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			logs.LogError.Println("exit tickrReconnect()")
		} else {
			logs.LogError.Println("exit tickrReconnect()")
		}
	}()
	rootctx := ctx.ActorSystem().Root
	self := ctx.Self()
	t1 := time.NewTicker(timeout)
	defer t1.Stop()
	t0 := time.NewTimer(20 * time.Second)
	defer t0.Stop()
	for {
		select {
		case <-t1.C:
			rootctx.Send(self, &RequestPlatformParameters{})
		case <-t0.C:
			rootctx.Send(self, &RequestPlatformParameters{})
		case <-contxt.Done():
			return
		}
	}
}
