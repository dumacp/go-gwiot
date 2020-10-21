package messages

import "encoding/json"

type Status struct {
	Gstatus   interface{}         `json:"gstatus"`
	SnDev     string              `json:"sn-dev"`
	SnModem   string              `json:"sn-modem"`
	SnDisplay string              `json:"sn-display"`
	TimeStamp float64             `json:"timeStamp"`
	IpMask    map[string][]string `json:"ipMaskMap"`
	//IpMask		map[string][]net.Addr	`json:"ipMaskMap"`
	Hostname                    string                   `json:"hostname"`
	AppVers                     map[string]string        `json:"AppVers"`
	SimStatus                   string                   `json:"simStatus"`
	SimImei                     int64                    `json:"simImei"`
	UsosTranspCount             int64                    `json:"usosTranspCount"`
	ErrsTranspCount             int64                    `json:"errsTranspCount"`
	Volt                        map[string]float64       `json:"volt"`
	Mac                         string                   `json:"mac"`
	AppTablesVers               map[string]string        `json:"AppTablesVers"`
	CpuStatus                   []float64                `json:"cpuStatus"`
	Dns                         []string                 `json:"dns"`
	UpTime                      string                   `json:"upTime"`
	DeviceDataList              []map[string]interface{} `json:"deviceDataList"`
	Gateway                     string                   `json:"gateway"`
	TypeDev                     string                   `json:"type-dev"`
	Ruta                        map[string]string        `json:"ruta"`
	TurnstileUpAccum            uint64                   `json:"turnstileUpAccum"`
	TurnstileDownAccum          uint64                   `json:"turnstileDownAccum"`
	FrontDoorPassengerUpAccum   uint64                   `json:"frontDoorPassengerUpAccum"`
	FrontDoorPassengerDownAccum uint64                   `json:"frontDoorPassengerDownAccum"`
	BackDoorPassengerUpAccum    uint64                   `json:"backDoorPassengerUpAccum"`
	BackDoorPassengerDownAccum  uint64                   `json:"backDoorPassengerDownAccum"`
	TurnstileBattery            int64                    `json:"turnstileBattery"`
	TurnstileObstruction        int64                    `json:"turnstileObstruction"`
	TurnstileAnomaliesAccum     uint64                   `json:"turnstileAnomaliesAccum"`
	RefSystem                   int                      `json:"refSys"`
}

//ReplaceKeys rename json fields
func (s *Status) ReplaceKeys() ([]byte, error) {
	res, err := json.Marshal(struct {
		// *Status
		// OmitGstatus                     omit `json:"gstatus,omitempty"`
		// OmitSnDev                       omit `json:"sn-dev,omitempty"`
		// OmitSnModem                     omit `json:"sn-modem,omitempty"`
		// OmitSnDisplay                   omit `json:"sn-display,omitempty"`
		// OmitTimeStamp                   omit `json:"timeStamp,omitempty"`
		// OmitIpMask                      omit `json:"ipMaskMap,omitempty"`
		// OmitHostname                    omit `json:"hostname,omitempty"`
		// OmitAppVers                     omit `json:"AppVers,omitempty"`
		// OmitSimStatus                   omit `json:"simStatus,omitempty"`
		// OmitSimImei                     omit `json:"simImei,omitempty"`
		// OmitUsosTranspCount             omit `json:"usosTranspCount,omitempty"`
		// OmitErrsTranspCount             omit `json:"errsTranspCount,omitempty"`
		// OmitVolt                        omit `json:"volt,omitempty"`
		// OmitMac                         omit `json:"mac,omitempty"`
		// OmitAppTablesVers               omit `json:"AppTablesVers,omitempty"`
		// OmitCpuStatus                   omit `json:"cpuStatus,omitempty"`
		// OmitDns                         omit `json:"dns,omitempty"`
		// OmitUpTime                      omit `json:"upTime,omitempty"`
		// OmitDeviceDataList              omit `json:"deviceDataList,omitempty"`
		// OmitGateway                     omit `json:"gateway,omitempty"`
		// OmitTypeDev                     omit `json:"type-dev,omitempty"`
		// OmitRuta                        omit `json:"ruta,omitempty"`
		// OmitTurnstileUpAccum            omit `json:"turnstileUpAccum,omitempty"`
		// OmitTurnstileDownAccum          omit `json:"turnstileDownAccum,omitempty"`
		// OmitFrontDoorPassengerUpAccum   omit `json:"frontDoorPassengerUpAccum,omitempty"`
		// OmitFrontDoorPassengerDownAccum omit `json:"frontDoorPassengerDownAccum,omitempty"`
		// OmitBackDoorPassengerUpAccum    omit `json:"backDoorPassengerUpAccum,omitempty"`
		// OmitBackDoorPassengerDownAccum  omit `json:"backDoorPassengerDownAccum,omitempty"`
		// OmitTurnstileBattery            omit `json:"turnstileBattery,omitempty"`
		// OmitTurnstileObstruction        omit `json:"turnstileObstruction,omitempty"`
		// OmitTurnstileAnomaliesAccum     omit `json:"turnstileAnomaliesAccum,omitempty"`
		// OmitRefSystem                   omit `json:"refSys"`

		Gstatus                     interface{}              `json:"gstatus"`
		SnDev                       string                   `json:"sn-dev"`
		SnModem                     string                   `json:"sn-modem"`
		SnDisplay                   string                   `json:"sn-display"`
		TimeStamp                   float64                  `json:"timeStamp"`
		IpMask                      map[string][]string      `json:"ipMaskMap"`
		Hostname                    string                   `json:"h"`
		AppVers                     map[string]string        `json:"AVer"`
		SimStatus                   string                   `json:"sS"`
		SimImei                     int64                    `json:"sI"`
		UsosTranspCount             int64                    `json:"uTC"`
		ErrsTranspCount             int64                    `json:"eTC"`
		Volt                        map[string]float64       `json:"vo"`
		Mac                         string                   `json:"m"`
		AppTablesVers               map[string]string        `json:"ATVer"`
		CpuStatus                   []float64                `json:"cS"`
		Dns                         []string                 `json:"dns"`
		UpTime                      string                   `json:"uT"`
		DeviceDataList              []map[string]interface{} `json:"dD"`
		Gateway                     string                   `json:"g"`
		TypeDev                     string                   `json:"tDv"`
		Ruta                        map[string]string        `json:"ruta"`
		TurnstileUpAccum            uint64                   `json:"tsUA"`
		TurnstileDownAccum          uint64                   `json:"tsDA"`
		FrontDoorPassengerUpAccum   uint64                   `json:"fDPUA"`
		FrontDoorPassengerDownAccum uint64                   `json:"fDPDA"`
		BackDoorPassengerUpAccum    uint64                   `json:"bPUA"`
		BackDoorPassengerDownAccum  uint64                   `json:"bDPDA"`
		TurnstileBattery            int64                    `json:"tsBT"`
		TurnstileObstruction        int64                    `json:"tsOB"`
		TurnstileAnomaliesAccum     uint64                   `json:"tsAN"`
		RefSystem                   int                      `json:"refSys"`
	}{
		// Status:                      s,
		Gstatus:                     s.Gstatus,
		SnDev:                       s.SnDev,
		SnModem:                     s.SnModem,
		SnDisplay:                   s.SnDisplay,
		TimeStamp:                   s.TimeStamp,
		IpMask:                      s.IpMask,
		Hostname:                    s.Hostname,
		AppVers:                     s.AppVers,
		SimStatus:                   s.SimStatus,
		SimImei:                     s.SimImei,
		UsosTranspCount:             s.UsosTranspCount,
		ErrsTranspCount:             s.ErrsTranspCount,
		Volt:                        s.Volt,
		Mac:                         s.Mac,
		AppTablesVers:               s.AppTablesVers,
		CpuStatus:                   s.CpuStatus,
		Dns:                         s.Dns,
		UpTime:                      s.UpTime,
		DeviceDataList:              s.DeviceDataList,
		Gateway:                     s.Gateway,
		TypeDev:                     s.TypeDev,
		Ruta:                        s.Ruta,
		TurnstileUpAccum:            s.TurnstileUpAccum,
		TurnstileDownAccum:          s.TurnstileDownAccum,
		FrontDoorPassengerUpAccum:   s.FrontDoorPassengerUpAccum,
		FrontDoorPassengerDownAccum: s.FrontDoorPassengerDownAccum,
		BackDoorPassengerUpAccum:    s.BackDoorPassengerUpAccum,
		BackDoorPassengerDownAccum:  s.BackDoorPassengerDownAccum,
		TurnstileBattery:            s.TurnstileBattery,
		TurnstileObstruction:        s.TurnstileObstruction,
		TurnstileAnomaliesAccum:     s.TurnstileAnomaliesAccum,
		RefSystem:                   s.RefSystem,
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}
