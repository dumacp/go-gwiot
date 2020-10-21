package controlext

import "testing"

func Test_pingFunc(t *testing.T) {
	type args struct {
		testIP string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test1",
			args{
				testIP: "8.0.0.1",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := pingFunc(tt.args.testIP); (err != nil) != tt.wantErr {
				t.Errorf("pingFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
