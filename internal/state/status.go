package state

import (
	"bytes"
	"encoding/json"
	"log"
	"math"

	"golang.org/x/exp/constraints"

	"github.com/dumacp/go-logs/pkg/logs"
)

//StatusMsg status message
type StatusMsg map[string]interface{}

var keys2Abrev = map[string]string{"upTime": "uT", "sn-dev": "sDv", "sn-modem": "sM", "sn-display": "sDs", "timestamp": "t", "timeStamp": "t", "ipMaskMap": "iMM",
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

//ReplaceKeys rename field keys
func (s *StatusMsg) ReplaceKeys() *StatusMsg {
	res := ReplaceKeys(*s)
	return (*StatusMsg)(&res)
}

func ReplaceKeys(data map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})

	for k1, v := range data {
		if v == nil {
			continue
		}

		switch datai := v.(type) {
		case map[string]interface{}:

			if vi := ReplaceKeys(datai); len(vi) > 0 {
				// log.Printf("replace -> %v, %v", k1, vi)
				if k2, ok := keys2Abrev[k1]; ok {
					(res)[k2] = vi
				} else {
					(res)[k1] = vi
				}
			} else {
				if k2, ok := keys2Abrev[k1]; ok {
					(res)[k2] = v
				} else {
					(res)[k1] = v
				}
			}
		case []interface{}:
			for i, vi := range datai {
				if vii, ok := vi.(map[string]interface{}); ok {
					if viii := ReplaceKeys(vii); len(viii) > 0 {
						datai[i] = viii
					}
				}
			}
			if k2, ok := keys2Abrev[k1]; ok {
				(res)[k2] = datai
			} else {
				(res)[k1] = datai
			}

		default:
			if k2, ok := keys2Abrev[k1]; ok {
				(res)[k2] = v
			} else {
				(res)[k1] = v
			}
		}
	}

	// status = res

	return res
}

var keysStatic = map[string]int{"uTC": 0, "eTC": 0, "t": 0, "sDv": 0, "tsOB": 0, "tsBT": 0,
	"tsUA": 0, "tsDA": 0, "tsAN": 0, "bDPUA": 0, "bDPDA": 0, "fDPUA": 0, "fDPDA": 0, "AVer": 0,
	"pAFC": 0, "pARC": 0, "pTRC": 0, "pABC": 0, "pTBC": 0, "pTFC": 0, "pTFS": 0, "pTRS": 0}
var keysRemove = map[string]int{"sM": 0, "sDs": 0, "sW": 0, "sn-cn": 0}

//OnlyChanges expose only new values in fields
func (s *StatusMsg) OnlyChanges(last *StatusMsg) *StatusMsg {

	if last == nil {
		return s
	}

	result := funcCompare_v2("alldata", (map[string]interface{})(*s), (map[string]interface{})(*last))

	if v, ok := result.(map[string]interface{}); ok {
		return (*StatusMsg)(&v)
	} else {
		return nil
	}

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
			log.Printf("last %q -> %T, %v", k, vlast, vlast)
			switch ri := vlast.(type) {

			case map[string]interface{}:
				if result := funcCompare(v.(map[string]interface{}), ri); len(result) > 0 {
					log.Printf("last -> %v, %v", k, result)
					res[k] = result
				}
			case []string:
				if vdi, ok := v.([]string); ok {
					equal := true
					for ii, vi := range ri {
						log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], vi == vdi[ii])
						if vi != vdi[ii] {
							equal = false
							break
						}
					}
					if !equal {
						res[k] = v
					}
				}
			case [][]byte:
				if vdi, ok := v.([][]byte); ok {
					equal := true
					for ii, vi := range ri {
						log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], bytes.EqualFold(vi, vdi[ii]))
						if !bytes.EqualFold(vi, vdi[ii]) {
							equal = false
							break
						}
					}
					if !equal {
						res[k] = v
					}
				}
			case []int:
				if vdi, ok := v.([]int); ok {
					equal := true
					for ii, vi := range ri {
						log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], vi == vdi[ii])
						if vi != vdi[ii] {
							equal = false
							break
						}
					}
					if !equal {
						res[k] = v
					}
				}
			case []int64:
				if vdi, ok := v.([]int64); ok {
					equal := true
					for ii, vi := range ri {
						log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], vi == vdi[ii])
						if vi != vdi[ii] {
							equal = false
							break
						}
					}
					if !equal {
						res[k] = v
					}
				}
			case []float64:
				if vdi, ok := v.([]float64); ok {
					equal := true
					for ii, vi := range ri {
						log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], vi == vdi[ii])
						if math.Abs(vi-vdi[ii]) >= 0.001 {
							equal = false
							break
						}
					}
					if !equal {
						res[k] = v
					}
				}
			case []interface{}:

				res[k] = v

			default:

				res[k] = v
			}
		} else {
			res[k] = v
		}
	}
	return res
}

func compareSlice[S ~[]T, T constraints.Ordered](t1, t2 S) bool {

	for i, v := range t2 {
		// log.Printf("last el -> %v, new el -> %v, equal -> %v", vi, vdi[ii], bytes.EqualFold(vi, vdi[ii]))
		// if !strings.EqualFold(v, new[i]) {
		if t1[i] != v {
			return false
		}
	}
	return true
}

func funcCompare_v2(key string, data, lastData interface{}) interface{} {

	if _, ok := keysRemove[key]; ok {
		return nil
	}
	if _, ok := keysStatic[key]; ok {
		return data
	}

	switch last := lastData.(type) {

	case map[string]interface{}:

		// log.Printf("///////  last %q -> %T, %v", key, data, data)
		new, ok := data.(map[string]interface{})
		if !ok {
			break
		}
		result := make(map[string]interface{})
		for k, v := range last {
			if res := funcCompare_v2(k, new[k], v); res != nil {
				result[k] = res
			}
		}
		if len(result) <= 0 {
			return nil
		}
		data = result
	case []string:
		// log.Printf("///////  last %q -> %T, %v", key, data, data)
		new, ok := data.([]string)
		if !ok {
			return data
		}
		if compareSlice(new, last) {
			return nil
		}
		return new
	case [][]byte:
		// log.Printf("///////  last %q -> %T, %v", key, data, data)
		new, ok := data.([]string)
		if !ok {
			return data
		}
		last_ := make([]string, 0)
		for _, v := range last {
			last_ = append(last_, string(v))
		}
		if compareSlice(new, last_) {
			return nil
		}
		return new
	case []int:
		new, ok := data.([]int)
		if !ok {
			return data
		}
		if compareSlice(new, last) {
			return nil
		}
		return new
	case []int64:
		new, ok := data.([]int64)
		if !ok {
			return data
		}
		if compareSlice(new, last) {
			return nil
		}
		return new
	case []float64:
		new, ok := data.([]float64)
		if !ok {
			return data
		}
		if compareSlice(new, last) {
			return nil
		}
		return new
	case int, int64, float64, string:
		// log.Printf("last %q -> %T, %v", key, data, data)
		if last != data {
			return data
		}
		return nil
	case []byte:
		new, ok := data.([]byte)
		if !ok {
			return data
		}
		// log.Printf("last %q -> %T, %v", key, data, data)
		if bytes.Contains(new, last) {
			return data
		}
		return nil
	case []interface{}:
		// log.Printf("/////// /////    ////   last__ %q -> %T, %v", key, data, data)
		if len(last) <= 0 {
			return data
		}
		new_, ok := data.([]interface{})
		if !ok {
			return data
		}
		new := make([]string, 0)
		for _, v := range new_ {
			if vi, oki := v.(string); !oki {
				return data
			} else {
				new = append(new, vi)
			}
		}
		last_ := make([]string, 0)
		for _, v := range last {
			if vi, oki := v.(string); !oki {
				return data
			} else {
				last_ = append(last_, vi)
			}
		}
		// log.Printf("///////  last__ %q -> %T, %v", key, last_, last_)
		if compareSlice(new, last_) {
			return nil
		}
		// log.Printf("///////  last__ %q -> %T, %v", key, last_, last_)
	default:
		log.Printf("/////// /////    ////   last %q -> %T, %T, %v", key, data, last, data)
	}
	return data
}
