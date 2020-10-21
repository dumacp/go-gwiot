package controlext

import (
	"errors"
	"fmt"
	"time"

	"github.com/looplab/fsm"
)

const (
	timeoutReset = 15 * time.Minute
	ipLocalTest  = "192.168.188.24"
)

const (
	startEvent       = "startEvent"
	resetEvent       = "resetEvent"
	waitEvent        = "waitEvent"
	timeoutEvent     = "timeoutEvent"
	testFailEvent    = "testFailEvent"
	testOKEvent      = "testOKEvent"
	testFailMaxEvent = "testFailMaxEvent"
	connFailEvent    = "connFailEvent"
	connOKEvent      = "connOKEvent"
	modemFailEvent   = "modemFailEvent"
	modemOKEvent     = "modemOKEvent"
	resetCmdEvent    = "resetCmdEvent"
)

const (
	sStart      = "sStart"
	sTestConn1  = "sTestConn1"
	sWait       = "sWait"
	sReset      = "sReset"
	sResetHard  = "sResetHard"
	sWaitModem1 = "sWaitModem1"
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

func (act *CheckModemActor) initFSM() {
	act.fsm = fsm.NewFSM(
		sStart,
		fsm.Events{
			{Name: startEvent, Src: []string{sStart}, Dst: sWaitModem1},
			{Name: modemOKEvent, Src: []string{sWaitModem1}, Dst: sTestConn1},
			{Name: modemFailEvent, Src: []string{sWaitModem1}, Dst: sResetHard},
			{Name: testOKEvent, Src: []string{sTestConn1}, Dst: sWait},
			{Name: testFailEvent, Src: []string{sTestConn1}, Dst: sReset},
			{Name: resetEvent, Src: []string{sReset}, Dst: sResetHard},
			{Name: resetEvent, Src: []string{sResetHard}, Dst: sWaitModem1},
			{Name: timeoutEvent, Src: []string{sWaitModem1}, Dst: sWaitModem1},
			{Name: timeoutEvent, Src: []string{sWait}, Dst: sTestConn1},
			{Name: timeoutEvent, Src: []string{sReset}, Dst: sWaitModem1},

			{Name: resetCmdEvent, Src: []string{
				sWait,
				sWaitModem1,
				sTestConn1,
			},
				Dst: sReset},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) {
				if act.debug {
					infolog.Printf("FSM MODEM state Src: %v, state Dst: %v", e.Src, e.Dst)
				}
			},
			"leave_state": func(e *fsm.Event) {
				act.countError = 0
				// infolog.Printf("countError = %v; resetCmd = %v", act.countError, act.resetCmd)
				if e.Err != nil {
					e.Cancel(e.Err)
				}
			},
			"before_event": func(e *fsm.Event) {
				if e.Err != nil {
					e.Cancel(e.Err)
				}
			},
			enterState(sWait): func(e *fsm.Event) {
				act.countWait = 0
			},
			enterState(sReset): func(e *fsm.Event) {
				act.resetCmd = false
			},
		},
	)
}

func (act *CheckModemActor) startfsm() {

	// log.Println(m.Current())
	funcRutine := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				errlog.Println("Recovered in \"startfsm()\",", r)
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
		for {
			// infolog.Printf("current state: %v", act.fsm.Current())
			switch act.fsm.Current() {
			case sStart:
				act.behavior.Become(act.stateRun)
				act.modem = NewModemExt()
				act.fsm.Event(startEvent)
			case sWaitModem1:
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if err := pingFunc(ipLocalTest); err != nil {
					if act.countError >= 40 {
						warnlog.Println(err)
						act.fsm.Event(modemFailEvent)
					} else {
						act.countError++
						time.Sleep(3 * time.Second)
						act.fsm.Event(timeoutEvent)
					}
					break
				}
				act.fsm.Event(modemOKEvent)
			case sTestConn1:
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if err := pingFunc(act.testIP); err != nil {
					warnlog.Println(err)
					act.fsm.Event(testFailEvent)
					break
				}
				act.fsm.Event(testOKEvent)
			case sWait:
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if act.countWait > 10 {
					act.fsm.Event(timeoutEvent)
					break
				}
				time.Sleep(3 * time.Second)
				act.countWait++
			case sReset:
				if act.lastReset.Add(timeoutReset).Unix() > time.Now().Unix() {
					act.fsm.Event(timeoutEvent)
					break
				}
				act.fsm.Event(resetEvent)
			case sResetHard:
				act.behavior.Become(act.stateReset)
				warnlog.Println("reset modem")
				if err := act.modem.resetmodem(act.debug); err != nil {
					errlog.Printf("Error reset Modem: %v", err)
				}
				act.lastReset = time.Now()
				time.Sleep(3 * time.Second)
				act.fsm.Event(resetEvent)
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
