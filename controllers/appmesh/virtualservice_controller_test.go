package controllers

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_virtualservice "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func Test_virtualServiceReconciler_reconcile(t *testing.T) {
	type fields struct {
		Reconcile func(ctx context.Context, vs *appmesh.VirtualService) error
	}
	type args struct {
		vs *appmesh.VirtualService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name: "virtualService with reconcile error",
			fields: fields{
				Reconcile: func(ctx context.Context, ref *appmesh.VirtualService) error {
					return errors.New("Test Exception")
				},
			},
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Status: appmesh.VirtualServiceStatus{},
				},
			},
			want:    "",
			wantErr: errors.New("Test Exception"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			vsResManager := mock_virtualservice.NewMockResourceManager(ctrl)
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)

			err := k8sClient.Create(ctx, tt.args.vs.DeepCopy())
			assert.NoError(t, err)

			finalizerManager := k8s.NewDefaultFinalizerManager(k8sClient, logr.New(&log.NullLogSink{}))

			recorder := record.NewFakeRecorder(3)

			r := &virtualServiceReconciler{
				k8sClient:        k8sClient,
				finalizerManager: finalizerManager,
				vsResManager:     vsResManager,
				log:              logr.New(&log.NullLogSink{}),
				recorder:         recorder,
			}

			if tt.fields.Reconcile != nil {
				vsResManager.EXPECT().Reconcile(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.Reconcile)
			}

			err = r.reconcile(ctx, reconcile.Request{
				NamespacedName: k8s.NamespacedName(tt.args.vs),
			})
			if tt.wantErr != nil {
				assert.Greater(t, len(recorder.Events), 0)
				assert.Equal(t, "Warning ReconcileError "+tt.wantErr.Error(), <-recorder.Events)
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
