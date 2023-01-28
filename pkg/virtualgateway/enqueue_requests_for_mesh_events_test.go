package virtualgateway

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/workqueue"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func Test_enqueueRequestsForMeshEvents_Update(t *testing.T) {
	vg1 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-1",
		},
		Spec: appmesh.VirtualGatewaySpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}
	vg2 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-2",
		},
		Spec: appmesh.VirtualGatewaySpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}

	type env struct {
		virtualGateways []*appmesh.VirtualGateway
	}
	type args struct {
		e event.UpdateEvent
	}
	tests := []struct {
		name         string
		env          env
		args         args
		wantRequests []reconcile.Request
	}{
		{
			name: "meshActive status changed",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vg1, vg2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					ObjectNew: &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
						Status: appmesh.MeshStatus{
							Conditions: []appmesh.MeshCondition{
								{
									Type:   appmesh.MeshActive,
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			wantRequests: nil,
		},
		{
			name: "meshActive status changed",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vg1, vg2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					ObjectNew: &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
						Status: appmesh.MeshStatus{
							Conditions: []appmesh.MeshCondition{
								{
									Type:   appmesh.MeshActive,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vg1),
				},
				{
					NamespacedName: k8s.NamespacedName(vg2),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			h := &enqueueRequestsForMeshEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, vg := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, vg.DeepCopy())
				assert.NoError(t, err)
			}

			h.Update(tt.args.e, queue)
			var gotRequests []reconcile.Request
			queueLen := queue.Len()
			for i := 0; i < queueLen; i++ {
				item, _ := queue.Get()
				gotRequests = append(gotRequests, item.(reconcile.Request))
			}

			opt := cmpopts.SortSlices(compareReconcileRequest)
			assert.True(t, cmp.Equal(tt.wantRequests, gotRequests, opt), "diff: %v", cmp.Diff(tt.wantRequests, gotRequests, opt))

		})
	}
}

func Test_enqueueRequestsForMeshEvents_enqueueVirtualGatewaysForMesh(t *testing.T) {
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
		},
		Status: appmesh.MeshStatus{
			Conditions: []appmesh.MeshCondition{
				{
					Type:   appmesh.MeshActive,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	vgWithoutMeshRef := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-without-mesh-ref",
		},
		Spec: appmesh.VirtualGatewaySpec{},
	}
	vgWithNonMatchingMeshRef := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-with-non-matching-mesh-ref",
		},
		Spec: appmesh.VirtualGatewaySpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "0d65db83-1b4c-40aa-90ba-57064dd73c98",
			},
		},
	}
	vgWithMatchingMeshRef_1 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-with-non-matching-mesh-ref-1",
		},
		Spec: appmesh.VirtualGatewaySpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}
	vgWithMatchingMeshRef_2 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vg-with-non-matching-mesh-ref-2",
		},
		Spec: appmesh.VirtualGatewaySpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}

	type env struct {
		virtualGateways []*appmesh.VirtualGateway
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name         string
		env          env
		args         args
		wantRequests []reconcile.Request
	}{
		{
			name: "vg without meshRef shouldn't be enqueued",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithoutMeshRef,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: nil,
		},
		{
			name: "vg with non-matching meshRef shouldn't be enqueued",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNonMatchingMeshRef,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: nil,
		},
		{
			name: "vg with matching meshRef should be enqueued",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithMatchingMeshRef_1,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vgWithMatchingMeshRef_1),
				},
			},
		},
		{
			name: "multiple vg should enqueue correctly",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithoutMeshRef,
					vgWithNonMatchingMeshRef,
					vgWithMatchingMeshRef_1,
					vgWithMatchingMeshRef_2,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vgWithMatchingMeshRef_1),
				},
				{
					NamespacedName: k8s.NamespacedName(vgWithMatchingMeshRef_2),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			h := &enqueueRequestsForMeshEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, vg := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, vg.DeepCopy())
				assert.NoError(t, err)
			}

			h.enqueueVirtualGatewaysForMesh(ctx, queue, tt.args.ms)
			var gotRequests []reconcile.Request
			queueLen := queue.Len()
			for i := 0; i < queueLen; i++ {
				item, _ := queue.Get()
				gotRequests = append(gotRequests, item.(reconcile.Request))
			}

			opt := cmpopts.SortSlices(compareReconcileRequest)
			assert.True(t, cmp.Equal(tt.wantRequests, gotRequests, opt), "diff: %v", cmp.Diff(tt.wantRequests, gotRequests, opt))
		})
	}
}

func compareReconcileRequest(a reconcile.Request, b reconcile.Request) bool {
	return a.String() < b.String()
}
