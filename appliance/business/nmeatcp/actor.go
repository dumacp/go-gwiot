package nmeatcp

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/gpsnmea"
	"github.com/looplab/fsm"
)

type actornmea struct {
	debug       bool
	context     actor.Context
	fsm         *fsm.FSM
	dev         *gpsnmea.DeviceTCP
	portNmea    string
	timeout     int
	distanceMin int
	modemPID    *actor.PID
	pubsubPID   *actor.PID
}

func NewNmeaActor(debug bool, portNmea string, baudRate, timeout, distanceMin int) actor.Actor {
	initLogs(debug)
	act := &actornmea{}
	act.debug = debug
	act.portNmea = portNmea
	act.timeout = timeout
	act.distanceMin = distanceMin
	act.initFSM()
	return act
}

func (act *actornmea) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		infolog.Printf("actor started \"%s\"", ctx.Self().Id)

		propsPubSub := actor.PropsFromFunc(NewPubSubActor(act.debug).Receive)
		pidPubSub, err := ctx.SpawnNamed(propsPubSub, "nmeaPubSub")
		if err != nil {
			errlog.Panic(err)
		}
		act.pubsubPID = pidPubSub
		ctx.Watch(pidPubSub)
		act.startfsm()
	case *msgFatal:
		errlog.Println(msg.err)
		panic(msg.err)
	case *AddressModem:
		act.modemPID = actor.NewPID(msg.Addr, msg.ID)
	case *AddressPubSub:
		act.pubsubPID = actor.NewLocalPID(msg.ID)
	case *actor.Terminated:
		errlog.Printf("actor terminated: %s", msg.Who.GetId())
	}
}
