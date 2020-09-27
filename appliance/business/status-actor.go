package business

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
)

//StatusActor actor to listen events
type StatusActor struct {
	context       actor.Context
	lastStatusMsg *StatusMsg
}

//NewStatus create listen actor
func NewStatus() *StatusActor {
	act := &StatusActor{}
	return act
}

//Receive func Receive in actor
func (act *StatusActor) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
		if err := pubsub.Subscribe("STATUS/state", ctx.Self(), parseStatus); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}
	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)
	case *StatusMsg:
		status := msg.ReplaceKeys()
		act.lastStatusMsg = status
		onlychanges := status.OnlyChanges(act.lastStatusMsg)
		ctx.Send(ctx.Parent(), onlychanges)
	}
}
