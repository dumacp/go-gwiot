package events

import (
	"github.com/google/uuid"
)

// type DeviceGeneralInformationReported struct {
// 	SDv    string      `json:"sDv,omitempty"`
// 	State  DeviceState `json:"state,omitempty"`
// 	Events []Event     `json:"events,omitempty"`
// }

/*
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
/**/

type DeviceState struct {
	SnDev                       string                 `json:"sn-dev,omitempty"`
	Gstatus                     map[string]interface{} `json:"gstatus,omitempty"`
	SnModem                     string                 `json:"sn-modem,omitempty"`
	SnDisplay                   string                 `json:"sn-display,omitempty"`
	TimeStamp                   float64                `json:"timeStamp,omitempty"`
	IpMask                      map[string][]string    `json:"ipMaskMap,omitempty"`
	Hostname                    string                 `json:"hostname,omitempty"`
	AppVers                     map[string]string      `json:"AppVers,omitempty"`
	SimStatus                   string                 `json:"simStatus,omitempty"`
	SimImei                     int64                  `json:"simImei,omitempty"`
	SimIccid                    string                 `json:"simIccid,omitempty"`
	UsosTranspCount             int64                  `json:"usosTranspCount,omitempty"`
	ErrsTranspCount             int64                  `json:"errsTranspCount,omitempty"`
	Volt                        map[string]float64     `json:"volt,omitempty"`
	Mac                         string                 `json:"mac,omitempty"`
	AppTablesVers               map[string]string      `json:"AppTablesVers,omitempty"`
	CpuStatus                   []float64              `json:"cpuStatus,omitempty"`
	Dns                         []string               `json:"dns,omitempty"`
	UpTime                      string                 `json:"upTime,omitempty"`
	DeviceDataList              []*DiskDetail          `json:"deviceDataList,omitempty"`
	Gateway                     string                 `json:"gateway,omitempty"`
	TypeDev                     string                 `json:"type-dev,omitempty"`
	Ruta                        map[string]string      `json:"ruta,omitempty"`
	TurnstileUpAccum            uint64                 `json:"turnstileUpAccum,omitempty"`
	TurnstileDownAccum          uint64                 `json:"turnstileDownAccum,omitempty"`
	FrontDoorPassengerUpAccum   uint64                 `json:"frontDoorPassengerUpAccum,omitempty"`
	FrontDoorPassengerDownAccum uint64                 `json:"frontDoorPassengerDownAccum,omitempty"`
	BackDoorPassengerUpAccum    uint64                 `json:"backDoorPassengerUpAccum,omitempty"`
	BackDoorPassengerDownAccum  uint64                 `json:"backDoorPassengerDownAccum,omitempty"`
	FrontDoorAnomaliesAccum     uint64                 `json:"pAFC,omitempty"`
	FrontDoorTamperingAccum     uint64                 `json:"pTFC,omitempty"`
	BackDoorAnomaliesAccum      uint64                 `json:"pARC,omitempty"`
	BackDoorTamperingAccum      uint64                 `json:"pTRC,omitempty"`
	FrontDoorTamperingState     int                    `json:"pTFS,omitempty"`
	BackDoorTamperingState      int                    `json:"pTRS,omitempty"`
	TurnstileBattery            int64                  `json:"turnstileBattery,omitempty"`
	TurnstileObstruction        int64                  `json:"turnstileObstruction,omitempty"`
	TurnstileAnomaliesAccum     uint64                 `json:"turnstileAnomaliesAccum,omitempty"`
	RefSystem                   int                    `json:"refSys,omitempty"`
	Version                     string                 `json:"v,omitempty"`
	SendTimestamp               int64                  `json:"sendTimestamp,omitempty"`
}

func DeviceStateToDeviceSTS(ds *DeviceState) *DeviceSTS {

	if ds == nil {
		return nil
	}

	devSts := &DeviceSTS{}

	uuid, err := uuid.NewUUID()
	if err != nil {
		return nil
	}
	devSts.Muuid = uuid.String()

	devSts.Timestamp = int64(ds.TimeStamp * 1000)
	devSts.DeviceSerial = ds.SnDev
	// devSts.GroupName = group_name
	devSts.ReportVersion = ds.Version
	devSts.SystemSTS = &SystemSTS{
		// Temp:          ds.UpTime,
		ReportVersion: ds.Version,
		Cs:            ds.CpuStatus,
		UpTime:        ds.UpTime,
		RefSystem:     ds.RefSystem,
		DisplaySerial: ds.SnDisplay,
		DeviceType:    ds.TypeDev,
		// GroupName: func() string {
		// 	split := strings.Split(group_name, "_")
		// 	if len(split) > 1 {
		// 		return split[0]
		// 	}
		// 	return ""
		// }(),
		// OrganizationID: func() string {
		// 	split := strings.Split(group_name, "_")
		// 	if len(split) > 1 {
		// 		return split[1]
		// 	}
		// 	return ""
		// }(),
		DeviceSerial: ds.SnDev,
		// Ram:          &SystemVolume{},
		// Flash:        &SystemVolume{},
		// SDCard:       &SystemVolume{},
	}

	if ds.Gstatus != nil {
		devSts.SystemSTS.Temp, _ = ds.Gstatus["temperature"].(float64)
	}

	if ds.DeviceDataList != nil {
		for _, disk := range ds.DeviceDataList {
			switch disk.Tp {
			case "MEM":
				devSts.SystemSTS.Ram = &SystemVolume{}
				devSts.SystemSTS.Ram.TotalValue = disk.Tv
				devSts.SystemSTS.Ram.CurrentValue = disk.Cv
				devSts.SystemSTS.Ram.UnitInformation = disk.Ui
				devSts.SystemSTS.Ram.Type = disk.Tp
			case "FLASH":
				devSts.SystemSTS.Flash = &SystemVolume{}
				devSts.SystemSTS.Flash.TotalValue = disk.Tv
				devSts.SystemSTS.Flash.CurrentValue = disk.Cv
				devSts.SystemSTS.Flash.UnitInformation = disk.Ui
				devSts.SystemSTS.Flash.Type = disk.Tp
			case "SD":
				devSts.SystemSTS.SDCard = &SystemVolume{}
				devSts.SystemSTS.SDCard.TotalValue = disk.Tv
				devSts.SystemSTS.SDCard.CurrentValue = disk.Cv
				devSts.SystemSTS.SDCard.UnitInformation = disk.Ui
				devSts.SystemSTS.SDCard.Type = disk.Tp
			}
		}
	}

	devSts.ModemSTS = &ModemSTS{
		ModemSerial: ds.SnModem,
		ModemImei:   ds.SimImei,
		ModemIccid:  ds.SimIccid,
		SimStatus:   ds.SimStatus,
		Gstatus:     ds.Gstatus,
	}

	devSts.NetworkSTS = &NetworkSTS{
		Hostname: ds.Hostname,
		IpMask:   ds.IpMask,
		Dns:      ds.Dns,
		Gateway:  ds.Gateway,
		Mac:      ds.Mac,
	}

	devSts.AppsSTS = &AppsSTS{
		AppVers: ds.AppVers,
	}

	devSts.PaxcounterSTS = &PaxcounterSTS{
		FrontInput:          ds.FrontDoorPassengerUpAccum,
		FrontOutput:         ds.FrontDoorPassengerDownAccum,
		BackInput:           ds.BackDoorPassengerUpAccum,
		BackOutput:          ds.BackDoorPassengerDownAccum,
		FrontAnomalies:      ds.FrontDoorAnomaliesAccum,
		BackAnomalies:       ds.BackDoorAnomaliesAccum,
		FrontTampering:      ds.FrontDoorTamperingAccum,
		BackTampering:       ds.BackDoorTamperingAccum,
		FrontAnomaliesState: ds.FrontDoorTamperingState,
		BackAnomaliesState:  ds.BackDoorTamperingState,
	}

	devSts.TurnstileSTS = &TurnstileSTS{
		TurnstileUpAccum:        ds.TurnstileUpAccum,
		TurnstileDownAccum:      ds.TurnstileDownAccum,
		TurnstileBattery:        ds.TurnstileBattery,
		TurnstileObstruction:    ds.TurnstileObstruction,
		TurnstileAnomaliesAccum: ds.TurnstileAnomaliesAccum,
	}

	devSts.VoltageSTS = &VoltageSTS{
		Voltage: ds.Volt,
	}

	return devSts
}

// type VoltageInfo struct {
// 	Current float64 `json:"current"`
// }

type DeviceSTS struct {
	Muuid        string `json:"muuid,omitempty"`
	Timestamp    int64  `json:"t,omitempty"`
	DeviceSerial string `json:"sDv,omitempty"`
	// GroupName     string         `json:"gN,omitempty"`
	ReportVersion string         `json:"v,omitempty"`
	SystemSTS     *SystemSTS     `json:"system,omitempty"`
	ModemSTS      *ModemSTS      `json:"modem,omitempty"`
	NetworkSTS    *NetworkSTS    `json:"network,omitempty"`
	AppsSTS       *AppsSTS       `json:"apps,omitempty"`
	PaxcounterSTS *PaxcounterSTS `json:"paxcounter,omitempty"`
	TurnstileSTS  *TurnstileSTS  `json:"turnstile,omitempty"`
	VoltageSTS    *VoltageSTS    `json:"voltage,omitempty"`
}

type DiskDetail struct {
	Cv float64 `json:"cV,omitempty"`
	Tv float64 `json:"tV,omitempty"`
	Ui string  `json:"uI,omitempty"`
	Tp string  `json:"tp,omitempty"`
}

type Event struct {
	Timestamp int64       `json:"timestamp,omitempty"`
	Type      string      `json:"type,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	// Dependiendo de la variabilidad de VL, podrías necesitar definir estructuras adicionales.
}

type SystemSTS struct {
	Temp          float64   `json:"temp,omitempty"`
	ReportVersion string    `json:"v,omitempty"`
	Cs            []float64 `json:"cS,omitempty"`
	UpTime        string    `json:"uT,omitempty"`
	RefSystem     int       `json:"refSys,omitempty"`
	DisplaySerial string    `json:"sDs,omitempty"`
	DeviceType    string    `json:"tDv,omitempty"`
	// GroupName      string        `json:"gN,omitempty"`
	// OrganizationID string        `json:"orgI,omitempty"`
	DeviceSerial string        `json:"sDv,omitempty"`
	Ram          *SystemVolume `json:"ram,omitempty"`
	Flash        *SystemVolume `json:"flash,omitempty"`
	SDCard       *SystemVolume `json:"sd,omitempty"`
}

type SystemVolume struct {
	TotalValue      float64 `json:"tV,omitempty"`
	CurrentValue    float64 `json:"cV,omitempty"`
	UnitInformation string  `json:"uI,omitempty"`
	Type            string  `json:"tp,omitempty"`
}

type ModemSTS struct {
	ModemSerial string                 `json:"sM,omitempty"`
	ModemImei   int64                  `json:"sI,omitempty"`
	ModemIccid  string                 `json:"simIccid,omitempty"`
	SimStatus   string                 `json:"sS,omitempty"`
	Gstatus     map[string]interface{} `json:"gs,omitempty"`
}

type NetworkSTS struct {
	Hostname string              `json:"h,omitempty"`
	IpMask   map[string][]string `json:"iMM,omitempty"`
	Dns      []string            `json:"dns,omitempty"`
	Gateway  string              `json:"g,omitempty"`
	Mac      string              `json:"m,omitempty"`
}

type AppsSTS struct {
	AppVers map[string]string `json:"AVer,omitempty"`
}

type PaxcounterSTS struct {
	FrontInput          uint64 `json:"fDPUA,omitempty"`
	FrontOutput         uint64 `json:"fDPDA,omitempty"`
	BackInput           uint64 `json:"bDPUA,omitempty"`
	BackOutput          uint64 `json:"bDPDA,omitempty"`
	FrontAnomalies      uint64 `json:"pAFC,omitempty"`
	BackAnomalies       uint64 `json:"pABC,omitempty"`
	FrontTampering      uint64 `json:"pTFC,omitempty"`
	BackTampering       uint64 `json:"pTBC,omitempty"`
	FrontAnomaliesState int    `json:"pTFS,omitempty"`
	BackAnomaliesState  int    `json:"pTRS,omitempty"`
}

type TurnstileSTS struct {
	TurnstileUpAccum        uint64 `json:"tsUA,omitempty"`
	TurnstileDownAccum      uint64 `json:"tsDA,omitempty"`
	TurnstileBattery        int64  `json:"tsBT,omitempty"`
	TurnstileObstruction    int64  `json:"tsOB,omitempty"`
	TurnstileAnomaliesAccum uint64 `json:"tsAN,omitempty"`
}

type VoltageSTS struct {
	Voltage map[string]float64 `json:"vo,omitempty"`
}
