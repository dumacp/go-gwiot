package events

import (
	"reflect"
)

type TypeVersion interface {
	int | []int
}

type Type interface {
	string | []string
}

type Message[T Type, V TypeVersion] struct {
	H *Header           `json:"h"`
	P *Properties[T, V] `json:"p"`
	B any               `json:"b"`
}

/**
 * MESSAGE HEADER: (Built at transmission time)
 * sDv: device Serial  (Static Id)
 * seq: sequenceNumber (for ack)
 * sts: (OPT) sending timestamp - when this message was sent from the source - the source sets this value
 * rts: (OPT) received timestamp - when this message was received by the destination - the destination sets this value upon arrival
 */
type Header struct {
	DeviceSerial      string `json:"sDv,omitempty"`
	SequenceNumber    int    `json:"seq,omitempty"`
	SendingTimestamp  int64  `json:"sts,omitempty"`
	ReceivedTimestamp int64  `json:"rts,omitempty"`
}

/**
 * APPLICATION PROPERTIES: (Built at creation time)
 * gn: groupName
 * orgi: organization id
 * comi: company id
 * muuid: message Uuid
 * dpid: devicePlatformId (dynamic ID) - if undefined the destination sets this value upon arrival
 * type: Message Type: STATE - GPGGA - COUNTERSDOOR - EVENTS (for MultiEvent types)
 * typv: Message Type Version
 * cts: (OPT) creationTimestamp - how old this message is
 * ttl: (OPT) timeToLive in seconds
 * bf: (OPT) body format - default JSON, others: JSON_ARRAY, PROTOBUF, XML, JSONSTR
 * bc: (OPT) body compression (if apply) - GZIP, BZIP2
 */
type Properties[T Type, V TypeVersion] struct {
	GroupName         string `json:"gn,omitempty"`
	OrganizationID    string `json:"orgi,omitempty"`
	CompanyID         string `json:"comi,omitempty"`
	MessageUUID       string `json:"muuid,omitempty"`
	DevicesPlatformID string `json:"dpid,omitempty"`
	Type              T      `json:"type,omitempty"`
	TypeVersion       V      `json:"typv,omitempty"`
	CreationTimestamp int64  `json:"cts,omitempty"`
	TimeToLive        int    `json:"ttl,omitempty"`
	BodyFormat        string `json:"bf,omitempty"`
	BodyCompression   string `json:"bc,omitempty"`
}

// type MessageInterface interface {
// 	DeviceSerial(deviceSerial string) MessageInterface
// 	SendTimestamp(sendTimestamp int64) MessageInterface
// 	ReceivedTimestamp(receivedTimestamp int64) MessageInterface
// 	MessageUuid(messageUuid string) MessageInterface
// 	GroupName(groupName string) MessageInterface
// 	OrganizationId(organizationId string) MessageInterface
// 	CompanyId(companyId string) MessageInterface
// 	DevicesPlatformId(devicesPlatformId string) MessageInterface
// 	CreationTimestamp(creationTimestamp int64) MessageInterface
// 	Format(format string) MessageInterface
// 	AddType(t string) MessageInterface
// 	AddTypeVersion(typeVersion int) MessageInterface
// 	BodyContent(body interface{}) MessageInterface
// }

func NewMessage() *Message[string, int] {
	return &Message[string, int]{
		H: &Header{},
		P: &Properties[string, int]{},
		B: nil,
	}
}

type MessageGroup Message[[]string, []int]

func NewMessageGroup() *Message[[]string, []int] {
	return &Message[[]string, []int]{
		H: &Header{},
		P: &Properties[[]string, []int]{},
		B: make([]any, 0),
	}
}

func (m *Message[T, V]) DeviceSerial(deviceSerial string) *Message[T, V] {
	if m.H == nil {
		m.H = &Header{}
	}
	m.H.DeviceSerial = deviceSerial
	return m
}

func (m *Message[T, V]) SendTimestamp(sendTimestamp int64) *Message[T, V] {
	if m.H == nil {
		m.H = &Header{}
	}
	m.H.SendingTimestamp = sendTimestamp
	return m
}

func (m *Message[T, V]) ReceivedTimestamp(receivedTimestamp int64) *Message[T, V] {
	if m.H == nil {
		m.H = &Header{}
	}
	m.H.ReceivedTimestamp = receivedTimestamp
	return m
}

func (m *Message[T, V]) MessageUuid(messageUuid string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.MessageUUID = messageUuid
	return m
}

func (m *Message[T, V]) GroupName(groupName string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.GroupName = groupName
	return m
}

func (m *Message[T, V]) OrganizationId(organizationId string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.OrganizationID = organizationId
	return m
}

func (m *Message[T, V]) CompanyId(companyId string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.CompanyID = companyId
	return m
}

func (m *Message[T, V]) DevicesPlatformId(devicesPlatformId string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.DevicesPlatformID = devicesPlatformId
	return m
}

func (m *Message[T, V]) CreationTimestamp(creationTimestamp int64) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.CreationTimestamp = creationTimestamp
	return m
}

func (m *Message[T, V]) Format(format string) *Message[T, V] {
	if m.P == nil {
		m.P = &Properties[T, V]{}
	}
	m.P.BodyFormat = format
	return m
}

func (m *Message[T, V]) AddType(t string) *Message[T, V] {
	switch v := any(m.P.Type).(type) {
	case string:
		m.P.Type = any(t).(T)
	case []string:
		m.P.Type = any(append(v, t)).(T)
	}
	return m
}

func (m *Message[T, V]) Type() T {
	return m.P.Type
}

func (m *Message[T, V]) AddTypeVersion(tv int) *Message[T, V] {
	switch v := any(m.P.TypeVersion).(type) {
	case int:
		m.P.TypeVersion = any(tv).(V)
	case []int:
		m.P.TypeVersion = any(append(v, tv)).(V)
	}
	return m
}

func (m *Message[T, V]) BodyContent(body any) *Message[T, V] {
	if m.B == nil {
		m.B = body
	}
	switch reflect.TypeOf(m.B).Kind() {
	case reflect.Slice:
		m.B = any(append(m.B.([]any), body))
	default:
		m.B = body
	}
	return m
}
