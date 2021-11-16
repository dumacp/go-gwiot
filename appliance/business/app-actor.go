package business

import (
	"encoding/json"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/google/uuid"

	// svcpubsub "github.com/dumacp/go-gwiot/appliance/business/services/pubsub"
	"github.com/dumacp/go-logs/pkg/logs"
)

const (
	messagesVersion = 3
)

//DeviceMSG device message
type DeviceMSG struct {
	State  *StatusMsg `json:"state"`
	Events *Events    `json:"events"`
}

//DeviceStateMSG device message
type DeviceStateMSG struct {
	State *StatusMsg `json:"state"`
}

//DeviceEventsMSG device message
type DeviceEventsMSG struct {
	Events *Events `json:"events"`
}

//App main app
type App struct {
	pidKeycloak *actor.PID
	pidRemote   *actor.PID
	propsRemote *actor.Props
	snDev       string
}

//NewApp new actor
func NewApp(propsRemote *actor.Props) *App {
	app := &App{propsRemote: propsRemote}
	return app
}

// func services(ctx actor.Context) error {
// 	var err error
// 	propsGrpc := actor.PropsFromProducer(func() actor.Actor {
// 		return grpc.NewService()
// 	})
// 	_, err = ctx.SpawnNamed(propsGrpc, "svc-grpc")
// 	if err != nil {
// 		return err
// 	}
// 	propsPubSub := actor.PropsFromProducer(func() actor.Actor {
// 		return svcpubsub.NewService()
// 	})
// 	_, err = ctx.SpawnNamed(propsPubSub, "svc-mqtt")
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

type ErrorRemote struct{}

//Receive function Receive
func (app *App) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Println("starting actor")

		propsStatus := actor.PropsFromProducer(func() actor.Actor {
			return NewStatus()
		})
		_, err := ctx.SpawnNamed(propsStatus, "status-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		propsEvents := actor.PropsFromProducer(func() actor.Actor {
			return NewEvents(5 * time.Second)
		})
		_, err = ctx.SpawnNamed(propsEvents, "events-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		propsKeycloak := actor.PropsFromProducer(func() actor.Actor {
			return NewKeycActor(Hostname(), Hostname())
		})
		app.pidKeycloak, err = ctx.SpawnNamed(propsKeycloak, "keyc-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		// propsRemote := actor.PropsFromProducer(func() actor.Actor {
		// 	return NewStatus()
		// })
		app.pidRemote, err = ctx.SpawnNamed(app.propsRemote, "remote-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		ctx.Watch(app.pidRemote)

		// if err := services(ctx); err != nil {
		// 	time.Sleep(3 * time.Second)
		// 	logs.LogError.Panic(err)
		// }

		logs.LogInfo.Println("started actor")

	case *messages.KeycloakAddressRequest:
		ctx.Send(ctx.Sender(), app.pidKeycloak)

	case *actor.Terminated:
		logs.LogInfo.Printf("actor %q terminated (supervised)", msg.Who.GetId())
		if msg.Who.GetId() == app.pidRemote.GetId() {
			var err error
			app.pidRemote, err = ctx.SpawnNamed(app.propsRemote, "remote-actor")
			if err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panic(err)
			}
		}
	case *messages.GroupIDRequest:
		req := ctx.RequestFuture(app.pidKeycloak, &messages.GroupIDRequest{}, 3*time.Second)
		if err := req.Wait(); err != nil {
			break
		}
		v, err := req.Result()
		if err != nil {
			break
		}
		// log.Printf("groupID -> %+v", v)
		ctx.Send(ctx.Sender(), v)

	case *StatusMsg:
		if v, ok := (*msg)["sDv"].(string); ok {
			app.snDev = v
		}
		muuid, err := uuid.NewUUID()
		if err != nil {
			logs.LogWarn.Printf("status muuid error -> %s", err)
		}
		(*msg)["muuid"] = muuid
		(*msg)["v"] = messagesVersion

		stateMsg := &DeviceStateMSG{State: msg}
		dataState, err := json.Marshal(stateMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		eventsMsg := &DeviceEventsMSG{Events: &Events{}}
		dataEvents, err := json.Marshal(eventsMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		ctx.Send(app.pidRemote, &messages.RemoteMSG2{State: dataState, Events: dataEvents, Serial: app.snDev, Retry: 0, TimeStamp: time.Now().Unix(), Version: 2, Data: nil})
	case *Events:
		if len(app.snDev) <= 0 {
			app.snDev = Hostname()
		}
		state := make(map[string]interface{})
		(state)["sDv"] = app.snDev
		muuid, err := uuid.NewUUID()
		if err != nil {
			logs.LogWarn.Printf("events muuid error -> %s", err)
		}
		(state)["muuid"] = muuid
		(state)["v"] = messagesVersion

		stateMsg := &DeviceStateMSG{State: (*StatusMsg)(&state)}
		dataState, err := json.Marshal(stateMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		eventsMsg := &DeviceEventsMSG{Events: msg}
		dataEvents, err := json.Marshal(eventsMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		ctx.Send(app.pidRemote, &messages.RemoteMSG2{State: dataState, Events: dataEvents, Serial: app.snDev, Retry: 0, TimeStamp: time.Now().Unix(), Version: 2, Data: nil})
	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	case *ErrorRemote:
		ctx.PoisonFuture(ctx.Sender()).Wait()
		ctx.SpawnNamed(app.propsRemote, "remote-actor")
	}
}
