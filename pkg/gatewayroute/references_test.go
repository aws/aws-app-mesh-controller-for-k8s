package gatewayroute

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test_ExtractVirtualServiceReferences(t *testing.T) {
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name string
		args args
		want []appmesh.VirtualServiceReference
	}{
		{
			name: "single GRPC route",
			args: args{
				gr: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						GRPCRoute: &appmesh.GRPCGatewayRoute{
							Action: appmesh.GRPCGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualServiceReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vs-1",
				},
			},
		},
		{
			name: "single HTTP route",
			args: args{
				gr: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualServiceReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vs-1",
				},
			},
		},
		{
			name: "single HTTP2 route",
			args: args{
				gr: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						HTTP2Route: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualServiceReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vs-1",
				},
			},
		},
		{
			name: "multiple routes",
			args: args{
				gr: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
						HTTP2Route: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-2",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualServiceReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vs-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vs-2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVirtualServiceReferences(tt.args.gr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVirtualServiceReferenceIndexFunc(t *testing.T) {
	type args struct {
		obj runtime.Object
	}
	tests := []struct {
		name string
		args args
		want []types.NamespacedName
	}{
		{
			name: "single gatewayRoute - with namespace",
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.GatewayRouteSpec{
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("other-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Namespace: "other-ns",
					Name:      "vs-1",
				},
			},
		},
		{
			name: "single routes - without namespace",
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.GatewayRouteSpec{
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Name: "vs-1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Namespace: "my-ns",
					Name:      "vs-1",
				},
			},
		},
		{
			name: "multiple routes",
			args: args{
				obj: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						GRPCRoute: &appmesh.GRPCGatewayRoute{
							Action: appmesh.GRPCGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("my-ns"),
											Name:      "vs-2",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Namespace: "my-ns",
					Name:      "vs-1",
				},
				{
					Namespace: "my-ns",
					Name:      "vs-2",
				},
			},
		},

		{
			name: "zero routes",
			args: args{
				obj: &appmesh.GatewayRoute{
					Spec: appmesh.GatewayRouteSpec{
						GRPCRoute:  nil,
						HTTPRoute:  nil,
						HTTP2Route: nil,
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VirtualServiceReferenceIndexFunc(tt.args.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}
