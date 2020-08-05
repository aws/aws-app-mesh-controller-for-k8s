package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualRouterValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		vr    *appmesh.VirtualRouter
		oldVR *appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualRouter immutable fields didn't change",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVR: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "VirtualRouter field awsName changed",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVR: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualRouter update may not change these fields: spec.awsName"),
		},
		{
			name: "VirtualRouter field meshRef changed",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVR: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualRouter update may not change these fields: spec.meshRef"),
		},
		{
			name: "VirtualRouter fields awsName and meshRef changed",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVR: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualRouter update may not change these fields: spec.awsName,spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualRouterValidator{}
			err := v.enforceFieldsImmutability(tt.args.vr, tt.args.oldVR)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_virtualRouterValidator_checkForDuplicateRouteEntries(t *testing.T) {
	testRESTMethod := "GET"
	type args struct {
		vr    *appmesh.VirtualRouter
		oldVR *appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualRouter has duplicate routes",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Routes: []appmesh.Route{
							{
								Name: "testRoute",
								HTTPRoute: &appmesh.HTTPRoute{
									Match: appmesh.HTTPRouteMatch{
										Method: &testRESTMethod,
										Prefix: "/",
									},
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: &appmesh.VirtualNodeReference{
													Name: "testVN",
												},
												Weight: 1,
											},
										},
									},
									RetryPolicy: nil,
									Timeout:     nil,
								},
							},
							{
								Name: "testRoute",
								HTTPRoute: &appmesh.HTTPRoute{
									Match: appmesh.HTTPRouteMatch{
										Method: &testRESTMethod,
										Prefix: "/test",
									},
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: &appmesh.VirtualNodeReference{
													Name: "testVN",
												},
												Weight: 1,
											},
										},
									},
									RetryPolicy: nil,
									Timeout:     nil,
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualRouter-my-vr has duplicate route entries for testRoute"),
		},
		{
			name: "No duplicate routes",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Routes: []appmesh.Route{
							{
								Name: "testRoute",
								HTTPRoute: &appmesh.HTTPRoute{
									Match: appmesh.HTTPRouteMatch{
										Method: &testRESTMethod,
										Prefix: "/",
									},
									Action: appmesh.HTTPRouteAction{
										WeightedTargets: []appmesh.WeightedTarget{
											{
												VirtualNodeRef: &appmesh.VirtualNodeReference{
													Name: "testVN",
												},
												Weight: 1,
											},
										},
									},
									RetryPolicy: nil,
									Timeout:     nil,
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualRouterValidator{}
			err := v.checkForDuplicateRouteEntries(tt.args.vr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
