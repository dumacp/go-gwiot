package localevents

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/pubsub"
	"github.com/dumacp/go-gwiot/pkg/events"
	"github.com/dumacp/go-logs/pkg/logs"
)

type Actor struct {
	ctx       actor.Context
	groupName string
}

func parseEvents(msg []byte) interface{} {

	event := new(events.Event)
	if err := json.Unmarshal(msg, event); err != nil {
		fmt.Printf("parse error in events -> %s", err)
		logs.LogWarn.Printf("parse error in events -> %s", err)
		return nil
	}

	return event
}

func parseState(msg []byte) interface{} {

	event := new(events.DeviceState)
	if err := json.Unmarshal(msg, event); err != nil {
		fmt.Printf("parse error in state -> %s", err)
		logs.LogWarn.Printf("parse error in state -> %s", err)
		return nil
	}

	return event
}

// Receive func Receive in actor
func (a *Actor) Receive(ctx actor.Context) {
	logs.LogBuild.Printf("Message arrived in %q: %T, %s",
		ctx.Self().GetId(), ctx.Message(), ctx.Sender())
	// logs.LogBuild.Printf("Message arrived in gpsActor: %s, %T, %s",
	// 	ctx.Message(), ctx.Message(), ctx.Sender())

	a.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
		if err := pubsub.Subscribe("EVENTS/#", ctx.Self(), parseEvents); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}
		if err := pubsub.Subscribe("STATUS/state", ctx.Self(), parseState); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)
	case *actor.Terminated:
		logs.LogWarn.Printf("\"%s\" - Terminated actor, reason -> %v", ctx.Self(), msg)
	case *events.Event:
		logs.LogBuild.Printf("Event arrived: %v", msg)
		mss := events.MessageFromEvent(msg)
		if ctx.Parent() != nil {
			ctx.Send(ctx.Parent(), mss)
		}
	case *events.DeviceState:
		logs.LogBuild.Printf("State arrived: %v", msg)
		sts := events.DeviceStateToDeviceSTS(msg)

		if len(a.groupName) <= 0 {
			if ctx.Parent() != nil {
				res, err := ctx.RequestFuture(ctx.Parent(), &GetGroupName{}, 300*time.Millisecond).Result()
				if err != nil {
					logs.LogError.Printf("error getting group name: %s", err)
				} else {
					if res, ok := res.(*GetGroupNameResponse); ok {
						a.groupName = res.GroupName
					}
				}
			}
		}
		mss := events.MessageFromDeviceSTS(a.groupName, sts)
		for _, m := range mss {
			if ctx.Parent() != nil {
				ctx.Send(ctx.Parent(), m)
			}
		}
	}
}
