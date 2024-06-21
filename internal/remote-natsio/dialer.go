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

var countbytes int64
var t0 = time.Now()
var t1 = time.Now()

func (c *customConn) Read(b []byte) (int, error) {

	n, err := c.Conn.Read(b)
	if err != nil {
		return n, err
	}

	countbytes += int64(n)

	log.Printf("/////////////// data, rate %.02f kB in %s\n", float64(countbytes)/1000, time.Since(t0))
	log.Printf("/////////////// data, rate %.02f kB/s\n", float64(n)/(1000*time.Since(t1).Seconds()))
	// time.Sleep(c.delay)
	t1 = time.Now()

	return n, err
}
