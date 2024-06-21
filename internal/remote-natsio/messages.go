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

type GetOrgID struct{}
type GetGroupName struct{}
type GetOrgIDResponse struct {
	OrgID string
}
type GetGroupNameResponse struct {
	GroupName string
}
