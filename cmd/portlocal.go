package main

import (
	"fmt"
	"net"
	"time"

	"github.com/dumacp/go-logs/pkg/logs"
)

func portlocal() int {
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
	return portlocal
}
