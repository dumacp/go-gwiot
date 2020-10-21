package grpc

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-modem/appliance/business/messages"
	svc "github.com/dumacp/go-modem/appliance/business/services"
	"github.com/dumacp/go-modem/appliance/services"
)

//Gateway interface
type Gateway interface {
	Receive(ctx actor.Context)
}

type grpcActor struct {
	svc services.Service
	// ctx actor.Context
}

//NewService create Service actor
func NewService() Gateway {
	act := &grpcActor{svc: svc.GetInstance()}
	//TODO:
	return act
}

//Receive function
func (act *grpcActor) Receive(ctx actor.Context) {
	// act.ctx = ctx
	switch ctx.Message().(type) {
	case *actor.Started:
		// receptionist.
	case *messages.Start:
		act.svc.Start()
	case *messages.Stop:
		act.svc.Stop()
	case *messages.Restart:
		act.svc.Restart()
	case *messages.StatusRequest:
		msg := act.svc.Status()
		ctx.Respond(msg)
	}
}
