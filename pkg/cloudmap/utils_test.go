package cloudmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_utils_AwaitOperationSuccess(t *testing.T) {
	var testSlice []int64
	tests := []struct {
		name           string
		timeoutSeconds time.Duration
		tickerSeconds  time.Duration
		want           string
		f              func() string
	}{
		{
			name:           "Await function should succeed",
			timeoutSeconds: 5,
			tickerSeconds:  1,
			want:           SUCCESS,
			f: func() string {
				time.Sleep(time.Second)
				return SUCCESS
			},
		},
		{
			name:           "Await function should timeout because defined function takes too long",
			timeoutSeconds: 2,
			tickerSeconds:  1,
			want:           TIMEOUT,
			f: func() string {
				time.Sleep(3 * time.Second)
				return SUCCESS
			},
		},
		{
			name:           "Await function should timeout because list does not reach desired size",
			timeoutSeconds: 2,
			tickerSeconds:  1,
			want:           TIMEOUT,
			f: func() string {
				if len(testSlice) < 5 {
					testSlice = append(testSlice, 1)
					time.Sleep(time.Second)
					return FAIL
				}
				return SUCCESS
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := AwaitOperationSuccess(tt.timeoutSeconds, tt.tickerSeconds, tt.f)
			assert.Equal(t, tt.want, status)
		})
	}
}
