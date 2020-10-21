package control

import (
	"github.com/dumacp/omvz/sierra"
)

const (
	portModem = "/dev/ttyUSB2"
	baudModem = 115200
)

//NewModem function to connect with sierra modem
func newModem() sierra.SierraModem {
	opts := &sierra.PortOptions{
		Port: portModem,
		Baud: baudModem,
	}
	modSierra := sierra.NewModem(opts)
	return modSierra
}

func verifyModem(m sierra.SierraModem) int {
	if !m.Open() {
		return 1
	}
	defer m.Close()
	if !m.Verify() {
		return 2
	}
	return 0
}

func verifySIM(m sierra.SierraModem) bool {
	if !m.Open() {
		return false
	}
	defer m.Close()
	return m.IsSimOK()
}

func powerOffModem(m sierra.SierraModem) bool {
	return m.PowerOff()
}

func powerOnModem(m sierra.SierraModem) bool {
	return m.PowerOn()
}

func resetModem(m sierra.SierraModem) bool {
	return m.ResetHw()
}

func resetSWModem(m sierra.SierraModem) bool {
	return m.ResetSw()
}

func resetUSBHost(m sierra.SierraModem) bool {
	return m.ResetUSBHost()
}

func reConnect(m sierra.SierraModem, apn string) bool {
	if !m.Open() {
		return false
	}
	defer m.Close()

	succ, _ := m.ConnectToApn(apn)
	return succ
}
