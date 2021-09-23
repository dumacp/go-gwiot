package services

import (
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/dumacp/go-gwiot/appliance/services"
)

type service struct {
	state    messages.StatusResponse_StateType
	evstream *eventstream.EventStream
}

var instance *service
var once sync.Once

//GetInstane get instance of service
func GetInstance() services.Service {
	if instance == nil {
		once.Do(func() {
			instance = &service{}
			instance.evstream = eventstream.NewEventStream()
		})
	}
	return instance
}

func (svc *service) Subscribe(fun func(evt interface{})) {
	svc.evstream.Subscribe(fun)
}

func (svc *service) Start() {
	svc.state = messages.STARTED
	svc.evstream.Publish(&messages.Start{})
}

func (svc *service) Stop() {
	svc.state = messages.STOPPED
	svc.evstream.Publish(&messages.Stop{})
}

func (svc *service) Restart() {
	svc.state = messages.STOPPED
	svc.evstream.Publish(&messages.Stop{})
	time.Sleep(1 * time.Second)
	svc.evstream.Publish(&messages.Start{})
	svc.state = messages.STARTED
}

func (svc *service) Status() *messages.StatusResponse {
	return &messages.StatusResponse{
		State: svc.state,
	}
}

// func (svc *service) Info(ctx actor.Context, pid *actor.PID) (*messages.IgnitionStateResponse, error) {
// 	future := ctx.RequestFuture(pid, &messages.IgnitionStateRequest{}, time.Second*3)
// 	err := future.Wait()
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := future.Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	msg, ok := res.(*messages.IgnitionStateResponse)
// 	if !ok {
// 		return nil, fmt.Errorf("message error: %T", msg)
// 	}
// 	return msg, nil
// }

// func (svc *service) EventsSubscription(ctx actor.Context, pid *actor.PID) (*messages.IgnitionEventsSubscriptionAck, error) {
// 	future := ctx.RequestFuture(pid, &messages.IgnitionEventsSubscription{}, time.Second*3)
// 	err := future.Wait()
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := future.Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	msg, ok := res.(*messages.IgnitionEventsSubscriptionAck)
// 	if !ok {
// 		return nil, fmt.Errorf("message error: %T", msg)
// 	}
// 	return msg, nil
// }
