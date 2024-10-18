package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/events"
	"github.com/dumacp/go-gwiot/internal/keyc"
	"github.com/dumacp/go-gwiot/internal/parameters"
	"github.com/dumacp/go-gwiot/internal/state"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/messages"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/google/uuid"
)

const (
	messagesVersion = 3
)

// DeviceMSG device message
type DeviceMSG struct {
	State  *state.StatusMsg `json:"state"`
	Events *events.Events   `json:"events"`
}

// DeviceStateMSG device message
type DeviceStateMSG struct {
	State *state.StatusMsg `json:"state"`
}

func (b *DeviceStateMSG) MarshalJSON() ([]byte, error) {

	state := make(state.StatusMsg)

	for k, v := range *b.State {
		if v != nil {
			state[k] = v
		}
	}

	statejson, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`{"state": %s}`, statejson)), nil
}

// DeviceEventsMSG device message
type DeviceEventsMSG struct {
	Events *events.Events `json:"events"`
}

// App main app
type App struct {
	pidKeycloak *actor.PID
	pidRemote   *actor.PID
	propsRemote *actor.Props
	params      *parameters.PlatformParameters
	snDev       string
}

// NewApp new actor
func NewApp(propsRemote *actor.Props) *App {
	app := &App{propsRemote: propsRemote}
	return app
}

type ErrorRemote struct{}

// Receive function Receive
func (app *App) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Println("starting actor")

		propsParams := actor.PropsFromProducer(func() actor.Actor {
			return parameters.NewActor(utils.Hostname())
		})

		if _, err := ctx.SpawnNamed(propsParams, "params-actor"); err != nil {
			logs.LogError.Printf("error starting actor: %s", err)
		}

		propsStatus := actor.PropsFromProducer(func() actor.Actor {
			return state.NewStatus()
		})
		_, err := ctx.SpawnNamed(propsStatus, "status-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		propsEvents := actor.PropsFromProducer(func() actor.Actor {
			return events.NewEvents(5 * time.Second)
		})
		_, err = ctx.SpawnNamed(propsEvents, "events-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		if len(keyc.Keycloakurl) > 0 {
			propsKeycloak := actor.PropsFromProducer(func() actor.Actor {
				return keyc.NewKeycActor(utils.Hostname(), utils.Hostname())
			})
			app.pidKeycloak, err = ctx.SpawnNamed(propsKeycloak, "keyc-actor")
			if err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panic(err)
			}
		}

		app.pidRemote, err = ctx.SpawnNamed(app.propsRemote, "remote-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		ctx.Watch(app.pidRemote)

		logs.LogInfo.Println("started actor")

	case *messages.KeycloakAddressRequest:
		if app.pidKeycloak == nil {
			break
		}
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
	case *parameters.PlatformParameters:
		app.params = msg
	case *parameters.GetPlatformParameters:
		if ctx.Sender() == nil {
			break
		}
		if app.params != nil {
			ctx.Respond(app.params)
		}
	case *messages.GroupIDRequest:
		if app.pidKeycloak == nil {
			break
		}
		req := ctx.RequestFuture(app.pidKeycloak, &messages.GroupIDRequest{}, 6*time.Second)
		if err := req.Wait(); err != nil {
			break
		}
		v, err := req.Result()
		if err != nil {
			break
		}
		ctx.Send(ctx.Sender(), v)

	case *state.StatusMsg:
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
		eventsMsg := &DeviceEventsMSG{Events: &events.Events{}}
		dataEvents, err := json.Marshal(eventsMsg)
		if err != nil {
			logs.LogWarn.Printf("events messages error -> %q", err)
		}
		ctx.Send(app.pidRemote, &messages.RemoteMSG2{State: dataState, Events: dataEvents, Serial: app.snDev, Retry: 0, TimeStamp: time.Now().Unix(), Version: 2, Data: nil})
	case *events.Events:
		if app.params != nil && app.params.Props != nil &&
			len(app.params.Props.BROKER_URL_EXTERNAL) > 0 &&
			len(app.params.Props.DEV_PID) > 0 {
			for _, event := range *msg {
				for k, v := range event {
					if strings.EqualFold(k, "tp") {
						if strings.EqualFold(v.(string), "GPRMC") ||
							strings.EqualFold(v.(string), "GPGGA") {
							data, err := json.Marshal(event)
							if err != nil {
								logs.LogWarn.Printf("external events messages error -> %q", err)
							}
							ctx.Send(app.pidRemote, &messages.ExternalEvent{
								Data: data,
							})
						}
					}
				}
			}
		}
		if len(app.snDev) <= 0 {
			app.snDev = utils.Hostname()
		}
		state_ := make(map[string]interface{})
		(state_)["sDv"] = app.snDev
		muuid, err := uuid.NewUUID()
		if err != nil {
			logs.LogWarn.Printf("events muuid error -> %s", err)
		}
		(state_)["muuid"] = muuid
		(state_)["v"] = messagesVersion

		stateMsg := &DeviceStateMSG{State: (*(state.StatusMsg))(&state_)}
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
