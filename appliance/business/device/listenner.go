package device

import (
	"log"
	"time"

	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	evdev "github.com/gvalkov/golang-evdev"
)

//NewEventDevice connect with device serial
func NewEventDevice(path string) (*evdev.InputDevice, error) {
	dev, err := evdev.Open(path)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

//EventUP up
type EventUP struct{}

//EventDown down
type EventDown struct{}

//Listen function to listen events
func Listen(quit chan int, dev *evdev.InputDevice) <-chan interface{} {

	ch := make(chan interface{})

	go func() {
		defer close(ch)
		// var eventMem *evdev.InputEvent
		for {
			log.Println("leyendo!")
			iv, err := dev.ReadOne()
			if err != nil {
				continue
			}
			logs.LogBuild.Printf("event: %v", iv)
			switch iv.Code {
			case evdev.KEY_WAKEUP:
				if iv.Value == 0 {
					select {
					case ch <- &EventUP{}:
					case <-time.After(10 * time.Second):
					}
				} else {
					select {
					case ch <- &EventDown{}:
					case <-time.After(10 * time.Second):
					}
				}
			}
		}
	}()

	return ch
}
