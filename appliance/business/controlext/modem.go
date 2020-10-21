package controlext

import (
	"time"

	"github.com/brian-armstrong/gpio"
)

const (
	addrcontrol = 164
)

type modem struct {
	pinReset gpio.Pin
}

func NewModemExt() *modem {
	m := &modem{}
	m.pinReset = gpio.NewOutput(addrcontrol, true)
	return m
}

func (m *modem) resetmodem(debug bool) error {

	defer func() {
		if debug {
			infolog.Printf("modem: HIGH")
		}
		if err := m.pinReset.High(); err != nil {
			errlog.Printf("error reset modem: %s\n", err)
		}
	}()
	if debug {
		infolog.Printf("modem: LOW")
	}
	if err := m.pinReset.Low(); err != nil {
		errlog.Printf("error reset modem: %s\n", err)
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}
