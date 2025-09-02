package remqtt

import (
	"testing"
	"time"
)

func Test_calculateNextReconnectDelay(t *testing.T) {
	type args struct {
		nextTry   time.Time
		step      time.Duration
		nextDelay time.Duration
		maxDelay  time.Duration
	}
	tests := []struct {
		name  string
		args  args
		want  time.Duration
		want1 bool
	}{
		// TODO: Add test cases.
		{
			name: "test_case_1",
			args: args{
				nextTry:   time.Now().Add(1 * time.Minute),
				step:      30 * time.Second,
				nextDelay: 30 * time.Second,
				maxDelay:  60 * time.Second,
			},
			want:  0 * time.Second,
			want1: false,
		},
		{
			name: "test_case_2",
			args: args{
				nextTry:   time.Now().Add(-1 * time.Minute),
				step:      30 * time.Second,
				nextDelay: 59 * time.Second,
				maxDelay:  60 * time.Second,
			},
			want:  0 * time.Second,
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := calculateNextReconnectDelay(tt.args.nextTry, tt.args.step, tt.args.nextDelay, tt.args.maxDelay)
			if got != tt.want {
				t.Errorf("calculateNextReconnectDelay() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("calculateNextReconnectDelay() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
