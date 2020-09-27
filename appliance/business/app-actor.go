package business

import (
	"encoding/json"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/appliance/business/messages"

	// svcpubsub "github.com/dumacp/go-gwiot/appliance/business/services/pubsub"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
)

//DeviceMSG device message
type DeviceMSG struct {
	State  *StatusMsg `json:"state"`
	Events *Events    `json:"events"`
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

		// if err := services(ctx); err != nil {
		// 	time.Sleep(3 * time.Second)
		// 	logs.LogError.Panic(err)
		// }

		logs.LogInfo.Println("started actor")

	case *messages.KeycloakAddressRequest:
		ctx.Send(ctx.Sender(), app.pidKeycloak)

	case *StatusMsg:
		if v, ok := (*msg)["sDv"].(string); ok {
			app.snDev = v
		}
		remoteMsg := &DeviceMSG{State: msg}
		data, err := json.Marshal(remoteMsg)
		if err != nil {
			logs.LogWarn.Printf("status messages error -> %q", err)
		}
		ctx.Send(app.pidRemote, &messages.RemoteMSG{Data: data})
	case *Events:
		if len(app.snDev) <= 0 {
			app.snDev = Hostname()
		}
		state := make(map[string]interface{})
		(state)["sDv"] = app.snDev

		remoteMsg := &DeviceMSG{State: (*StatusMsg)(&state), Events: msg}
		data, err := json.Marshal(remoteMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		ctx.Send(app.pidRemote, &messages.RemoteMSG{Data: data})
	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	}
}
