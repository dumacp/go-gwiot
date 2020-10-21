package control

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-modem/appliance/business/messages"
	"github.com/dumacp/omvz/sierra"
	"github.com/looplab/fsm"
)

type msgFatal struct {
	err error
}

type CheckModemActor struct {
	debug      bool
	behavior   actor.Behavior
	mSierra    sierra.SierraModem
	context    actor.Context
	fsm        *fsm.FSM
	testIP     string
	apn        string
	countError int
	countReset int
	countWait  int
	resetCmd   bool
	lastReset  time.Time
}

const (
	maxError      = 5
	maxSuccess    = 3
	timeoutWait   = 300 * time.Second
	ipTestInitial = "8.8.8.8"
)

func NewCheckModemActor(debug bool) actor.Actor {
	initLogs(debug)
	act := &CheckModemActor{
		behavior: actor.NewBehavior(),
	}
	act.debug = debug
	act.mSierra = newModem()
	act.initFSM()
	act.behavior.Become(act.stateInitial)

	return act
}

func (state *CheckModemActor) Receive(context actor.Context) {
	state.context = context
	state.behavior.Receive(context)
}

func (state *CheckModemActor) stateInitial(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		infolog.Printf("Starting, \"netmodem\", pid: %v\n", context.Self())
		state.testIP = ipTestInitial

		state.startfsm()

	case *actor.Stopping:
		fmt.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		fmt.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		fmt.Println("Restarting, actor is about to restart")
	case *messages.ModemCheck:
		fmt.Printf("ModemCheck: %s\n", msg)
		if len(msg.Addr) > 0 {
			if _, err := net.ResolveIPAddr("ip4:icmp", msg.Addr); err != nil {
				log.Println(err)
			} else {
				state.testIP = msg.Addr
			}
		}
		state.apn = msg.Apn
	case *msgFatal:
		panic(msg.err)
	}
}

func (state *CheckModemActor) stateRun(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Stopping:
		fmt.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		fmt.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		fmt.Println("Restarting, actor is about to restart")
	case *messages.ModemCheck:
		fmt.Printf("ModemCheck: %s\n", msg)
		if len(msg.Addr) > 0 {
			if _, err := net.ResolveIPAddr("ip4:icmp", msg.Addr); err != nil {
				log.Println(err)
			} else {
				state.testIP = msg.Addr
			}
		}
		state.apn = msg.Apn
	case *messages.ModemOnRequest:
		fmt.Printf("%s\n", msg)
		context.Respond(&messages.ModemOnResponse{State: true})
	case *messages.ModemReset:
		fmt.Printf("%s\n", msg)
		warnlog.Printf("external modem reset from \"%s\"\n", context.Sender().String())
		state.resetCmd = true
	case *msgFatal:
		panic(msg.err)
	}
}

func (state *CheckModemActor) stateReset(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Stopping:
		fmt.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		fmt.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		fmt.Println("Restarting, actor is about to restart")
	case *messages.ModemCheck:
		fmt.Printf("ModemCheck: %s\n", msg)
		if len(msg.Addr) > 0 {
			if _, err := net.ResolveIPAddr("ip4:icmp", msg.Addr); err != nil {
				log.Println(err)
			} else {
				state.testIP = msg.Addr
			}
		}
		state.apn = msg.Apn
	case *messages.ModemOnRequest:
		fmt.Printf("%s\n", msg)
		context.Respond(&messages.ModemOnResponse{State: false})
	case *msgFatal:
		panic(msg.err)
	}
}
