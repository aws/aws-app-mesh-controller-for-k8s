package cloudmap

import (
	"context"
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/cache"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
				log:       log.NullLogger{},
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
				log:       log.NullLogger{},
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			response := m.isCloudMapServiceCreated(ctx, tt.args.vn)
			assert.True(t, cmp.Equal(tt.want, response), "diff", cmp.Diff(tt.want, response))
		})
	}
}

func Test_defaultResourceManager_isCloudMapServiceUpdated(t *testing.T) {
	type args struct {
		vn         *appmesh.VirtualNode
		svcSummary *serviceSummary
	}
	testARN := "cloudMapARN"
	testCloudMapNamespace := "CloudMapNamespace"
	testCloudMapService := "CloudMapService"
	var originalCloudMapDnsTTL int64 = 300
	var newCloudMapDnsTTL int64 = 200
	tests := []struct {
		name                  string
		args                  args
		wantCloudMapDnsConfig *servicediscovery.DnsConfig
	}{
		{
			name: "virtualNode has cloudMap service annotation and specs",
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
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: testCloudMapNamespace,
								ServiceName:   testCloudMapService,
							},
						},
					},
				},
				svcSummary: &serviceSummary{
					serviceID:  testCloudMapService,
					serviceARN: &testARN,
					DnsConfig: &servicediscovery.DnsConfig{
						RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
						DnsRecords: []*servicediscovery.DnsRecord{
							{
								Type: awssdk.String(servicediscovery.RecordTypeA),
								TTL:  awssdk.Int64(originalCloudMapDnsTTL),
							},
						},
					},
				},
			},
			wantCloudMapDnsConfig: &servicediscovery.DnsConfig{
				RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
				DnsRecords: []*servicediscovery.DnsRecord{
					{
						Type: awssdk.String(servicediscovery.RecordTypeA),
						TTL:  awssdk.Int64(newCloudMapDnsTTL),
					},
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
				k8sClient:           k8sClient,
				log:                 log.NullLogger{},
				serviceSummaryCache: cache.NewLRUExpireCache(defaultServiceCacheMaxSize),
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			nsSummary := &servicediscovery.NamespaceSummary{
				Name: &testCloudMapNamespace,
				Id:   &testCloudMapNamespace,
			}
			svc := &servicediscovery.Service{
				Arn:       &testARN,
				Id:        &testCloudMapService,
				DnsConfig: tt.args.svcSummary.DnsConfig,
			}
			svcSummary := m.addCloudMapServiceToServiceSummaryCache(nsSummary, svc)
			svcSummary.DnsConfig = tt.wantCloudMapDnsConfig
			m.updateCloudMapServiceInServiceSummaryCache(nsSummary, svcSummary, *svc.Id)
			key := m.buildCloudMapServiceSummaryCacheKey(nsSummary, *svc.Id)
			val, _ := m.serviceSummaryCache.Get(key)
			opts := cmp.Options{
				cmpopts.EquateEmpty(),
				cmp.AllowUnexported(serviceSummary{}),
			}
			assert.True(t, cmp.Equal(svcSummary, val, opts))
		})
	}
}

func Test_defaultResourceManager_updateCloudMapServiceForOtherNamespaceTypes(t *testing.T) {
	type args struct {
		vn         *appmesh.VirtualNode
		svcSummary *serviceSummary
	}
	testARN := "cloudMapARN"
	testCloudMapNamespace := "CloudMapNamespace"
	testCloudMapService := "CloudMapService"
	var cloudMapDnsTTL int64 = 300
	tests := []struct {
		name   string
		args   args
		want   bool
		nsType string
	}{
		{
			name: "cloudmap service should not update for http namespace type",
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
				svcSummary: &serviceSummary{
					serviceID:  testCloudMapService,
					serviceARN: &testARN,
					DnsConfig: &servicediscovery.DnsConfig{
						RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
						DnsRecords: []*servicediscovery.DnsRecord{
							{
								Type: awssdk.String(servicediscovery.RecordTypeA),
								TTL:  awssdk.Int64(cloudMapDnsTTL),
							},
						},
					},
				},
			},
			want:   false,
			nsType: servicediscovery.NamespaceTypeHttp,
		},
		{
			name: "cloudmap service should not update for public namespace type",
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
				svcSummary: &serviceSummary{
					serviceID:  testCloudMapService,
					serviceARN: &testARN,
					DnsConfig: &servicediscovery.DnsConfig{
						RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
						DnsRecords: []*servicediscovery.DnsRecord{
							{
								Type: awssdk.String(servicediscovery.RecordTypeA),
								TTL:  awssdk.Int64(cloudMapDnsTTL),
							},
						},
					},
				},
			},
			want:   false,
			nsType: servicediscovery.NamespaceTypeDnsPublic,
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
				log:       log.NullLogger{},
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			nsSummary := &servicediscovery.NamespaceSummary{
				Name: &testCloudMapNamespace,
				Id:   &testCloudMapNamespace,
				Type: &tt.nsType,
			}
			svcSummary, err := m.updateCloudMapService(ctx, tt.args.svcSummary, tt.args.vn, nsSummary, tt.args.svcSummary.serviceID, cloudMapDnsTTL)
			assert.NoError(t, err)
			opts := cmp.Options{
				cmpopts.EquateEmpty(),
				cmp.AllowUnexported(serviceSummary{}),
			}
			assert.True(t, cmp.Equal(svcSummary, tt.args.svcSummary, opts))
		})
	}
}
