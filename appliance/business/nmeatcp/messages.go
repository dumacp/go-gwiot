package nmeatcp

type msgFatal struct {
	err error
}

type msgGPS struct {
	data string
}

type eventGPS struct {
	event string
}

type AddressModem struct {
	Addr string
	ID   string
}

type AddressPubSub struct {
	ID string
}
