package nmeatcp

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/dumacp/gpsnmea"
	"github.com/dumacp/go-modem/appliance/business/messages"
	"github.com/looplab/fsm"
)

var filter = []string{"$GPRMC", "$GPGGA"}

const (
	sStart   = "sStart"
	sConnect = "sConnect"
	sRun     = "sRun"
	sReset   = "sReset"
)

const (
	startEvent       = "startEvent"
	readFailEvent    = "readFailEvent"
	connectOKEvent   = "connectOKEvent"
	connectFailEvent = "connectFailEvent"
	timeoutEvent     = "tomeoutEvent"
	resetEvent       = "resetEvent"
)

func beforeEvent(event string) string {
	return fmt.Sprintf("before_%s", event)
}
func enterState(state string) string {
	return fmt.Sprintf("enter_%s", state)
}
func leaveState(state string) string {
	return fmt.Sprintf("leave_%s", state)
}

func (act *actornmea) initFSM() {
	act.fsm = fsm.NewFSM(
		sStart,
		fsm.Events{
			{Name: startEvent, Src: []string{sStart}, Dst: sConnect},
			{Name: connectOKEvent, Src: []string{sConnect}, Dst: sRun},
			{Name: connectFailEvent, Src: []string{sConnect}, Dst: sReset},
			{Name: timeoutEvent, Src: []string{sConnect}, Dst: sConnect},
			{Name: readFailEvent, Src: []string{sRun}, Dst: sReset},
			{Name: resetEvent, Src: []string{sReset}, Dst: sConnect},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) {
				if act.debug {
					infolog.Printf("FSM NMEA state Src: %v, state Dst: %v", e.Src, e.Dst)
				}
			},
			"leave_state": func(e *fsm.Event) {
				if e.Err != nil {
					e.Cancel(e.Err)
				}
			},
			"before_event": func(e *fsm.Event) {
				if e.Err != nil {
					e.Cancel(e.Err)
				}
			},
		},
	)
}

func (act *actornmea) startfsm() {
	// log.Println(m.Current())
	funcRutine := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				errlog.Println("Recovered in \"startfsm()\", ", r)
				switch x := r.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = errors.New("Unknown panic")
				}
			}
		}()

		var conn net.Conn
		for {
			// log.Println(m.Current())
			switch act.fsm.Current() {
			case sStart:
				dev, err := gpsnmea.NewDeviceTCP(act.portNmea)
				if err != nil {
					warnlog.Println(err)
					act.fsm.Event(connectFailEvent)
					break
				}
				act.dev = dev
				act.fsm.Event(startEvent)
			case sConnect:
				conn.Close()
				conn, err = act.dev.Accept()
				if err != nil {
					warnlog.Println(err)
					time.Sleep(5 * time.Second)
					act.fsm.Event(connectFailEvent)
					break
				}
				act.fsm.Event(connectOKEvent)
			case sReset:
				act.context.Request(act.modemPID, &messages.ModemReset{})
				time.Sleep(30 * time.Second)
				act.fsm.Event(resetEvent)
			case sRun:
				if err := act.run(conn, act.timeout, act.distanceMin); err != nil {
					errlog.Println(err)
				}
				act.fsm.Event(readFailEvent)
			default:
				time.Sleep(3 * time.Second)
			}
		}
	}
	go func() {
		if err := funcRutine(); err != nil {
			act.context.Send(act.context.Self(), &msgFatal{err: err})
		}
	}()
}
