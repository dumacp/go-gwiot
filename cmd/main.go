package main

import (
	"fmt"
	"net"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-logs/pkg/logs"
)

func main() {

	sys := actor.NewActorSystem()

	portlocal := 8009
	for {
		portlocal++

		socket := fmt.Sprintf("127.0.0.1:%d", portlocal)
		testConn, err := net.DialTimeout("tcp", socket, 1*time.Second)
		if err != nil {
			break
		}
		testConn.Close()
		logs.LogWarn.Printf("socket busy -> \"%s\"", socket)
		time.Sleep(1 * time.Second)
	}

	rconfig := remote.Configure("127.0.0.1", portlocal)
	r := remote.NewRemote(sys, rconfig)
	r.Start()

}
