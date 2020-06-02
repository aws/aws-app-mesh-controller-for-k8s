package references

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func TestObjectKeyForVirtualGatewayReference(t *testing.T) {
	type args struct {
		obj   metav1.Object
		vgRef appmesh.VirtualGatewayReference
	}
	tests := []struct {
		name string
		args args
		want types.NamespacedName
	}{
		{
			name: "namespace un-specified",
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gr-ns",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Name: "vg",
				},
			},
			want: types.NamespacedName{Namespace: "gr-ns", Name: "vg"},
		},
		{
			name: "namespace specified",
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "gr-ns",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Namespace: aws.String("vg-ns"),
					Name:      "vg",
				},
			},
			want: types.NamespacedName{Namespace: "vg-ns", Name: "vg"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjectKeyForVirtualGatewayReference(tt.args.obj, tt.args.vgRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestObjectKeyForVirtualNodeReference(t *testing.T) {
	type args struct {
		obj   metav1.Object
		vnRef appmesh.VirtualNodeReference
	}
	tests := []struct {
		name string
		args args
		want types.NamespacedName
	}{
		{
			name: "namespace un-specified",
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vs-ns",
						Name:      "vs",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Name: "vn",
				},
			},
			want: types.NamespacedName{Namespace: "vs-ns", Name: "vn"},
		},
		{
			name: "namespace specified",
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vs-ns",
						Name:      "vs",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Namespace: aws.String("vn-ns"),
					Name:      "vn",
				},
			},
			want: types.NamespacedName{Namespace: "vn-ns", Name: "vn"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjectKeyForVirtualNodeReference(tt.args.obj, tt.args.vnRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestObjectKeyForVirtualServiceReference(t *testing.T) {
	type args struct {
		obj   metav1.Object
		vsRef appmesh.VirtualServiceReference
	}
	tests := []struct {
		name string
		args args
		want types.NamespacedName
	}{
		{
			name: "namespace un-specified",
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vn-ns",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Name: "vs",
				},
			},
			want: types.NamespacedName{Namespace: "vn-ns", Name: "vs"},
		},
		{
			name: "namespace specified",
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vn-ns",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Namespace: aws.String("vs-ns"),
					Name:      "vs",
				},
			},
			want: types.NamespacedName{Namespace: "vs-ns", Name: "vs"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjectKeyForVirtualServiceReference(tt.args.obj, tt.args.vsRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestObjectKeyForVirtualRouterReference(t *testing.T) {
	type args struct {
		obj   metav1.Object
		vrRef appmesh.VirtualRouterReference
	}
	tests := []struct {
		name string
		args args
		want types.NamespacedName
	}{
		{
			name: "namespace un-specified",
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vs-ns",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Name: "vr",
				},
			},
			want: types.NamespacedName{Namespace: "vs-ns", Name: "vr"},
		},
		{
			name: "namespace specified",
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "vs-ns",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Namespace: aws.String("vr-ns"),
					Name:      "vr",
				},
			},
			want: types.NamespacedName{Namespace: "vr-ns", Name: "vr"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjectKeyForVirtualRouterReference(tt.args.obj, tt.args.vrRef)
			assert.Equal(t, tt.want, got)
		})
	}
}
