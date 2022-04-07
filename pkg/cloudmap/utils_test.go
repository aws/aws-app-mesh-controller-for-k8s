package cloudmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_utils_AwaitFunction(t *testing.T) {
	type args struct {
		timeoutSeconds time.Duration
		tickerSeconds  time.Duration
		desiredLen     int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Await function should succeed",
			args: args{
				timeoutSeconds: 10,
				tickerSeconds:  1,
			},
			want: SUCCESS,
		},
		{
			name: "Await function should timeout",
			args: args{
				timeoutSeconds: 2,
				tickerSeconds:  1,
			},
			want: TIMEOUT,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testSlice []int64
			status := AwaitOperationSuccess(tt.args.timeoutSeconds, tt.args.tickerSeconds, func() string {
				if len(testSlice) < 5 {
					testSlice = append(testSlice, 1)
					if tt.want == TIMEOUT {
						time.Sleep(time.Second)
					}
					return FAIL
				}
				return SUCCESS
			})
			assert.Equal(t, tt.want, status)
		})
	}
}
