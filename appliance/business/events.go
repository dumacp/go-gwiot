package business

import (
	"encoding/json"

	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
)

//EventMsg event message
type EventMsg map[string]interface{}

//ReplaceKeys replace fields json
func (s *EventMsg) ReplaceKeys() *EventMsg {

	res := make(map[string]interface{})

	keys2Abrev := map[string]string{"upTime": "uT", "sn-dev": "sDv", "sn-modem": "sM", "sn-display": "sDs", "timestamp": "t", "timeStamp": "t", "ipMaskMap": "iMM",
		"hostname": "h", "simStatus": "sS", "simImei": "sI", "usosTranspCount": "uTC", "sn-wifi": "sW", "volt": "vo", "currentValue": "cV",
		"frontDoorPassengerUpAccum": "fDPUA", "frontDoorPassengerDownAccum": "fDPDA", "backDoorPassengerUpAccum": "bDPUA", "backDoorPassengerDownAccum": "bDPDA",
		"turnstileUpAccum":   "tsUA",
		"turnstileDownAccum": "tsDA",
		// "passengersNumber": "pasN",
		"turnstileUpCount":        "tsUC",
		"turnstileDownCount":      "tsDC",
		"turnstileBattery":        "tsBT",
		"turnstileAnomaliesAccum": "tsAN",
		"turnstileObstruction":    "tsOB",
		"highestValue":            "hV", "lowestValue": "lV", "mac": "m", "cpuStatus": "cS", "dns": "dns", "deviceDataList": "dD", "totalValue": "tV",
		"unitInformation": "uI", "gateway": "g", "errsTranspCount": "eTC", "value": "vl", "type": "tp", "gstatus": "gs",
		"temperature": "temp", "AppVers": "AVer", "libparamoperacioncliente": "lPOC", "libcontrolregistros": "lCR",
		"embedded.libgestionhardware": "elGH", "libcontrolconsecutivos": "lCC", "AppUsosTrasnporte": "aUT", "libcommonentities": "lCE",
		"libcontrolmensajeria": "lCM", "libgestionhardware": "lGH", "type-dev": "tDv", "AppTablesVers": "ATVer",
		"TablaTrayectos": "TT", "Trayectos": "TT", "Class": "LN", "ListaNegra": "LN", "Listas restrictivas": "LN",
		"Limites de tiempo": "LT", "TablaLimitesTiempo": "LT", "Mapa rutas": "MR", "MapaRutas": "MR",
		"Configuracion general": "CG", "ConfiguracionGeneral": "CG", "Mensajes de usuario": "MU", "MensajeUsuario": "MU",
		"groupName": "gN", "additionalData": "aD", "version": "v"}

	for k1, v := range *s {
		if v == nil {
			continue
		}
		if k2, ok := keys2Abrev[k1]; ok {
			(res)[k2] = v
		} else {
			(res)[k1] = v
		}
	}

	// status = res

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
