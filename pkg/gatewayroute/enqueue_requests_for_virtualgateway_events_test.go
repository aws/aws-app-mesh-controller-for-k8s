package gatewayroute

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
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

func Test_enqueueRequestsForVirtualGatewayEvents_Update(t *testing.T) {
	gr1 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-1",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				Namespace: aws.String("vg-ns"),
			},
		},
	}
	gr2 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-2",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				Namespace: aws.String("vg-ns"),
			},
		},
	}

	type env struct {
		gatewayRoutes []*appmesh.GatewayRoute
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
			name: "virtualGatewayActive status changed",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{gr1, gr2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &appmesh.VirtualGateway{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-vg",
							UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
							Namespace: "vg-ns",
						},
					},
					ObjectNew: &appmesh.VirtualGateway{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-vg",
							UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
							Namespace: "vg-ns",
						},
						Status: appmesh.VirtualGatewayStatus{
							Conditions: []appmesh.VirtualGatewayCondition{
								{
									Type:   appmesh.VirtualGatewayActive,
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
			name: "virtualGatewayActive status changed",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{gr1, gr2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &appmesh.VirtualGateway{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-vg",
							UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
							Namespace: "vg-ns",
						},
					},
					ObjectNew: &appmesh.VirtualGateway{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-vg",
							UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
							Namespace: "vg-ns",
						},
						Status: appmesh.VirtualGatewayStatus{
							Conditions: []appmesh.VirtualGatewayCondition{
								{
									Type:   appmesh.VirtualGatewayActive,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(gr1),
				},
				{
					NamespacedName: k8s.NamespacedName(gr2),
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
			h := &enqueueRequestsForVirtualGatewayEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, gr := range tt.env.gatewayRoutes {
				err := k8sClient.Create(ctx, gr.DeepCopy())
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

func Test_enqueueRequestsForVirtualGatewayEvents_enqueueGatewayRoutesForVirtualGateway(t *testing.T) {
	vg := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-vg",
			UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
			Namespace: "vg-ns",
		},
		Status: appmesh.VirtualGatewayStatus{
			Conditions: []appmesh.VirtualGatewayCondition{
				{
					Type:   appmesh.VirtualGatewayActive,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	grWithoutVirtualGatewayRef := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-without-vg-ref",
		},
		Spec: appmesh.GatewayRouteSpec{},
	}
	grWithNonMatchingVirtualGatewayRef := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-with-non-matching-vg-ref",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "0d65db83-1b4c-40aa-90ba-57064dd73c98",
				Namespace: aws.String("vg-ns"),
			},
		},
	}
	grWithMatchingVirtualGatewayRef_1 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-with-non-matching-vg-ref-1",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				Namespace: aws.String("vg-ns"),
			},
		},
	}
	grWithMatchingVirtualGatewayRef_2 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-with-non-matching-vg-ref-2",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				Namespace: aws.String("vg-ns"),
			},
		},
	}

	type env struct {
		gatewayRoutes []*appmesh.GatewayRoute
	}
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name         string
		env          env
		args         args
		wantRequests []reconcile.Request
	}{
		{
			name: "gr without virtualGatewayRef shouldn't be enqueued",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{
					grWithoutVirtualGatewayRef,
				},
			},
			args: args{
				vg: vg,
			},
			wantRequests: nil,
		},
		{
			name: "gr with non-matching virtualGatewayRef shouldn't be enqueued",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{
					grWithNonMatchingVirtualGatewayRef,
				},
			},
			args: args{
				vg: vg,
			},
			wantRequests: nil,
		},
		{
			name: "gr with matching virtualGatewayRef should be enqueued",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{
					grWithMatchingVirtualGatewayRef_1,
				},
			},
			args: args{
				vg: vg,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(grWithMatchingVirtualGatewayRef_1),
				},
			},
		},
		{
			name: "multiple gr should enqueue correctly",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{
					grWithoutVirtualGatewayRef,
					grWithNonMatchingVirtualGatewayRef,
					grWithMatchingVirtualGatewayRef_1,
					grWithMatchingVirtualGatewayRef_2,
				},
			},
			args: args{
				vg: vg,
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(grWithMatchingVirtualGatewayRef_1),
				},
				{
					NamespacedName: k8s.NamespacedName(grWithMatchingVirtualGatewayRef_2),
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
			h := &enqueueRequestsForVirtualGatewayEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, gr := range tt.env.gatewayRoutes {
				err := k8sClient.Create(ctx, gr.DeepCopy())
				assert.NoError(t, err)
			}

			h.enqueueGatewayRoutesForVirtualGateway(ctx, queue, tt.args.vg)
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
