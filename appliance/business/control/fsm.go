package control

import (
	"errors"
	"fmt"
	"time"

	"github.com/looplab/fsm"
)

const (
	timeoutReset = 15 * time.Minute
)

const (
	startEvent       = "startEvent"
	resetEvent       = "resetEvent"
	powerOffEvent    = "powerOffEvent"
	powerOnEvent     = "powerOnEvent"
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
	sReconnect  = "sReconnect"
	sReset      = "sReset"
	sResetHard  = "sResetHard"
	sIfDownUp   = "sIfDownUp"
	sTest       = "sTest"
	sPowerOff   = "sPowerOff"
	sPowerOn    = "sPowerOn"
	sWaitModem1 = "sWaitModem1"
	sWaitModem2 = "sWaitModem2"
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
			{Name: modemOKEvent, Src: []string{sWaitModem2}, Dst: sReconnect},
			{Name: modemFailEvent, Src: []string{sWaitModem1}, Dst: sResetHard},
			{Name: modemFailEvent, Src: []string{sWaitModem2}, Dst: sResetHard},
			// {Name: testFailMaxEvent, Src: []string{sWaitModem2}, Dst: sPowerOff},
			{Name: connOKEvent, Src: []string{sReconnect}, Dst: sIfDownUp},
			{Name: connFailEvent, Src: []string{sReconnect}, Dst: sReset},
			{Name: testOKEvent, Src: []string{sTestConn1}, Dst: sWait},
			{Name: testOKEvent, Src: []string{sIfDownUp}, Dst: sWait},
			{Name: testFailEvent, Src: []string{sTestConn1}, Dst: sReconnect},
			{Name: testFailEvent, Src: []string{sIfDownUp}, Dst: sReset},
			{Name: testFailEvent, Src: []string{sStart}, Dst: sPowerOff},
			{Name: testFailMaxEvent, Src: []string{sResetHard}, Dst: sPowerOff},
			// {Name: testFailMaxEvent, Src: []string{sIfDownUp}, Dst: sPowerOff},
			{Name: resetEvent, Src: []string{sReset}, Dst: sResetHard},
			{Name: resetEvent, Src: []string{sResetHard}, Dst: sWaitModem2},
			{Name: powerOffEvent, Src: []string{sPowerOff}, Dst: sPowerOn},
			{Name: powerOnEvent, Src: []string{sPowerOn}, Dst: sWaitModem1},
			{Name: timeoutEvent, Src: []string{sWaitModem1}, Dst: sWaitModem1},
			{Name: timeoutEvent, Src: []string{sWaitModem2}, Dst: sWaitModem2},
			{Name: timeoutEvent, Src: []string{sWait}, Dst: sTestConn1},
			{Name: timeoutEvent, Src: []string{sReset}, Dst: sWaitModem1},

			{Name: resetCmdEvent, Src: []string{
				sWait,
				sWaitModem1,
				sTestConn1,
				sIfDownUp,
				sReconnect,
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
			enterState(sPowerOff): func(e *fsm.Event) {
				act.countReset = 0
			},
		},
	)
}

type incCountError struct {
	value int
}

func newIncCountErr() *incCountError {
	return &incCountError{value: 10}
}
func (c *incCountError) get() int {
	return c.value
}
func (c *incCountError) add() {
	if c.value <= 30 {
		c.value += 3
	}
}
func (c *incCountError) restart() {
	c.value = 10
}

func (act *CheckModemActor) startfsm() {

	// log.Println(m.Current())
	errInc := newIncCountErr()
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
				if !verifySIM(act.mSierra) {
					warnlog.Println("SIM is not OK!")
					act.fsm.Event(testFailEvent)
				} else {
					infolog.Println("SIM is OK!")
					act.fsm.Event(startEvent)
				}
			case sWaitModem1:
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if verifyModem(act.mSierra) != 0 {
					if act.countError >= errInc.get() {
						errInc.add()
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
			case sReconnect:
				errInc.restart()
				act.behavior.Become(act.stateRun)
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if !reConnect(act.mSierra, act.apn) {
					if act.countError > 5 {
						if !verifySIM(act.mSierra) {
							warnlog.Println("SIM is not OK!")
						}
						act.fsm.Event(connFailEvent)
					} else {
						act.countError++
						time.Sleep(3 * time.Second)
					}
					break
				}
				act.fsm.Event(connOKEvent)
			case sIfDownUp:
				if act.resetCmd {
					act.fsm.Event(resetCmdEvent)
					break
				}
				if err := ifDown(); err != nil {
					if act.debug {
						infolog.Println(err)
					}
				}
				time.Sleep(1 * time.Second)
				if err := ifUp(); err != nil {
					warnlog.Println(err)
					act.fsm.Event(testFailEvent)
					break
				}
				if err := pingFunc(act.testIP); err != nil {
					if act.countError > 3 {
						warnlog.Printf("ping error: %s\n", err)
						act.fsm.Event(testFailEvent)
					} else {
						act.countError++
					}
					break
				}
				infolog.Println("modem NET connected!")
				act.fsm.Event(testOKEvent)
			case sReset:
				if act.lastReset.Add(timeoutReset).Unix() > time.Now().Unix() {
					act.fsm.Event(timeoutEvent)
					break
				}
				act.fsm.Event(resetEvent)
			case sResetHard:
				act.behavior.Become(act.stateReset)
				if act.countReset > maxError {
					act.fsm.Event(testFailMaxEvent)
					break
				}
				act.countReset++
				warnlog.Println("reset modem")
				if !resetSWModem(act.mSierra) {
					if !resetModem(act.mSierra) {
						errlog.Println("Error reset Modem")
					}
				}
				act.lastReset = time.Now()
				time.Sleep(3 * time.Second)
				act.fsm.Event(resetEvent)
			case sWaitModem2:
				switch verifyModem(act.mSierra) {
				case 0:
					act.fsm.Event(modemOKEvent)
				default:
					if act.countError >= errInc.get() {
						errInc.add()
						act.fsm.Event(modemFailEvent)
					} else {
						act.countError++
						time.Sleep(3 * time.Second)
						act.fsm.Event(timeoutEvent)
					}
				}
				act.fsm.Event(modemOKEvent)
			case sPowerOff:
				if verifyModem(act.mSierra) == 1 {
					errlog.Println("serial port connecting Error")
					resetUSBHost(act.mSierra)
				}
				if !powerOffModem(act.mSierra) {
					errlog.Println("Error power Off Modem")
				}
				act.lastReset = time.Now()
				time.Sleep(3 * time.Second)
				act.fsm.Event(powerOffEvent)
			case sPowerOn:
				if !powerOnModem(act.mSierra) {
					errlog.Println("Error power On Modem")
					panic("Error power On Modem")
				}
				time.Sleep(10 * time.Second)
				act.fsm.Event(powerOnEvent)
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
