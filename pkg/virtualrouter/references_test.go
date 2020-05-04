package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_extractVirtualNodeReferences(t *testing.T) {
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
			got := extractVirtualNodeReferences(tt.args.vr)
			assert.Equal(t, tt.want, got)
		})
	}
}
