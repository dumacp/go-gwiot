package events

import (
	"encoding/json"

	"github.com/dumacp/go-gwiot/internal/state"
	"github.com/dumacp/go-logs/pkg/logs"
)

//EventMsg event message
type EventMsg map[string]interface{}

//ReplaceKeys rename field keys
func (s *EventMsg) ReplaceKeys() *EventMsg {

	res := state.ReplaceKeys(*s)

	return (*EventMsg)(&res)

}

//Events events queue
type Events []EventMsg

func parseEvents(msg []byte) interface{} {

	event := new(EventMsg)
	if err := json.Unmarshal(msg, event); err != nil {
		logs.LogWarn.Printf("parse error in events -> %s", err)
		return nil
	}

	return event
}
