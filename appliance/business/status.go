package business

import (
	"encoding/json"

	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
)

//StatusMsg status message
type StatusMsg map[string]interface{}

//ReplaceKeys rename field keys
func (s *StatusMsg) ReplaceKeys() *StatusMsg {

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

	return (*StatusMsg)(&res)

}

var keysStatic = map[string]int{"uTC": 0, "eTC": 0, "t": 0, "sDv": 0, "tsOB": 0, "tsBT": 0,
	"tsUA": 0, "tsDA": 0, "tsAN": 0, "bDPUA": 0, "bDPDA": 0, "fDPUA": 0, "fDPDA": 0}
var keysRemove = map[string]int{"sM": 0, "sDs": 0, "sW": 0, "sn-cn": 0}

//OnlyChanges expose only new values in fields
func (s *StatusMsg) OnlyChanges(last *StatusMsg) *StatusMsg {

	if last == nil {
		return s
	}

	result := funcCompare(*s, *last)

	return (*StatusMsg)(&result)
}

func parseStatus(msg []byte) interface{} {

	status := new(StatusMsg)
	if err := json.Unmarshal(msg, status); err != nil {
		logs.LogWarn.Printf("error parsing status message -> %s", err)
		return nil
	}

	return status
}

func funcCompare(data, lastData map[string]interface{}) map[string]interface{} {

	res := make(map[string]interface{})

	for k, v := range data {
		// log.Printf("data %q -> %T, %v", k, v, v)

		if _, ok := keysRemove[k]; ok {
			continue
		}
		if _, ok := keysStatic[k]; ok {
			res[k] = v
			continue
		}
		if vlast, ok := (lastData)[k]; ok {
			// log.Printf("last %q -> %T, %v", k, vlast, vlast)
			switch ri := vlast.(type) {

			case map[string]interface{}:
				if result := funcCompare(v.(map[string]interface{}), ri); len(result) > 0 {
					res[k] = result
				}
			case []interface{}:
				res[k] = v
			default:
				if v != vlast {
					res[k] = v
				}
			}
		} else {
			res[k] = v
		}
	}
	return res
}
