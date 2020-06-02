package references

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func TestBuildSDKVirtualGatewayReferenceConvertFunc(t *testing.T) {
	gr := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr",
		},
	}
	vgRefWithNamespace := appmesh.VirtualGatewayReference{
		Namespace: aws.String("my-ns"),
		Name:      "vg-1",
	}
	vgRefWithoutNamespace := appmesh.VirtualGatewayReference{
		Namespace: nil,
		Name:      "vg-2",
	}
	vgRefInAnotherNamespace := appmesh.VirtualGatewayReference{
		Namespace: aws.String("my-other-ns"),
		Name:      "vg-3",
	}

	type args struct {
		obj     metav1.Object
		vgByKey map[types.NamespacedName]*appmesh.VirtualGateway
	}
	tests := []struct {
		name                  string
		args                  args
		wantAWSNameOrErrByRef map[appmesh.VirtualGatewayReference]struct {
			awsName string
			err     error
		}
	}{
		{
			name: "when all VirtualGatewayReference resolve correctly",
			args: args{
				obj: gr,
				vgByKey: map[types.NamespacedName]*appmesh.VirtualGateway{
					types.NamespacedName{Namespace: "my-ns", Name: "vg-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vg-1",
						},
						Spec: appmesh.VirtualGatewaySpec{
							AWSName: aws.String("vg-1_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-ns", Name: "vg-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vg-2",
						},
						Spec: appmesh.VirtualGatewaySpec{
							AWSName: aws.String("vg-2_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-other-ns", Name: "vg-3"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-other-ns",
							Name:      "vg-3",
						},
						Spec: appmesh.VirtualGatewaySpec{
							AWSName: aws.String("vg-3_my-other-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualGatewayReference]struct {
				awsName string
				err     error
			}{
				vgRefWithNamespace: {
					awsName: "vg-1_my-ns",
				},
				vgRefWithoutNamespace: {
					awsName: "vg-2_my-ns",
				},
				vgRefInAnotherNamespace: {
					awsName: "vg-3_my-other-ns",
				},
			},
		},
		{
			name: "when some VirtualGatewayReference cannot resolve correctly",
			args: args{
				obj: gr,
				vgByKey: map[types.NamespacedName]*appmesh.VirtualGateway{
					types.NamespacedName{Namespace: "my-ns", Name: "vg-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vg-1",
						},
						Spec: appmesh.VirtualGatewaySpec{
							AWSName: aws.String("vg-1_my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualGatewayReference]struct {
				awsName string
				err     error
			}{
				vgRefWithNamespace: {
					awsName: "vg-1_my-ns",
				},
				vgRefWithoutNamespace: {
					err: errors.New("unexpected VirtualGatewayReference: my-ns/vg-2"),
				},
				vgRefInAnotherNamespace: {
					err: errors.New("unexpected VirtualGatewayReference: my-other-ns/vg-3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertFunc := BuildSDKVirtualGatewayReferenceConvertFunc(tt.args.obj, tt.args.vgByKey)
			for vgRef, wantAWSNameOrErr := range tt.wantAWSNameOrErrByRef {
				var gotAWSName = ""
				gotErr := convertFunc(&vgRef, &gotAWSName, nil)
				if wantAWSNameOrErr.err != nil {
					assert.EqualError(t, gotErr, wantAWSNameOrErr.err.Error())
				} else {
					assert.NoError(t, gotErr)
					assert.Equal(t, wantAWSNameOrErr.awsName, gotAWSName)
				}
			}
		})
	}
}

func TestBuildSDKVirtualNodeReferenceConvertFunc(t *testing.T) {
	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vs",
		},
	}
	vnRefWithNamespace := appmesh.VirtualNodeReference{
		Namespace: aws.String("my-ns"),
		Name:      "vn-1",
	}
	vnRefWithoutNamespace := appmesh.VirtualNodeReference{
		Namespace: nil,
		Name:      "vn-2",
	}
	vnRefInAnotherNamespace := appmesh.VirtualNodeReference{
		Namespace: aws.String("my-other-ns"),
		Name:      "vn-3",
	}

	type args struct {
		obj     metav1.Object
		vnByKey map[types.NamespacedName]*appmesh.VirtualNode
	}
	tests := []struct {
		name                  string
		args                  args
		wantAWSNameOrErrByRef map[appmesh.VirtualNodeReference]struct {
			awsName string
			err     error
		}
	}{
		{
			name: "when all VirtualNodeReference resolve correctly",
			args: args{
				obj: vs,
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "my-ns", Name: "vn-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vn-1",
						},
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-ns", Name: "vn-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vn-2",
						},
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-2_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-other-ns", Name: "vn-3"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-other-ns",
							Name:      "vn-3",
						},
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-3_my-other-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualNodeReference]struct {
				awsName string
				err     error
			}{
				vnRefWithNamespace: {
					awsName: "vn-1_my-ns",
				},
				vnRefWithoutNamespace: {
					awsName: "vn-2_my-ns",
				},
				vnRefInAnotherNamespace: {
					awsName: "vn-3_my-other-ns",
				},
			},
		},
		{
			name: "when some VirtualNodeReference cannot resolve correctly",
			args: args{
				obj: vs,
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "my-ns", Name: "vn-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vn-1",
						},
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn-1_my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualNodeReference]struct {
				awsName string
				err     error
			}{
				vnRefWithNamespace: {
					awsName: "vn-1_my-ns",
				},
				vnRefWithoutNamespace: {
					err: errors.New("unexpected VirtualNodeReference: my-ns/vn-2"),
				},
				vnRefInAnotherNamespace: {
					err: errors.New("unexpected VirtualNodeReference: my-other-ns/vn-3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertFunc := BuildSDKVirtualNodeReferenceConvertFunc(tt.args.obj, tt.args.vnByKey)
			for vnRef, wantAWSNameOrErr := range tt.wantAWSNameOrErrByRef {
				var gotAWSName = ""
				gotErr := convertFunc(&vnRef, &gotAWSName, nil)
				if wantAWSNameOrErr.err != nil {
					assert.EqualError(t, gotErr, wantAWSNameOrErr.err.Error())
				} else {
					assert.NoError(t, gotErr)
					assert.Equal(t, wantAWSNameOrErr.awsName, gotAWSName)
				}
			}
		})
	}
}

func TestBuildSDKVirtualServiceReferenceConvertFunc(t *testing.T) {
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vn",
		},
	}
	vsRefWithNamespace := appmesh.VirtualServiceReference{
		Namespace: aws.String("my-ns"),
		Name:      "vs-1",
	}
	vsRefWithoutNamespace := appmesh.VirtualServiceReference{
		Namespace: nil,
		Name:      "vs-2",
	}
	vsRefInAnotherNamespace := appmesh.VirtualServiceReference{
		Namespace: aws.String("my-other-ns"),
		Name:      "vs-3",
	}

	type args struct {
		obj     metav1.Object
		vsByKey map[types.NamespacedName]*appmesh.VirtualService
	}
	tests := []struct {
		name                  string
		args                  args
		wantAWSNameOrErrByRef map[appmesh.VirtualServiceReference]struct {
			awsName string
			err     error
		}
	}{
		{
			name: "when all VirtualServiceReference resolve correctly",
			args: args{
				obj: vn,
				vsByKey: map[types.NamespacedName]*appmesh.VirtualService{
					types.NamespacedName{Namespace: "my-ns", Name: "vs-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-1",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-1.my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-ns", Name: "vs-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-2",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-2.my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-other-ns", Name: "vs-3"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-other-ns",
							Name:      "vs-3",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-3.my-other-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualServiceReference]struct {
				awsName string
				err     error
			}{
				vsRefWithNamespace: {
					awsName: "vs-1.my-ns",
				},
				vsRefWithoutNamespace: {
					awsName: "vs-2.my-ns",
				},
				vsRefInAnotherNamespace: {
					awsName: "vs-3.my-other-ns",
				},
			},
		},
		{
			name: "when some VirtualServiceReference cannot resolve correctly",
			args: args{
				obj: vn,
				vsByKey: map[types.NamespacedName]*appmesh.VirtualService{
					types.NamespacedName{Namespace: "my-ns", Name: "vs-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-1",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-1.my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualServiceReference]struct {
				awsName string
				err     error
			}{
				vsRefWithNamespace: {
					awsName: "vs-1.my-ns",
				},
				vsRefWithoutNamespace: {
					err: errors.New("unexpected VirtualServiceReference: my-ns/vs-2"),
				},
				vsRefInAnotherNamespace: {
					err: errors.New("unexpected VirtualServiceReference: my-other-ns/vs-3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertFunc := BuildSDKVirtualServiceReferenceConvertFunc(tt.args.obj, tt.args.vsByKey)
			for vsRef, wantAWSNameOrErr := range tt.wantAWSNameOrErrByRef {
				var gotAWSName = ""
				gotErr := convertFunc(&vsRef, &gotAWSName, nil)
				if wantAWSNameOrErr.err != nil {
					assert.EqualError(t, gotErr, wantAWSNameOrErr.err.Error())
				} else {
					assert.NoError(t, gotErr)
					assert.Equal(t, wantAWSNameOrErr.awsName, gotAWSName)
				}
			}
		})
	}
}

func TestBuildSDKVirtualRouterReferenceConvertFunc(t *testing.T) {
	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vs",
		},
	}
	vrRefWithNamespace := appmesh.VirtualRouterReference{
		Namespace: aws.String("my-ns"),
		Name:      "vr-1",
	}
	vrRefWithoutNamespace := appmesh.VirtualRouterReference{
		Namespace: nil,
		Name:      "vr-2",
	}
	vrRefInAnotherNamespace := appmesh.VirtualRouterReference{
		Namespace: aws.String("my-other-ns"),
		Name:      "vr-3",
	}

	type args struct {
		obj     metav1.Object
		vrByKey map[types.NamespacedName]*appmesh.VirtualRouter
	}
	tests := []struct {
		name                  string
		args                  args
		wantAWSNameOrErrByRef map[appmesh.VirtualRouterReference]struct {
			awsName string
			err     error
		}
	}{
		{
			name: "when all VirtualRouterReference resolve correctly",
			args: args{
				obj: vs,
				vrByKey: map[types.NamespacedName]*appmesh.VirtualRouter{
					types.NamespacedName{Namespace: "my-ns", Name: "vr-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vr-1",
						},
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr-1_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-ns", Name: "vr-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vr-2",
						},
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr-2_my-ns"),
						},
					},
					types.NamespacedName{Namespace: "my-other-ns", Name: "vr-3"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-other-ns",
							Name:      "vr-3",
						},
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr-3_my-other-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualRouterReference]struct {
				awsName string
				err     error
			}{
				vrRefWithNamespace: {
					awsName: "vr-1_my-ns",
				},
				vrRefWithoutNamespace: {
					awsName: "vr-2_my-ns",
				},
				vrRefInAnotherNamespace: {
					awsName: "vr-3_my-other-ns",
				},
			},
		},
		{
			name: "when some VirtualRouterReference cannot resolve correctly",
			args: args{
				obj: vs,
				vrByKey: map[types.NamespacedName]*appmesh.VirtualRouter{
					types.NamespacedName{Namespace: "my-ns", Name: "vr-1"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vr-1",
						},
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr-1_my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualRouterReference]struct {
				awsName string
				err     error
			}{
				vrRefWithNamespace: {
					awsName: "vr-1_my-ns",
				},
				vrRefWithoutNamespace: {
					err: errors.New("unexpected VirtualRouterReference: my-ns/vr-2"),
				},
				vrRefInAnotherNamespace: {
					err: errors.New("unexpected VirtualRouterReference: my-other-ns/vr-3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertFunc := BuildSDKVirtualRouterReferenceConvertFunc(tt.args.obj, tt.args.vrByKey)
			for vrRef, wantAWSNameOrErr := range tt.wantAWSNameOrErrByRef {
				var gotAWSName = ""
				gotErr := convertFunc(&vrRef, &gotAWSName, nil)
				if wantAWSNameOrErr.err != nil {
					assert.EqualError(t, gotErr, wantAWSNameOrErr.err.Error())
				} else {
					assert.NoError(t, gotErr)
					assert.Equal(t, wantAWSNameOrErr.awsName, gotAWSName)
				}
			}
		})
	}
}
