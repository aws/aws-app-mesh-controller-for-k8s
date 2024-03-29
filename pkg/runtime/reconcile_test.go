package runtime

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

func TestHandleReconcileError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		want    ctrl.Result
		wantErr error
	}{
		{
			name: "input err is nil",
			args: args{
				err: nil,
			},
			want:    ctrl.Result{},
			wantErr: nil,
		},
		{
			name: "input err is RequeueAfterError",
			args: args{
				err: NewRequeueAfterError(errors.New("some error"), 3*time.Second),
			},
			want: ctrl.Result{
				RequeueAfter: 3 * time.Second,
			},
			wantErr: nil,
		},
		{
			name: "input err is RequeueError",
			args: args{
				err: NewRequeueError(errors.New("some error")),
			},
			want: ctrl.Result{
				Requeue: true,
			},
			wantErr: nil,
		},
		{
			name: "input err is other error type",
			args: args{
				err: errors.New("some error"),
			},
			want:    ctrl.Result{},
			wantErr: errors.New("some error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HandleReconcileError(tt.args.err, logr.New(&log.NullLogSink{}))
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
