package business

import (
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-logs/pkg/logs"
)

//EventsActor actor to listen events
type EventsActor struct {
	ctx actor.Context
	// quit    chan int
	timeout time.Duration
	events  Events
	mux     sync.Mutex
}

//NewEvents create listen actor
func NewEvents(timeout time.Duration) *EventsActor {
	act := &EventsActor{timeout: timeout}
	act.events = make(Events, 0)
	// act.quit = make(chan int)
	act.mux = sync.Mutex{}
	act.tickSend()
	return act
}

func (act *EventsActor) putEvent(ev EventMsg) {
	act.mux.Lock()
	defer act.mux.Unlock()
	act.events = append(act.events, ev)
}

func (act *EventsActor) getEvents() Events {
	act.mux.Lock()
	defer act.mux.Unlock()
	dts := make(Events, len(act.events))
	copy(dts, act.events)
	act.events = make(Events, 0)
	return dts
}

//Receive func Receive in actor
func (act *EventsActor) Receive(ctx actor.Context) {
	act.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
		if err := pubsub.Subscribe("EVENTS/#", ctx.Self(), parseEvents); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)
	case *EventMsg:
		event := msg.ReplaceKeys()
		if v, ok := ((*event)["t"]).(float64); ok {
			(*event)["t"] = (int64)(v * 1000)
		}
		logs.LogBuild.Printf("event message -> %+v", event)
		act.putEvent(*event)
	case *tickMsg:
		events := act.getEvents()
		if len(events) > 0 {
			ctx.Send(ctx.Parent(), &events)
		}
	}
}

type tickMsg struct{}

func (act *EventsActor) tickSend() {
	go func() {
		tick := time.NewTicker(act.timeout)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if act.ctx != nil {
					act.ctx.Send(act.ctx.Self(), &tickMsg{})
				}
			}
		}
	}()
}
