package renatsio

import "github.com/nats-io/nats.go"

type Connection struct{}
type Reconnect struct{}
type Disconnected struct {
	Error error
}
type ConnectionResponse struct {
	Conn *nats.Conn
}
