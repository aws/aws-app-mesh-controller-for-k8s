package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test_ExtractVirtualNodeReferences(t *testing.T) {
	type args struct {
		vr *appmesh.VirtualRouter
	}
	tests := []struct {
		name string
		args args
		want []appmesh.VirtualNodeReference
	}{
		{
			name: "single GRPC route",
			args: args{
				vr: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								GRPCRoute: &appmesh.GRPCRoute{
									Action: appmesh.GRPCRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualNodeReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-2",
				},
			},
		},
		{
			name: "single HTTP route",
			args: args{
				vr: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								HTTPRoute: &appmesh.HTTPRoute{
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualNodeReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-2",
				},
			},
		},
		{
			name: "single HTTP2 route",
			args: args{
				vr: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								HTTP2Route: &appmesh.HTTPRoute{
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualNodeReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-2",
				},
			},
		},
		{
			name: "single TCP route",
			args: args{
				vr: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								TCPRoute: &appmesh.TCPRoute{
									Action: appmesh.TCPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualNodeReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-2",
				},
			},
		},
		{
			name: "multiple routes",
			args: args{
				vr: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								GRPCRoute: &appmesh.GRPCRoute{
									Action: appmesh.GRPCRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
										},
									},
								},
								HTTPRoute: &appmesh.HTTPRoute{
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
										},
									},
								},
								HTTP2Route: &appmesh.HTTPRoute{
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-3",
												},
											},
										},
									},
								},
								TCPRoute: &appmesh.TCPRoute{
									Action: appmesh.TCPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-4",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []appmesh.VirtualNodeReference{
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-1",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-2",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-3",
				},
				{
					Namespace: aws.String("my-ns"),
					Name:      "vn-4",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVirtualNodeReferences(tt.args.vr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVirtualNodeReferenceIndexFunc(t *testing.T) {
	type args struct {
		obj runtime.Object
	}
	tests := []struct {
		name string
		args args
		want []types.NamespacedName
	}{
		{
			name: "single routes - with namespace",
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								GRPCRoute: &appmesh.GRPCRoute{
									Action: appmesh.GRPCRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("other-ns"),
													Name:      "vn-1",
												},
											},
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
					Name:      "vn-1",
				},
			},
		},
		{
			name: "single routes - without namespace",
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								GRPCRoute: &appmesh.GRPCRoute{
									Action: appmesh.GRPCRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Name: "vn-1",
												},
											},
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
					Name:      "vn-1",
				},
			},
		},
		{
			name: "multiple routes",
			args: args{
				obj: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: []appmesh.Route{
							{
								GRPCRoute: &appmesh.GRPCRoute{
									Action: appmesh.GRPCRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-1",
												},
											},
										},
									},
								},
								HTTPRoute: &appmesh.HTTPRoute{
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: appmesh.VirtualNodeReference{
													Namespace: aws.String("my-ns"),
													Name:      "vn-2",
												},
											},
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
					Name:      "vn-1",
				},
				{
					Namespace: "my-ns",
					Name:      "vn-2",
				},
			},
		},

		{
			name: "zero routes",
			args: args{
				obj: &appmesh.VirtualRouter{
					Spec: appmesh.VirtualRouterSpec{
						Routes: nil,
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VirtualNodeReferenceIndexFunc(tt.args.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}
