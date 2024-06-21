package events

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func MessageFromDeviceSTS(group_name string, state *DeviceSTS) []*Message[string, int] {

	if state == nil {
		return nil
	}

	deviceSerial := state.DeviceSerial
	groupName := strings.Split(group_name, "_")[0]
	version := state.ReportVersion
	sendTime := time.Now().UnixMilli()
	timeStamp := int64(state.Timestamp)

	uid := state.Muuid
	if len(uid) == 0 {
		uid = uuid.NewString()
	}

	orgId := func() string {
		split := strings.Split(group_name, "_")
		if len(split) > 1 {
			return split[1]
		}
		return ""
	}()

	messagesGroup := make([]*Message[string, int], 0)

	if state.SystemSTS != nil {
		message := NewMessage().DeviceSerial(deviceSerial).SendTimestamp(sendTime).
			MessageUuid(uid).DevicesPlatformId("").OrganizationId(orgId).
			GroupName(groupName).CreationTimestamp(sendTime).Format("JSON")
		body := map[string]interface{}{
			"t":  timeStamp,
			"rV": version,
		}
		body["temp"] = state.SystemSTS.Temp
		body["cS"] = state.SystemSTS.Cs
		body["uT"] = state.SystemSTS.UpTime
		body["fw"] = state.SystemSTS.RefSystem
		body["diSn"] = state.SystemSTS.DisplaySerial
		body["tp"] = state.SystemSTS.DeviceType
		body["gn"] = groupName
		body["orgI"] = orgId
		body["deSn"] = state.SystemSTS.DeviceSerial

		if state.SystemSTS.Ram != nil {
			body["ram"] = map[string]interface{}{
				"tV": state.SystemSTS.Ram.TotalValue,
				"cV": state.SystemSTS.Ram.CurrentValue,
				"uI": state.SystemSTS.Ram.UnitInformation,
				"tp": state.SystemSTS.Ram.Type,
				"t":  timeStamp}
		}

		if state.SystemSTS.Flash != nil {
			body["flash"] = map[string]interface{}{
				"tV": state.SystemSTS.Flash.TotalValue,
				"cV": state.SystemSTS.Flash.CurrentValue,
				"uI": state.SystemSTS.Flash.UnitInformation,
				"tp": state.SystemSTS.Flash.Type,
				"t":  timeStamp}
		}

		if state.SystemSTS.SDCard != nil {
			body["sd"] = map[string]interface{}{
				"tV": state.SystemSTS.SDCard.TotalValue,
				"cV": state.SystemSTS.SDCard.CurrentValue,
				"uI": state.SystemSTS.SDCard.UnitInformation,
				"tp": state.SystemSTS.SDCard.Type,
				"t":  timeStamp}
		}
		message.AddType("SYSTEM_STS").AddTypeVersion(1).BodyContent(body)
		messagesGroup = append(messagesGroup, message)
	}

	if state.ModemSTS != nil {
		message := NewMessage().DeviceSerial(deviceSerial).SendTimestamp(sendTime).
			MessageUuid(uid).DevicesPlatformId("").OrganizationId(orgId).
			GroupName(groupName).CreationTimestamp(sendTime).Format("JSON")
		body := map[string]interface{}{
			"t":        timeStamp,
			"sn":       state.ModemSTS.ModemSerial,
			"sI":       state.ModemSTS.ModemImei,
			"sS":       state.ModemSTS.SimStatus,
			"simIccid": state.ModemSTS.ModemIccid,
		}
		if state.ModemSTS.Gstatus != nil {
			if gs, ok := state.ModemSTS.Gstatus["gs"].(map[string]interface{}); ok {
				for k, v := range gs {
					body[k] = v
				}
			}
		}
		message.AddType("MODEM_STS").AddTypeVersion(1).BodyContent(body)
		messagesGroup = append(messagesGroup, message)
	}

	if state.NetworkSTS != nil {
		message := NewMessage().DeviceSerial(deviceSerial).SendTimestamp(sendTime).
			MessageUuid(uid).DevicesPlatformId("").OrganizationId(orgId).
			GroupName(groupName).CreationTimestamp(sendTime).Format("JSON")
		body := map[string]interface{}{
			"t":   timeStamp,
			"hn":  state.NetworkSTS.Hostname,
			"mac": state.NetworkSTS.Mac,
			"gw":  state.NetworkSTS.Gateway,
			"ifs": func() map[string]interface{} {
				ifs := make(map[string]interface{})
				for k, v := range state.NetworkSTS.IpMask {
					ifs["n"] = k
					ifs["a"] = v
				}
				return ifs
			}(),
		}
		message.AddType("NETWORK_STS").AddTypeVersion(1).BodyContent(body)
		messagesGroup = append(messagesGroup, message)
	}

	if state.AppsSTS != nil {
		message := NewMessage().DeviceSerial(deviceSerial).SendTimestamp(sendTime).
			MessageUuid(uid).DevicesPlatformId("").OrganizationId(orgId).
			GroupName(groupName).CreationTimestamp(sendTime).Format("JSON")
		body := map[string]interface{}{
			"t":    timeStamp,
			"apps": state.AppsSTS.AppVers,
		}
		message.AddType("APPS_STS").AddTypeVersion(1).BodyContent(body)
		messagesGroup = append(messagesGroup, message)
	}

	if state.PaxcounterSTS != nil {
		message := NewMessage().DeviceSerial(deviceSerial).SendTimestamp(sendTime).
			MessageUuid(uid).DevicesPlatformId("").OrganizationId(orgId).
			GroupName(groupName).CreationTimestamp(sendTime).Format("JSON")
		body := map[string]interface{}{
			"t":    timeStamp,
			"fIn":  state.PaxcounterSTS.FrontInput,
			"fOut": state.PaxcounterSTS.FrontOutput,
			"bIn":  state.PaxcounterSTS.BackInput,
			"bOut": state.PaxcounterSTS.BackOutput,
		}
		message.AddType("PAXCOUNTER_STS").AddTypeVersion(1).BodyContent(body)
		messagesGroup = append(messagesGroup, message)
	}

	return messagesGroup

}
