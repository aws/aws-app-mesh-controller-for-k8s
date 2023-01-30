package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_references "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/cache"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

func Test_defaultResourceManager_updateCRDVirtualNode(t *testing.T) {
	type args struct {
		vn         *appmesh.VirtualNode
		svcSummary *serviceSummary
	}
	testARN := "cloudMapARN"
	tests := []struct {
		name    string
		args    args
		wantVN  *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "virtualNode needs cloudMap service ARN patch",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "vn-1",
						Annotations: map[string]string{},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
				svcSummary: &serviceSummary{
					serviceID:  "cloudMapService",
					serviceARN: &testARN,
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
					Annotations: map[string]string{
						"cloudMapServiceARN": "cloudMapARN",
					},
				},
			},
		},
		{
			name: "virtualNode patch flow with no ARN",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "vn-1",
						Annotations: map[string]string{},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
				svcSummary: &serviceSummary{
					serviceID: "cloudMapService",
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
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
			m := &defaultResourceManager{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualNode(ctx, tt.args.vn, tt.args.svcSummary)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVN := &appmesh.VirtualNode{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vn), gotVN)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVN, gotVN, opts), "diff", cmp.Diff(tt.wantVN, gotVN, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isCloudMapServiceCreated(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualNode has cloudMap service annotation",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
						Annotations: map[string]string{
							"cloudMapServiceARN": "cloudMapARN",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			want: true,
		},
		{
			name: "virtualNode doesn't have cloudMap service annotation",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := &defaultResourceManager{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			response := m.isCloudMapServiceCreated(ctx, tt.args.vn)
			assert.True(t, cmp.Equal(tt.want, response), "diff", cmp.Diff(tt.want, response))
		})
	}
}

func Test_defaultResourceManager_reconcile_arePodsReconciled(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name              string
		args              args
		shouldResolvePods bool
	}{
		{
			name: "virtualNode has pod selector, calls endpoint resolver to find pods",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
						Annotations: map[string]string{
							"cloudMapServiceARN": "cloudMapARN",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
					Spec: appmesh.VirtualNodeSpec{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"label": "app",
							},
						},
						MeshRef: &appmesh.MeshReference{},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cmnamespace",
								ServiceName:   "cmservice",
							},
						},
					},
				},
			},
			shouldResolvePods: true,
		},
		{
			name: "virtualNode doesn't have a pod selector, no pods",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
						Annotations: map[string]string{
							"cloudMapServiceARN": "cloudMapARN",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
					Spec: appmesh.VirtualNodeSpec{
						MeshRef: &appmesh.MeshReference{},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cmnamespace",
								ServiceName:   "cmservice",
							},
						},
					},
				},
			},
			shouldResolvePods: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			referencesResolver := mock_references.NewMockResolver(ctrl)
			cloudMapSDK := services.NewMockCloudMap(ctrl)
			virtualNodeEndpointResolver := NewMockVirtualNodeEndpointResolver(ctrl)
			instancesReconciler := NewMockInstancesReconciler(ctrl)

			mesh := &appmesh.Mesh{}
			svcSummary := serviceSummary{}

			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			cloudMapNamespace := servicediscovery.NamespaceSummary{Id: awssdk.String("namespace")}

			m := &defaultResourceManager{
				k8sClient:                   k8sClient,
				log:                         logr.New(&log.NullLogSink{}),
				referencesResolver:          referencesResolver,
				namespaceSummaryCache:       cache.NewLRUExpireCache(1),
				serviceSummaryCache:         cache.NewLRUExpireCache(1),
				cloudMapSDK:                 cloudMapSDK,
				virtualNodeEndpointResolver: virtualNodeEndpointResolver,
				instancesReconciler:         instancesReconciler,
			}

			m.namespaceSummaryCache.Add(tt.args.vn.Spec.ServiceDiscovery.AWSCloudMap.NamespaceName, &cloudMapNamespace, 1*time.Minute)
			m.serviceSummaryCache.Add("namespace/"+tt.args.vn.Spec.ServiceDiscovery.AWSCloudMap.ServiceName, &svcSummary, 1*time.Minute)

			referencesResolver.EXPECT().
				ResolveMeshReference(ctx, *tt.args.vn.Spec.MeshRef).
				Return(mesh, nil)

			var expectedReadyPods []*corev1.Pod
			var expectedNotReadyPods []*corev1.Pod
			if tt.shouldResolvePods {
				expectedReadyPods = []*corev1.Pod{{}, {}}
				expectedNotReadyPods = []*corev1.Pod{{}}
				virtualNodeEndpointResolver.EXPECT().
					Resolve(ctx, tt.args.vn).
					Return(expectedReadyPods, expectedNotReadyPods, nil, nil)
			}

			//ensure we pass the correct pods to the reconciler
			instancesReconciler.EXPECT().
				Reconcile(ctx, mesh, tt.args.vn, svcSummary, expectedReadyPods, expectedNotReadyPods, map[string]nodeAttributes{}).
				Return(nil)

			err := m.Reconcile(context.TODO(), tt.args.vn)
			assert.NoError(t, err)
		})
	}
}
