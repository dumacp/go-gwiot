package renatsio

import "github.com/nats-io/nats.go"

type Connection struct{}
type Reconnect struct{}
type ConnectionResponse struct {
	Conn *nats.Conn
}
