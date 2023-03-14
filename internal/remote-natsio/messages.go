package renatsio

import "github.com/nats-io/nats.go"

type Connect struct{}
type Reconnect struct{}
type Connected struct {
	Conn *nats.Conn
}
