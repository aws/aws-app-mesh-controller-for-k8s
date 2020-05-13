package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test_matchRoutesAgainstSDKRouteRefs(t *testing.T) {
	type args struct {
		routes       []appmesh.Route
		sdkRouteRefs []*appmeshsdk.RouteRef
	}
	tests := []struct {
		name                           string
		args                           args
		wantMatchedRouteAndSDKRouteRef []routeAndSDKRouteRef
		wantUnmatchedRoutes            []appmesh.Route
		wantUnmatchedSDKRouteRefs      []*appmeshsdk.RouteRef
	}{
		{
			name: "all route matches",
			args: args{
				routes: []appmesh.Route{
					{
						Name: "route-1",
					},
					{
						Name: "route-2",
					},
				},
				sdkRouteRefs: []*appmeshsdk.RouteRef{
					{
						RouteName: aws.String("route-1"),
					},
					{
						RouteName: aws.String("route-2"),
					},
				},
			},
			wantMatchedRouteAndSDKRouteRef: []routeAndSDKRouteRef{
				{
					route: appmesh.Route{
						Name: "route-1",
					},
					sdkRouteRef: &appmeshsdk.RouteRef{
						RouteName: aws.String("route-1"),
					},
				},
				{
					route: appmesh.Route{
						Name: "route-2",
					},
					sdkRouteRef: &appmeshsdk.RouteRef{
						RouteName: aws.String("route-2"),
					},
				},
			},
			wantUnmatchedRoutes:       []appmesh.Route{},
			wantUnmatchedSDKRouteRefs: []*appmeshsdk.RouteRef{},
		},
		{
			name: "all route un-matches",
			args: args{
				routes: []appmesh.Route{
					{
						Name: "route-1",
					},
					{
						Name: "route-2",
					},
				},
				sdkRouteRefs: []*appmeshsdk.RouteRef{},
			},
			wantMatchedRouteAndSDKRouteRef: []routeAndSDKRouteRef{},
			wantUnmatchedRoutes: []appmesh.Route{
				{
					Name: "route-1",
				},
				{
					Name: "route-2",
				},
			},
			wantUnmatchedSDKRouteRefs: []*appmeshsdk.RouteRef{},
		},
		{
			name: "all sdkRouteRef un-matches",
			args: args{
				routes: []appmesh.Route{},
				sdkRouteRefs: []*appmeshsdk.RouteRef{
					{
						RouteName: aws.String("route-1"),
					},
					{
						RouteName: aws.String("route-2"),
					},
				},
			},
			wantMatchedRouteAndSDKRouteRef: []routeAndSDKRouteRef{},
			wantUnmatchedRoutes:            []appmesh.Route{},
			wantUnmatchedSDKRouteRefs: []*appmeshsdk.RouteRef{
				{
					RouteName: aws.String("route-1"),
				},
				{
					RouteName: aws.String("route-2"),
				},
			},
		},
		{
			name: "some route un-match and some sdkRouteRef un-match",
			args: args{
				routes: []appmesh.Route{
					{
						Name: "route-1",
					},
					{
						Name: "route-2",
					},
					{
						Name: "route-3",
					},
				},
				sdkRouteRefs: []*appmeshsdk.RouteRef{
					{
						RouteName: aws.String("route-2"),
					},
					{
						RouteName: aws.String("route-3"),
					},
					{
						RouteName: aws.String("route-4"),
					},
				},
			},
			wantMatchedRouteAndSDKRouteRef: []routeAndSDKRouteRef{
				{
					route: appmesh.Route{
						Name: "route-2",
					},
					sdkRouteRef: &appmeshsdk.RouteRef{
						RouteName: aws.String("route-2"),
					},
				},
				{
					route: appmesh.Route{
						Name: "route-3",
					},
					sdkRouteRef: &appmeshsdk.RouteRef{
						RouteName: aws.String("route-3"),
					},
				},
			},
			wantUnmatchedRoutes: []appmesh.Route{
				{
					Name: "route-1",
				},
			},
			wantUnmatchedSDKRouteRefs: []*appmeshsdk.RouteRef{
				{
					RouteName: aws.String("route-4"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatchedRouteAndSDKRouteRef, gotUnmatchedRoutes, gotUnmatchedSDKRouteRefs := matchRoutesAgainstSDKRouteRefs(tt.args.routes, tt.args.sdkRouteRefs)
			assert.Equal(t, tt.wantMatchedRouteAndSDKRouteRef, gotMatchedRouteAndSDKRouteRef)
			assert.Equal(t, tt.wantUnmatchedRoutes, gotUnmatchedRoutes)
			assert.Equal(t, tt.wantUnmatchedSDKRouteRefs, gotUnmatchedSDKRouteRefs)
		})
	}
}

func Test_BuildSDKRouteSpec(t *testing.T) {
	type args struct {
		vr      *appmesh.VirtualRouter
		route   appmesh.Route
		vnByKey map[types.NamespacedName]*appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		want    *appmeshsdk.RouteSpec
		wantErr error
	}{
		{
			name: "GRPC route",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-ns",
					},
				},
				route: appmesh.Route{
					GRPCRoute: &appmesh.GRPCRoute{
						Match: appmesh.GRPCRouteMatch{
							Metadata: []appmesh.GRPCRouteMetadata{
								{
									Name: "User-Agent: X",
									Match: &appmesh.GRPCRouteMetadataMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.GRPCRouteMetadataMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							MethodName:  aws.String("stream"),
							ServiceName: aws.String("foo.foodomain.local"),
						},
						Action: appmesh.GRPCRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.GRPCRetryPolicy{
							GRPCRetryEvents: []appmesh.GRPCRetryPolicyEvent{"cancelled", "deadline-exceeded"},
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "ns-1", Name: "vn-1"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_ns-1"),
						},
					},
					types.NamespacedName{Namespace: "ns-2", Name: "vn-2"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-2_ns-2"),
						},
					},
				},
			},
			want: &appmeshsdk.RouteSpec{
				GrpcRoute: &appmeshsdk.GrpcRoute{
					Match: &appmeshsdk.GrpcRouteMatch{
						Metadata: []*appmeshsdk.GrpcRouteMetadata{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						MethodName:  aws.String("stream"),
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: &appmeshsdk.GrpcRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1_ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2_ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.GrpcRetryPolicy{
						GrpcRetryEvents: []*string{aws.String("cancelled"), aws.String("deadline-exceeded")},
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
			},
		},
		{
			name: "HTTP route",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-ns",
					},
				},
				route: appmesh.Route{
					HTTPRoute: &appmesh.HTTPRoute{
						Match: appmesh.HTTPRouteMatch{
							Headers: []appmesh.HTTPRouteHeader{
								{
									Name: "User-Agent: X",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							Method: aws.String("GET"),
							Prefix: "/appmesh",
							Scheme: aws.String("https"),
						},
						Action: appmesh.HTTPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.HTTPRetryPolicy{
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "ns-1", Name: "vn-1"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_ns-1"),
						},
					},
					types.NamespacedName{Namespace: "ns-2", Name: "vn-2"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-2_ns-2"),
						},
					},
				},
			},
			want: &appmeshsdk.RouteSpec{
				HttpRoute: &appmeshsdk.HttpRoute{
					Match: &appmeshsdk.HttpRouteMatch{
						Headers: []*appmeshsdk.HttpRouteHeader{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						Method: aws.String("GET"),
						Prefix: aws.String("/appmesh"),
						Scheme: aws.String("https"),
					},
					Action: &appmeshsdk.HttpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1_ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2_ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.HttpRetryPolicy{
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
			},
		},
		{
			name: "HTTP2 route",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-ns",
					},
				},
				route: appmesh.Route{
					HTTP2Route: &appmesh.HTTPRoute{
						Match: appmesh.HTTPRouteMatch{
							Headers: []appmesh.HTTPRouteHeader{
								{
									Name: "User-Agent: X",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							Method: aws.String("GET"),
							Prefix: "/appmesh",
							Scheme: aws.String("https"),
						},
						Action: appmesh.HTTPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.HTTPRetryPolicy{
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "ns-1", Name: "vn-1"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_ns-1"),
						},
					},
					types.NamespacedName{Namespace: "ns-2", Name: "vn-2"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-2_ns-2"),
						},
					},
				},
			},
			want: &appmeshsdk.RouteSpec{
				Http2Route: &appmeshsdk.HttpRoute{
					Match: &appmeshsdk.HttpRouteMatch{
						Headers: []*appmeshsdk.HttpRouteHeader{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						Method: aws.String("GET"),
						Prefix: aws.String("/appmesh"),
						Scheme: aws.String("https"),
					},
					Action: &appmeshsdk.HttpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1_ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2_ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.HttpRetryPolicy{
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
			},
		},
		{
			name: "TCP route",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-ns",
					},
				},
				route: appmesh.Route{
					TCPRoute: &appmesh.TCPRoute{
						Action: appmesh.TCPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "ns-1", Name: "vn-1"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_ns-1"),
						},
					},
					types.NamespacedName{Namespace: "ns-2", Name: "vn-2"}: {
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-2_ns-2"),
						},
					},
				},
			},
			want: &appmeshsdk.RouteSpec{
				TcpRoute: &appmeshsdk.TcpRoute{
					Action: &appmeshsdk.TcpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1_ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2_ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSDKRouteSpec(tt.args.vr, tt.args.route, tt.args.vnByKey)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
