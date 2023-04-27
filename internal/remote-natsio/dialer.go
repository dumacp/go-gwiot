package renatsio

import (
	"log"
	"net"
	"time"
)

type CustomDialer struct {
	*net.Dialer
	skipTLS bool
	delay   time.Duration
}

func (d *CustomDialer) Dial(network, address string) (net.Conn, error) {
	conn, err := d.Dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &customConn{conn, d.delay}, nil
}

func (sd *CustomDialer) SkipTLSHandshake() bool {
	return sd.skipTLS
}

type customConn struct {
	net.Conn
	delay time.Duration
}

func (c *customConn) Read(b []byte) (int, error) {

	n, err := c.Conn.Read(b)
	if err != nil {
		return n, err
	}

	log.Println("/////////////// data\n")
	// time.Sleep(c.delay)

	return n, err
}
