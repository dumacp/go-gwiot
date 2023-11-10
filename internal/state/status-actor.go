package state

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/pubsub"
	"github.com/dumacp/go-gwiot/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

// StatusActor actor to listen events
type StatusActor struct {
	context       actor.Context
	lastStatusMsg *StatusMsg
	groupID       []byte
}

// NewStatus create listen actor
func NewStatus() *StatusActor {
	act := &StatusActor{}
	return act
}

// Receive func Receive in actor
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
		if v, ok := ((*status)["t"]).(float64); ok {
			(*status)["t"] = (int64)(v * 1000)
		}

		if len(act.groupID) <= 0 {
			groupID := groupID(ctx)
			if len(groupID) > 0 {
				act.groupID = groupID
			}
		}
		// log.Printf("groupId end -> %T, %v", act.groupID, act.groupID)
		if len(act.groupID) > 0 {
			(*status)["aD"] = map[string]interface{}{"gN": string(act.groupID)}
		}

		// log.Printf("status before -> %+v", status)

		onlychanges := status.OnlyChanges(act.lastStatusMsg)
		act.lastStatusMsg = status
		logs.LogBuild.Printf("status messages -> %+v", onlychanges)
		ctx.Send(ctx.Parent(), onlychanges)
	}
}

func groupID(ctx actor.Context) []byte {
	req := ctx.RequestFuture(ctx.Parent(), &messages.GroupIDRequest{}, 6*time.Second)
	if err := req.Wait(); err != nil {
		return nil
	}
	v, err := req.Result()
	if err != nil {
		return nil
	}
	// log.Printf("groupId -> %T, %v", v, v)
	if groupid, ok := v.(*messages.GroupIDResponse); ok {
		// log.Printf("groupId -> %T, %v", v, v)
		return groupid.GetData()
	}

	return nil
}
