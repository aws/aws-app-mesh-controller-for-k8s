package virtualservice

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
	vs1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vs-1",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}
	vs2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vs-2",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}

	type env struct {
		virtualServices []*appmesh.VirtualService
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
			name: "meshActive status un-changed",
			env: env{
				virtualServices: []*appmesh.VirtualService{vs1, vs2},
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
				virtualServices: []*appmesh.VirtualService{vs1, vs2},
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
					NamespacedName: k8s.NamespacedName(vs1),
				},
				{
					NamespacedName: k8s.NamespacedName(vs2),
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

			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs.DeepCopy())
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

func Test_enqueueRequestsForMeshEvents_enqueueVirtualServicesForMesh(t *testing.T) {
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
	vsWithoutMeshRef := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vr-without-mesh-ref",
		},
		Spec: appmesh.VirtualServiceSpec{},
	}
	vsWithNonMatchingMeshRef := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vr-with-non-matching-mesh-ref",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "0d65db83-1b4c-40aa-90ba-57064dd73c98",
			},
		},
	}
	vsWithMatchingMeshRef_1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vr-with-non-matching-mesh-ref-1",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}
	vsWithMatchingMeshRef_2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vr-with-non-matching-mesh-ref-2",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
			},
		},
	}

	type env struct {
		virtualServices []*appmesh.VirtualService
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
			name: "vs without meshRef shouldn't be enqueued",
			env: env{
				virtualServices: []*appmesh.VirtualService{
					vsWithoutMeshRef,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: nil,
		},
		{
			name: "vs with non-matching meshRef shouldn't be enqueued",
			env: env{
				virtualServices: []*appmesh.VirtualService{
					vsWithNonMatchingMeshRef,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: nil,
		},
		{
			name: "vs with matching meshRef should be enqueued",
			env: env{
				virtualServices: []*appmesh.VirtualService{
					vsWithMatchingMeshRef_1,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vsWithMatchingMeshRef_1),
				},
			},
		},
		{
			name: "multiple vs should enqueue correctly",
			env: env{
				virtualServices: []*appmesh.VirtualService{
					vsWithoutMeshRef,
					vsWithNonMatchingMeshRef,
					vsWithMatchingMeshRef_1,
					vsWithMatchingMeshRef_2,
				},
			},
			args: args{
				ms: ms,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vsWithMatchingMeshRef_1),
				},
				{
					NamespacedName: k8s.NamespacedName(vsWithMatchingMeshRef_2),
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

			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			h.enqueueVirtualServicesForMesh(ctx, queue, tt.args.ms)
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
