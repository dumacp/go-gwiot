/*
Package implements a binary for read serial portNmea nmea.

*/
package nmeatcp

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/dumacp/gpsnmea"
)

// var timeout int
// var baudRate int
// var portNmea string
// var distanceMin int

const (
	// addrcontrol    = 84
	// resetModemPath = "/sys/class/leds/reset-pciusb/brightness"
	timeoutMax = 30 * time.Second
)

// func init() {
// 	flag.IntVar(&timeout, "timeout", 30, "timeout to capture frames.")
// 	flag.IntVar(&baudRate, "baudRate", 115200, "baud rate to capture nmea's frames.")
// 	flag.StringVar(&portNmea, "portNmea", "/dev/ttyUSB1", "device serial to read.")
// 	flag.IntVar(&distanceMin, "distance", 30, "minimun distance traveled before to send")
// }

//var pub *pubsub.PubSub

func (act *actornmea) run(conn net.Conn, timeout, distanceMin int) error {

	re := regexp.MustCompile(`\$[a-zA-Z]+,`)

	mapCaptures := make(map[string]string)

	long0 := float64(0)
	lat0 := float64(0)
	long0_t := float64(0)
	lat0_t := float64(0)

	chDist := make(chan int, 0)
	t1 := time.NewTimer(time.Duration(timeout) * time.Second)
	defer t1.Stop()

	chQuit := make(chan int, 0)
	defer close(chQuit)

	go func() {
		defer close(chDist)
		for {
			//only publish if frame GPRMC is not quiet
			select {
			case <-chDist:
				for _, v := range mapCaptures {
					if act.debug {
						infolog.Printf("distance, EVENT -> %s\n", v)
					}
					act.context.Send(act.pubsubPID, &eventGPS{event: v})
				}
				mapCaptures = make(map[string]string)
				if !t1.Stop() {
					select {
					case <-t1.C:
					default:
					}
				}
				t1.Reset(time.Duration(timeout) * time.Second)
			case <-t1.C:
				for _, v := range mapCaptures {
					if act.debug {
						infolog.Printf("timeout, EVENT -> %s\n", v)
					}
					lat0 = lat0_t
					long0 = long0_t
					act.context.Send(act.pubsubPID, &eventGPS{event: v})
				}
				mapCaptures = make(map[string]string)
				t1.Reset(time.Duration(timeout) * time.Second)
			case <-chQuit:
				warnlog.Println("chQuit nil")
				return
			}
		}
	}()

	chReset := make(chan int, 0)
	chResetStop := make(chan int, 0)

	timerModem := struct {
		timer  *time.Timer
		active bool
	}{}

	timerModem.timer = time.NewTimer(0)
	if !timerModem.timer.Stop() {
		select {
		case <-timerModem.timer.C:
		default:
		}
	}
	timerModem.active = false

	go func() {
		defer close(chReset)

		for {
			select {
			case <-timerModem.timer.C:
				timerModem.active = false
				chResetStop <- 0
			case v := <-chReset:
				if v <= 0 {
					// warnlog.Println("modem reset timer off, with GPS data")
					if !timerModem.timer.Stop() {
						select {
						case <-timerModem.timer.C:
						default:
						}
					}
					timerModem.active = false
					break
				}
				if timerModem.active {
					break
				}
				warnlog.Println("modem reset timer on, without GPS data")
				if !timerModem.timer.Stop() {
					select {
					case <-timerModem.timer.C:
					default:
					}
				}
				timerModem.timer.Reset(time.Duration(v) * time.Second)
				timerModem.active = true
			case <-chQuit:
				warnlog.Println("chQuit nil")
				return
			}
		}
	}()
	modemVerify := func(value int) {
		select {
		case chReset <- value:
		default:
		}
	}

	funcListen := func(debug bool) error {
		// defer act.dev.Close()

		defer func() {
			if r := recover(); r != nil {
				errlog.Println("Recovered in funcListen, ", r)
			}
		}()

		ch := gpsnmea.ReadTCP(conn, filter...)
		tmax := time.NewTicker(timeoutMax)
		defer tmax.Stop()

		tn := time.Now()
		for {
			select {
			case <-chResetStop:
				return fmt.Errorf("reset modem NMEA, timer stop")
			case <-tmax.C:
				// if len(mapCaptures) <= 0 {
				// 	break break_for
				// }
				if time.Now().Unix() > tn.Add(timeoutMax).Unix() {
					return fmt.Errorf("reset modem NMEA, timeout read frame")
				}
			case frame, ok := <-ch:
				if !ok {
					return fmt.Errorf("device channel error")
				}
				tn = time.Now()
				timeStamp := float64(tn.UnixNano()) / 1000000000
				if debug {
					infolog.Printf("frame: %s\n", frame)
				}
				if len(frame) > 34 {
					if timerModem.active {
						modemVerify(0)
					}
					gtype := re.FindString(frame)
					if strings.Count(frame, "$") > 1 {
						warnlog.Printf("frame bad format %s", frame)
						continue
					}
					if len(gtype) > 3 {
						mapCaptures[gtype] = fmt.Sprintf("{\"timeStamp\": %f, \"value\": %q, \"type\": %q}", timeStamp, frame, gtype[1:len(gtype)-1])
						act.context.Send(act.pubsubPID, &msgGPS{data: frame})

						if strings.Contains(gtype, "GPRMC") {
							gprmc := gpsnmea.ParseRMC(frame)
							long1 := gpsnmea.LatLongToDecimalDegree(gprmc.Long, gprmc.LongCord)
							lat1 := gpsnmea.LatLongToDecimalDegree(gprmc.Lat, gprmc.LatCord)
							distance := gpsnmea.Distance(lat0, long0, lat1, long1, "K")
							if distance > float64(distanceMin)/1000 {
								if debug {
									infolog.Printf("distance: %v K\n", distance)
								}
								long0 = long1
								lat0 = lat1
								chDist <- 1
							}
							long0_t = long1
							lat0_t = lat1
						}
					}
				} else {
					if !timerModem.active {
						modemVerify(240)
					}
				}
			}
		}
	}
	err := funcListen(act.debug)
	if err != nil {
		// errlog.Println(err)
		return err
	}
	return fmt.Errorf("funcListen terminated")
}

// func (act *actornmea) resetModem() {
// 	warnlog.Printf()
// 	act.context.Send(act.modemPID, &messages.ModemReset{})
// 	//TODO:
// }
