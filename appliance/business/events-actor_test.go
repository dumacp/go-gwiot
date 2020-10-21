package business

import (
	"reflect"
	"testing"
)

func Test_parseEvents(t *testing.T) {
	type args struct {
		msg []byte
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{[]byte(`
{
	"value": {
		"id": 1,
		"coord": "$GPRMC,161940.0,A,0612.751941,N,07534.640897,W,0.0,0.0,100518,4.7,W,A*0B",
		"state": 0
	},
	"timeStamp": 1573827712861,
	"type": "TAMPERING"
}`)},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseEvents(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}
