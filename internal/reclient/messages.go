package reclient

type MsgSendData struct {
	Data    []byte
	Subject string
}
type MsgAck struct {
}
type MsgError struct {
	Error error
}
type MsgReconnect struct {
}
