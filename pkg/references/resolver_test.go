package references

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResolver_ResolveMeshReference(t *testing.T) {
	meshUIDMatches := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
		},
	}
	meshUIDMismatches := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "f7d10a22-e8d5-4626-b780-261374fc68d4",
		},
	}

	type env struct {
		meshes []*appmesh.Mesh
	}
	type args struct {
		meshRef appmesh.MeshReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "mesh can be resolved when name and UID matches",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: meshUIDMatches,
		},
		{
			name: "mesh cannot be resolved when UID mismatches",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMismatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("mesh UID mismatch: my-mesh"),
		},
		{
			name: "mesh cannot be resolved if not found",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "another-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("unable to fetch mesh: another-mesh: meshs.appmesh.k8s.aws \"another-mesh\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultResolver(k8sClient, logr.New(&log.NullLogSink{}))

			for _, ms := range tt.env.meshes {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveMeshReference(ctx, tt.args.meshRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func Test_defaultResolver_ResolveVirtualGatewayReference(t *testing.T) {
	vgInNS1 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "my-vg",
			UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
		},
	}
	vgInNS2 := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "my-vg",
			UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
		},
	}
	vgUIDMismatches := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "my-vg",
			UID:       "f7d10a22-e8d5-4626-b780-261374fc68d4",
		},
	}

	type env struct {
		virtualGateways []*appmesh.VirtualGateway
	}
	type args struct {
		obj   metav1.Object
		vgRef appmesh.VirtualGatewayReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualGateway
		wantErr error
	}{
		{
			name: "when VirtualGatewayReference contains namespace, UID and name",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vgInNS1, vgInNS2},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Namespace: aws.String("ns-2"),
					Name:      "my-vg",
					UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: vgInNS2,
		},
		{
			name: "when VirtualGatewayReference contains name and UID only",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vgInNS1, vgInNS2},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Name: "my-vg",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: vgInNS1,
		},
		{
			name: "virtual gateway cannot be resolved when UID mismatches",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vgUIDMismatches},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Namespace: aws.String("ns-1"),
					Name:      "my-vg",
					UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("virtualGateway UID mismatch: my-vg"),
		},
		{
			name: "virtual gateway cannot be resolved if not found",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{vgInNS1},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "gr",
					},
				},
				vgRef: appmesh.VirtualGatewayReference{
					Namespace: aws.String("ns-1"),
					Name:      "another-vg",
					UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("unable to fetch virtualGateway: ns-1/another-vg: virtualgatewaies.appmesh.k8s.aws \"another-vg\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultResolver(k8sClient, logr.New(&log.NullLogSink{}))

			for _, ms := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualGatewayReference(ctx, tt.args.obj, tt.args.vgRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func Test_defaultResolver_ResolveVirtualNodeReference(t *testing.T) {
	vnInNS1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vn",
		},
	}
	vnInNS2 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vn",
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		obj   metav1.Object
		vnRef appmesh.VirtualNodeReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "when VirtualNodeReference contains both namespace and name",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vr",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Namespace: aws.String("ns-2"),
					Name:      "vn",
				},
			},
			want: vnInNS2,
		},
		{
			name: "when VirtualNodeReference contains name only",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vr",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Name: "vn",
				},
			},
			want: vnInNS1,
		},
		{
			name: "when VirtualNodeReference didn't reference existing vs",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Namespace: aws.String("ns-3"),
					Name:      "vn",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualNode: ns-3/vn: virtualnodes.appmesh.k8s.aws \"vn\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultResolver(k8sClient, logr.New(&log.NullLogSink{}))

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualNodeReference(ctx, tt.args.obj, tt.args.vnRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func Test_defaultResolver_ResolveVirtualServiceReference(t *testing.T) {
	vsInNS1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vs",
		},
	}
	vsInNS2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vs",
		},
	}

	type env struct {
		virtualServices []*appmesh.VirtualService
	}
	type args struct {
		obj   metav1.Object
		vsRef appmesh.VirtualServiceReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualService
		wantErr error
	}{
		{
			name: "when VirtualServiceReference contains both namespace and name",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Namespace: aws.String("ns-2"),
					Name:      "vs",
				},
			},
			want: vsInNS2,
		},
		{
			name: "when VirtualServiceReference contains name only",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Name: "vs",
				},
			},
			want: vsInNS1,
		},
		{
			name: "when VirtualServiceReference didn't reference existing vs",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Namespace: aws.String("ns-3"),
					Name:      "vs",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualService: ns-3/vs: virtualservices.appmesh.k8s.aws \"vs\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultResolver(k8sClient, logr.New(&log.NullLogSink{}))

			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualServiceReference(ctx, tt.args.obj, tt.args.vsRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func Test_defaultResolver_ResolveVirtualRouterReference(t *testing.T) {
	vrInNS1 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vr",
		},
	}
	vrInNS2 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vr",
		},
	}

	type env struct {
		virtualRouters []*appmesh.VirtualRouter
	}
	type args struct {
		obj   metav1.Object
		vrRef appmesh.VirtualRouterReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "when VirtualRouterReference contains both namespace and name",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Namespace: aws.String("ns-2"),
					Name:      "vr",
				},
			},
			want: vrInNS2,
		},
		{
			name: "when VirtualRouterReference contains name only",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Name: "vr",
				},
			},
			want: vrInNS1,
		},
		{
			name: "when VirtualRouterReference didn't reference existing vs",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Namespace: aws.String("ns-3"),
					Name:      "vr",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualRouter: ns-3/vr: virtualrouters.appmesh.k8s.aws \"vr\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultResolver(k8sClient, logr.New(&log.NullLogSink{}))

			for _, vr := range tt.env.virtualRouters {
				err := k8sClient.Create(ctx, vr.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualRouterReference(ctx, tt.args.obj, tt.args.vrRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}
