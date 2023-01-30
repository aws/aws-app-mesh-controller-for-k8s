package controllers

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_mesh "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func Test_meshReconciler_reconcile(t *testing.T) {
	type fields struct {
		Reconcile func(ctx context.Context, ms *appmesh.Mesh) error
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name: "mesh with reconcile error",
			fields: fields{
				Reconcile: func(ctx context.Context, ref *appmesh.Mesh) error {
					return errors.New("Test Exception")
				},
			},
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mesh-1",
					},
					Status: appmesh.MeshStatus{
						MeshARN: aws.String("arn-1"),
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
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
			meshResManager := mock_mesh.NewMockResourceManager(ctrl)
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)

			err := k8sClient.Create(ctx, tt.args.ms.DeepCopy())
			assert.NoError(t, err)

			finalizerManager := k8s.NewDefaultFinalizerManager(k8sClient, logr.New(&log.NullLogSink{}))

			recorder := record.NewFakeRecorder(3)

			r := &meshReconciler{
				k8sClient:        k8sClient,
				finalizerManager: finalizerManager,
				meshResManager:   meshResManager,
				log:              logr.New(&log.NullLogSink{}),
				recorder:         recorder,
			}

			if tt.fields.Reconcile != nil {
				meshResManager.EXPECT().Reconcile(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.Reconcile)
			}

			err = r.reconcile(ctx, reconcile.Request{
				NamespacedName: k8s.NamespacedName(tt.args.ms),
			})
			if tt.wantErr != nil {
				assert.Greater(t, len(recorder.Events), 0)
				assert.Equal(t, "Warning ReconcileError "+tt.wantErr.Error(), <-recorder.Events)
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
