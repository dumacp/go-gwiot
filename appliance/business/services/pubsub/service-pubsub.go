package pubsub

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-modem/appliance/business/messages"
	svc "github.com/dumacp/go-modem/appliance/business/services"
	"github.com/dumacp/go-modem/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-modem/appliance/crosscutting/logs"
	"github.com/dumacp/go-modem/appliance/services"
)

//Gateway interface
type Gateway interface {
	Receive(ctx actor.Context)
}

type pubsubActor struct {
	svc services.Service
	ctx actor.Context
}

//NewService create Service actor
func NewService() Gateway {
	act := &pubsubActor{svc: svc.GetInstance()}
	return act
}

func service(ctx actor.Context) {
	pubsub.Subscribe(pubsub.TopicStart, ctx.Self(), func(msg []byte) interface{} {
		return &messages.Start{}
	})
	pubsub.Subscribe(pubsub.TopicStop, ctx.Self(), func(msg []byte) interface{} {
		return &messages.Stop{}
	})
	pubsub.Subscribe(pubsub.TopicRestart, ctx.Self(), func(msg []byte) interface{} {
		return &messages.Restart{}
	})
	pubsub.Subscribe(pubsub.TopicStatus, ctx.Self(), func(msg []byte) interface{} {
		req := &messages.StatusRequest{}
		if err := req.Unmarshal(msg); err != nil {
			logs.LogWarn.Println(err)
			return nil
		}
		return req
	})

}

//Receive function
func (act *pubsubActor) Receive(ctx actor.Context) {
	act.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *messages.Start:
		act.svc.Start()
	case *messages.Stop:
		act.svc.Stop()
	case *messages.Restart:
		act.svc.Restart()
	case *messages.StatusRequest:
		resp := act.svc.Status()
		payload, err := resp.Marshal()
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		pubsub.Publish(fmt.Sprintf("%s/%s", pubsub.TopicStatus, msg.GetSender()), payload)
	}
}
