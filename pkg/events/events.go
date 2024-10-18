package events

import (
	"time"

	"github.com/google/uuid"
)

func MessageFromEvent(event *Event) *Message[string, int] {

	sendTime := time.Now().UnixMilli()

	uid, _ := uuid.NewUUID()

	messageGroup := NewMessage().SendTimestamp(sendTime).
		MessageUuid(uid.String()).DevicesPlatformId("").
		CreationTimestamp(sendTime).Format("JSON")

	switch event.Type {
	case "GPRMC":
		body := map[string]interface{}{
			"t":     event.Timestamp,
			"gprmc": event.Value,
		}

		messageGroup.AddType("LOC_EVT").AddTypeVersion(1).BodyContent(body)
	case "GPGGA":
		body := map[string]interface{}{
			"t":     event.Timestamp,
			"gpgga": event.Value,
		}

		messageGroup.AddType("LOC_EVT").AddTypeVersion(1).BodyContent(body)
		// Añadir casos para otros tipos de eventos según sea necesario
	case "GPSERROR":
		gpsErrorTimeStamp := func() int64 {
			if event.Timestamp*1000 > time.Now().UnixMilli() {
				return event.Timestamp
			}
			return event.Timestamp * 1000
		}()

		messageGroup.AddType("LOC_STS").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":  gpsErrorTimeStamp,
			"ea": event.Value,
		})

	case "lowest_volt":
		messageGroup.AddType("VOLT_STS").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"l": event.Value,
		})
	case "highest_volt":
		messageGroup.AddType("VOLT_STS").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"h": event.Value,
		})
	case "alert_volt":
		messageGroup.AddType("VOLT_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"v": event.Value,
			"a": func() string {
				if event.Value.(float64) <= 12 {
					return "low"
				}
				return "high"
			}(),
		})
	case "DOOR":
		/* SAMPLE
		   {
		       "t": 1619557659926,
		       "tp": "DOOR",
		       "vl": {
		           "coord": "$GPRMC,210739.0,A,0614.697883,N,07534.316962,W,19.1,105.2,270421,4.7,W,A*31",
		           "id": 1,
		           "state": 1
		       }
		   }
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		state, _ := vl["state"].(float64)

		messageGroup.AddType("DOOR_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":     event.Timestamp,
			"c":     coord,
			"id":    int(id),
			"state": state,
		})
	case "TURNSTILE":
		/* SAMPLE
		{
			"t": 1619557695755,
			"tp": "TURNSTILE",
			"vl": {
				"coord": "$GPRMC,210815.0,A,0616.739734,N,07536.352549,W,0.0,114.2,270421,4.7,W,A*0A",
				"tsDC": 0,
				"tsUC": 1
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		tsDC, _ := vl["tsDC"].(float64)
		tsUC, _ := vl["tsUC"].(float64)

		messageGroup.AddType("TURNSTILE_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":    event.Timestamp,
			"c":    coord,
			"tsDC": int(tsDC),
			"tsUC": int(tsUC),
		})
	case "PANIC_BUTTON_EVT":
		/*
			{
				"t":1661437296162,
				"tp":"PANIC_BUTTON_EVT"
			 }
		*/
		messageGroup.AddType("PANIC_BUTTON_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":  event.Timestamp,
			"tp": event.Type,
		})
	case "TURNSTILE_BATTERY":
		/* SAMPLE
		{
			"t": 1619556001642,
			"tp": "TURNSTILE_BATTERY",
			"vl": {
				"batteryOn": true,
				"coord": "$GPRMC,204001.0,A,0616.680041,N,07538.325067,W,0.0,280.5,270421,4.7,W,A*0A"
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		batteryOn, _ := vl["batteryOn"].(bool)
		coord, _ := vl["coord"].(string)

		messageGroup.AddType("TURNSTILE_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":  event.Timestamp,
			"oB": batteryOn,
			"c":  coord,
		})
	case "TURNSTILE_ANOMALIES":
		/* SAMPLE
		{
			"t": 1619556274580,
			"tp": "TURNSTILE_ANOMALIES",
			"vl": {
				"coord": "$GPRMC,204434.0,A,0616.383278,N,07535.587169,W,0.0,30.4,270421,4.7,W,A*32",
				"turnstileAnomalyCount": 1
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord := vl["coord"].(string)
		turnstileAnomalyCount := vl["turnstileAnomalyCount"].(float64)

		messageGroup.AddType("TURNSTILE_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"c": coord,
			"a": int(turnstileAnomalyCount),
		})
	case "TURNSTILE_OBSTRUCTION":
		/* SAMPLE
		{
			"t": 1619556364751,
			"tp": "TURNSTILE_OBSTRUCTION",
			"vl": {
				"coord": "$GPRMC,204604.0,A,0615.863729,N,07535.784640,W,0.0,15.7,270421,4.7,W,A*3D",
				"obstruction": true
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		obstruction, _ := vl["obstruction"].(bool)

		messageGroup.AddType("TURNSTILE_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t":  event.Timestamp,
			"c":  coord,
			"od": obstruction,
		})
	case "alert_status_volt":
		/* SAMPLE
		{
			"t": 1671544613642,
			"tp": "alert_status_volt",
			"vl": {
				"active": true,
				"vl": 0.30400002
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		active, _ := vl["active"].(bool)
		v, _ := vl["vl"].(float64)

		messageGroup.AddType("ALERT_VOLT_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"a": active,
			"v": v,
		})
	case "COUNTERSDOOR":
		/* SAMPLE
		{
			"t": 1619556262596,
			"tp": "COUNTERSDOOR",
			"vl": {
				"coord": "$GPRMC,204422.0,A,0614.312946,N,07532.912510,W,0.0,76.6,270421,4.7,W,A*34",
				"counters": [
				1,
				0
				],
				"id": 0,
				"state": 1
			}
		}
		*/

		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		counters, _ := vl["counters"].([]interface{})
		state, _ := vl["state"].(float64)
		v := make(map[string]interface{})

		v["t"] = event.Timestamp
		v["c"] = coord

		if id == 0 {
			if len(counters) > 0 {
				v["fI"] = int(counters[0].(float64))
			}
			if len(counters) > 1 {
				v["fO"] = int(counters[1].(float64))
			}
			v["fC"] = state
		} else {
			if len(counters) > 0 {
				v["bI"] = int(counters[0].(float64))
			}
			if len(counters) > 1 {
				v["bO"] = int(counters[1].(float64))
			}
			v["bC"] = state
		}

		messageGroup.AddType("PAXCOUNTER_EVT").AddTypeVersion(1).BodyContent(v)
	case "TAMPERING":
		/* SAMPLE
		{
			"t": 1619556394565,
			"tp": "TAMPERING",
			"vl": {
				"coord": "$GPRMC,204634.0,A,0613.652629,N,07531.777874,W,14.9,135.2,270421,4.8,W,A*31",
				"counters": [
				0,
				0
				],
				"id": 1,
				"state": 1
			}
		}
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		state, _ := vl["state"].(float64)

		v := make(map[string]interface{})
		v["t"] = event.Timestamp
		v["c"] = coord
		if id == 0 {
			v["fT"] = 1
			v["fC"] = state
		} else {
			v["bT"] = 1
			v["bC"] = state
		}

		messageGroup.AddType("PAXCOUNTER_EVT").AddTypeVersion(1).BodyContent(v)
	case "CounterDisconnect":
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		tp, _ := vl["type"].(string)

		v := make(map[string]interface{})

		v["t"] = event.Timestamp
		v["c"] = coord
		v["ty"] = tp

		if id == 0 {
			v["fCx"] = 0
		} else {
			v["bCx"] = 0
		}

		messageGroup.AddType("PAXCOUNTER_EVT").AddTypeVersion(1).BodyContent(v)
	case "CounterConnected":
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		tp, _ := vl["type"].(string)

		v := make(map[string]interface{})

		v["t"] = event.Timestamp
		v["c"] = coord
		v["ty"] = tp

		if id == 0 {
			v["fCx"] = 1
		} else {
			v["bCx"] = 1
		}

		messageGroup.AddType("PAXCOUNTER_EVT").AddTypeVersion(1).BodyContent(v)

	case "PAX_ANOMALIES":
		/* SAMPLE
		   {
		       "t": 1619556394565,
		       "tp": "PAX_ANOMALIES",
		       "vl": {
		           "coord": "$GPRMC,204634.0,A,0613.652629,N,07531.777874,W,14.9,135.2,270421,4.8,W,A*31",
		           "counters": [0,0],
		           "id": 1,
		           "state": 1
		       }
		   }
		*/
		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		id, _ := vl["id"].(float64)
		state, _ := vl["state"].(float64)

		v := make(map[string]interface{})
		v["t"] = event.Timestamp
		v["c"] = coord
		if id == 0 {
			v["fA"] = 1
			v["fC"] = state
		} else {
			v["bA"] = 1
			v["bC"] = state
		}

		messageGroup.AddType("PAXCOUNTER_EVT").AddTypeVersion(1).BodyContent(v)

	case "IgnitionEvent":
		/* SAMPLE
		   {
		       "t": 1619556279305,
		       "tp": "IgnitionEvent",
		       "vl": {
		           "coord": "$GPRMC,204431.0,A,0617.003175,N,07537.521110,W,0.0,322.8,270421,4.7,W,A*0F",
		           "state": 1
		       }
		   }
		*/

		vl, _ := event.Value.(map[string]interface{})
		coord, _ := vl["coord"].(string)
		state, _ := vl["state"].(float64)

		messageGroup.AddType("IGNITION_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"c": coord,
			"o": state,
		})

	case "tagValidation":
		messageGroup.AddType("TAG_VALIDATION_EVT").AddTypeVersion(1)

	case "READER_EVT":
		/* SAMPLE
		{
			"t": 1619556274580,
			"tp": "READER_EVT",
			"vl": {
				"error": "error description",
				"state": false
			}
		}
		*/

		vl, _ := event.Value.(map[string]interface{})
		errorDesc, _ := vl["error"].(string)
		state, _ := vl["state"].(bool)

		messageGroup.AddType("READER_EVT").AddTypeVersion(1).BodyContent(map[string]interface{}{
			"t": event.Timestamp,
			"e": errorDesc,
			"s": state,
		})

	}

	return messageGroup
}
