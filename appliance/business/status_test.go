package business

import (
	"reflect"
	"testing"
)

func Test_parseStatus(t *testing.T) {
	type args struct {
		msg []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "test_parseStatus_1",
			args: args{
				[]byte(`
				{"gstatus":{"temperature":0,"mode":"","band":"","rsrq":0,"sinr":0,"tac":0,"cellid":0,"systemMode":"","resetCounter":0,"currentTime":0},"sn-dev":"FLO-W7-0109","sn-modem":"35907206570924","sn-display":" ","timeStamp":1601066516.777,"ipMaskMap":{"eth0":["192.168.188.22/24","fe80::201:daff:fe18:7116/64"],"lo":["127.0.0.1/8","::1/128"]},"hostname":"FLO-W7-0109","AppVers":null,"simStatus":"SIM ERROR","simImei":0,"usosTranspCount":0,"errsTranspCount":0,"volt":{"current":11.488,"high":11.728,"low":11.216000000000001},"mac":"00:01:da:18:71:16","AppTablesVers":null,"cpuStatus":[0.5,4.5,6.5],"dns":[],"upTime":" 15:41:50 up 15:51, load average: 0.01, 0.09, 0.13","deviceDataList":[{"currentValue":310988,"totalValue":1026444,"type":"MEM","unitInformation":"KiB"},{"currentValue":375,"totalValue":3150,"type":"FLASH","unitInformation":"MiB"},{"currentValue":304,"totalValue":7445,"type":"SD","unitInformation":"MiB"}],"gateway":"","type-dev":"OMVZ7","ruta":null,"turnstileUpAccum":0,"turnstileDownAccum":0,"frontDoorPassengerUpAccum":0,"frontDoorPassengerDownAccum":0,"backDoorPassengerUpAccum":0,"backDoorPassengerDownAccum":0,"turnstileBattery":-1,"turnstileObstruction":-1,"turnstileAnomaliesAccum":-1,"refSys":8}
				`),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStatus(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusMsg_ReplaceKeys(t *testing.T) {

	data := []byte(`
	{"gstatus":{"temperature":0,"mode":"","band":"","rsrq":0,"sinr":0,"tac":0,"cellid":0,"systemMode":"","resetCounter":0,"currentTime":0},"sn-dev":"FLO-W7-0109","sn-modem":"35907206570924","sn-display":" ","timeStamp":1601066516.777,"ipMaskMap":{"eth0":["192.168.188.22/24","fe80::201:daff:fe18:7116/64"],"lo":["127.0.0.1/8","::1/128"]},"hostname":"FLO-W7-0109","AppVers":null,"simStatus":"SIM ERROR","simImei":0,"usosTranspCount":0,"errsTranspCount":0,"volt":{"current":11.488,"high":11.728,"low":11.216000000000001},"mac":"00:01:da:18:71:16","AppTablesVers":null,"cpuStatus":[0.5,4.5,6.5],"dns":[],"upTime":" 15:41:50 up 15:51, load average: 0.01, 0.09, 0.13","deviceDataList":[{"currentValue":310988,"totalValue":1026444,"type":"MEM","unitInformation":"KiB"},{"currentValue":375,"totalValue":3150,"type":"FLASH","unitInformation":"MiB"},{"currentValue":304,"totalValue":7445,"type":"SD","unitInformation":"MiB"}],"gateway":"","type-dev":"OMVZ7","ruta":null,"turnstileUpAccum":0,"turnstileDownAccum":0,"frontDoorPassengerUpAccum":0,"frontDoorPassengerDownAccum":0,"backDoorPassengerUpAccum":0,"backDoorPassengerDownAccum":0,"turnstileBattery":-1,"turnstileObstruction":-1,"turnstileAnomaliesAccum":-1,"refSys":8}
	`)

	statusInt := parseStatus(data)
	t.Errorf("statusInt parse -> %v", statusInt)

	status, ok := statusInt.(*StatusMsg)
	if !ok {
		t.Errorf("status parse error -> %v", status)
	}
	t.Errorf("status parse -> %v", status)

	var want *StatusMsg

	if got := status.ReplaceKeys(); !reflect.DeepEqual(got, want) {
		t.Errorf("StatusMsg.ReplaceKeys() = %v, want %v", got, want)
	}
}

func TestStatusMsg_OnlyChangeKeys(t *testing.T) {

	data := []byte(`
	{"gstatus":{"temperature":34,"mode":"ONLINE","band":"1900","rsrq":2,"sinr":0,"tac":10004,"cellid":0,"systemMode":"WCDMA","resetCounter":0,"currentTime":2712},"sn-dev":"FLO-W7-0109","sn-modem":"35907206570924","sn-display":" ","timeStamp":1601174267.776,"ipMaskMap":{"eth0":["192.168.188.22/24","fe80::201:daff:fe18:7116/64"],"lo":["127.0.0.1/8","::1/128"]},"hostname":"FLO-W7-0109","AppVers":null,"simStatus":"SIM ERROR","simImei":359072065760924,"usosTranspCount":0,"errsTranspCount":0,"volt":{"current":11.504,"high":11.728,"low":11.216000000000001},"mac":"00:01:da:18:71:16","AppTablesVers":null,"cpuStatus":[1,3,2.5],"dns":["192.168.188.188"],"upTime":" 21:37:45 up 21:47, load average: 0.02, 0.06, 0.05","deviceDataList":[{"currentValue":328728,"totalValue":1026444,"type":"MEM","unitInformation":"KiB"},{"currentValue":375,"totalValue":3150,"type":"FLASH","unitInformation":"MiB"},{"currentValue":314,"totalValue":7445,"type":"SD","unitInformation":"MiB"}],"gateway":"192.168.188.188","type-dev":"OMVZ7","ruta":null,"turnstileUpAccum":0,"turnstileDownAccum":0,"frontDoorPassengerUpAccum":0,"frontDoorPassengerDownAccum":0,"backDoorPassengerUpAccum":0,"backDoorPassengerDownAccum":0,"turnstileBattery":-1,"turnstileObstruction":-1,"turnstileAnomaliesAccum":-1,"refSys":8}
	`)

	statusInt := parseStatus(data)
	t.Errorf("statusInt parse -> %v", statusInt)

	status := statusInt.(*StatusMsg).ReplaceKeys()
	// if !ok {
	// 	t.Errorf("status parse error -> %v", status)
	// }
	t.Errorf("status parse -> %v", status)

	dataLast := []byte(`
	{"gstatus":{"temperature":34,"mode":"ONLINE","band":"1900","rsrq":0,"sinr":0,"tac":10004,"cellid":0,"systemMode":"WCDMA","resetCounter":0,"currentTime":2712},"sn-dev":"FLO-W7-0109","sn-modem":"35907206570924","sn-display":" ","timeStamp":1601174267.776,"ipMaskMap":{"eth0":["192.168.188.22/24","fe80::201:daff:fe18:7116/64"],"lo":["127.0.0.1/8","::1/128"]},"hostname":"FLO-W7-0109","AppVers":null,"simStatus":"SIM ERROR","simImei":359072065760924,"usosTranspCount":0,"errsTranspCount":0,"volt":{"current":11.504,"high":11.728,"low":11.216000000000001},"mac":"00:01:da:18:71:16","AppTablesVers":null,"cpuStatus":[1,3,2.5],"dns":["192.168.188.188"],"upTime":" 21:37:45 up 21:47, load average: 0.02, 0.06, 0.05","deviceDataList":[{"currentValue":328728,"totalValue":1026444,"type":"MEM","unitInformation":"KiB"},{"currentValue":375,"totalValue":3150,"type":"FLASH","unitInformation":"MiB"},{"currentValue":314,"totalValue":7445,"type":"SD","unitInformation":"MiB"}],"gateway":"192.168.188.188","type-dev":"OMVZ7","ruta":null,"turnstileUpAccum":0,"turnstileDownAccum":0,"frontDoorPassengerUpAccum":0,"frontDoorPassengerDownAccum":0,"backDoorPassengerUpAccum":0,"backDoorPassengerDownAccum":0,"turnstileBattery":-1,"turnstileObstruction":-1,"turnstileAnomaliesAccum":-1,"refSys":8}
	`)

	statusLastInt := parseStatus(dataLast)
	t.Errorf("statusInt parse -> %v", statusLastInt)

	statusLast := statusLastInt.(*StatusMsg).ReplaceKeys()
	// if !ok {
	// 	t.Errorf("status parse error -> %v", statusLast)
	// }
	t.Errorf("status Last parse -> %v", statusLast)

	var want *StatusMsg

	if got := status.OnlyChanges(statusLast); !reflect.DeepEqual(got, want) {
		t.Errorf("StatusMsg.OnlyChanges() = %v, want %v", got, want)
	}
}
