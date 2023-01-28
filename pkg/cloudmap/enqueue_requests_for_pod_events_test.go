package cloudmap

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

func Test_enqueueRequestsForPodEvents_Create(t *testing.T) {
	vn1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "global",
			Name:      "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "testapp",
				},
			},
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
				},
			},
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		e event.CreateEvent
	}

	tests := []struct {
		name         string
		env          env
		args         args
		wantRequests []reconcile.Request
	}{
		{
			name: "Pod is Created",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn1},
			},
			args: args{
				e: event.CreateEvent{
					Object: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vn1),
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
			h := &enqueueRequestsForPodEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
				assert.NoError(t, err)
			}

			h.Create(tt.args.e, queue)
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

func Test_enqueueRequestsForPodEvents_Update(t *testing.T) {
	vn1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "global",
			Name:      "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "testapp",
				},
			},
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
				},
			},
		},
	}
	vn2 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "global",
			Name:      "vn-2",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "othertestapp",
				},
			},
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
				},
			},
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
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
			name: "Pod status changed",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn1, vn2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
					},
					ObjectNew: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodPending,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
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
			name: "Pod is Ready",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn1, vn2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
					},
					ObjectNew: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.ContainersReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vn1),
				},
			},
		},
		{
			name: "Pod labels changed",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn1, vn2},
			},
			args: args{
				e: event.UpdateEvent{
					ObjectOld: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
						},
					},
					ObjectNew: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "othertestapp",
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.ContainersReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vn1),
				},
				{
					NamespacedName: k8s.NamespacedName(vn2),
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
			h := &enqueueRequestsForPodEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
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

func Test_enqueueRequestsForPodEvents_Delete(t *testing.T) {
	vn1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "global",
			Name:      "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "testapp",
				},
			},
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
				},
			},
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		e event.DeleteEvent
	}

	now := metav1.Now()
	tests := []struct {
		name         string
		env          env
		args         args
		wantRequests []reconcile.Request
	}{
		{
			name: "Pod is Created",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn1},
			},
			args: args{
				e: event.DeleteEvent{
					Object: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test_pod1",
							UID:  "b387048d-aba8-6235-9a11-5343764c8ab",
							Labels: map[string]string{
								"app": "testapp",
							},
							DeletionTimestamp: &now,
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantRequests: []reconcile.Request{
				{
					NamespacedName: k8s.NamespacedName(vn1),
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
			h := &enqueueRequestsForPodEvents{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
				assert.NoError(t, err)
			}

			h.Delete(tt.args.e, queue)
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
