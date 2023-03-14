package schsvc

import (
	"encoding/json"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/pubsub"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/pkg/messages"
	"github.com/dumacp/go-schservices/pkg/schservices"
)

type Actor struct{}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		d := &messages.Discover{
			Id:   ctx.Self().GetId(),
			Addr: ctx.Self().GetAddress(),
		}
		res, err := json.Marshal(d)
		if err != nil {
			logs.LogWarn.Printf("send discover error: %s", err)
			break
		}
		pubsub.Publish(schservices.SUBS_TOPIC, res)
	case *messages.ExternalService:
	case *messages.DiscoverResponse:
		pid := actor.NewPID(msg.Id, msg.Addr)
		ctx.Request(pid, &messages.Subscribe{})
	}
}
