package remote

type MsgSendData struct {
	Data    []byte
	Subject string
}
type MsgExternalSendData struct {
	Data    []byte
	Subject string
	Topic   string
}
type MsgAck struct {
}
type MsgError struct {
	Error error
}
type MsgReconnect struct {
}
