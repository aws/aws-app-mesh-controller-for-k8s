package mesh

import (
	"context"
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_pendingMembersFinalizer_buildPendingMembersEventMessage(t *testing.T) {
	type args struct {
		vsMembers []*appmesh.VirtualService
		vrMembers []*appmesh.VirtualRouter
		vnMembers []*appmesh.VirtualNode
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "two virtualService pending",
			args: args{
				vsMembers: []*appmesh.VirtualService{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vs-1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "vs-2",
						},
					},
				},
			},
			want: "objects belong to this mesh exists, please delete them to proceed. virtualService: 2",
		},
		{
			name: "two virtualRouter pending",
			args: args{
				vrMembers: []*appmesh.VirtualRouter{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vr-1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "vr-2",
						},
					},
				},
			},
			want: "objects belong to this mesh exists, please delete them to proceed. virtualRouter: 2",
		},
		{
			name: "two virtualNode pending",
			args: args{
				vnMembers: []*appmesh.VirtualNode{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vn-1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "vn-2",
						},
					},
				},
			},
			want: "objects belong to this mesh exists, please delete them to proceed. virtualNode: 2",
		},
		{
			name: "1 virtualService and 1 virtualNode pending",
			args: args{
				vsMembers: []*appmesh.VirtualService{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vs-1",
						},
					},
				},
				vnMembers: []*appmesh.VirtualNode{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vn-1",
						},
					},
				},
			},
			want: "objects belong to this mesh exists, please delete them to proceed. virtualService: 1, virtualNode: 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:     k8sClient,
				eventRecorder: eventRecorder,
				log:           &log.NullLogger{},
			}
			got := m.buildPendingMembersEventMessage(ctx, tt.args.vsMembers, tt.args.vrMembers, tt.args.vnMembers)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_pendingMembersFinalizer_findVirtualServiceMembers(t *testing.T) {
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-1",
			UID:  "uid-1",
		},
	}
	vsInMesh_1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vs-1",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vsInMesh_2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vs-2",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vsNotInMesh_1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-3",
			Name:      "vs-3",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-2",
				UID:  "uid-2",
			},
		},
	}
	vsNotInMesh_2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-4",
			Name:      "vs-4",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-2",
			},
		},
	}

	type env struct {
		virtualServices []*appmesh.VirtualService
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    []*appmesh.VirtualService
		wantErr error
	}{
		{
			name: "found no virtualService",
			env: env{
				virtualServices: []*appmesh.VirtualService{},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualService{},
		},
		{
			name: "found virtualServices that matches",
			env: env{
				virtualServices: []*appmesh.VirtualService{
					vsInMesh_1, vsInMesh_2, vsNotInMesh_1, vsNotInMesh_2,
				},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualService{vsInMesh_1, vsInMesh_2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              &log.NullLogger{},
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}

			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.findVirtualServiceMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.SortSlices(compareVirtualService),
				}
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func Test_pendingMembersFinalizer_findVirtualNodeMembers(t *testing.T) {
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-1",
			UID:  "uid-1",
		},
	}
	vnInMesh_1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vnInMesh_2 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vn-2",
		},
		Spec: appmesh.VirtualNodeSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vnNotInMesh_1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-3",
			Name:      "vn-3",
		},
		Spec: appmesh.VirtualNodeSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-2",
				UID:  "uid-2",
			},
		},
	}
	vnNotInMesh_2 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-4",
			Name:      "vn-4",
		},
		Spec: appmesh.VirtualNodeSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-2",
			},
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    []*appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "found no virtualNode",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualNode{},
		},
		{
			name: "found virtualNodes that matches",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{
					vnInMesh_1, vnInMesh_2, vnNotInMesh_1, vnNotInMesh_2,
				},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualNode{vnInMesh_1, vnInMesh_2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              &log.NullLogger{},
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.findVirtualNodeMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.SortSlices(compareVirtualNode),
				}
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func Test_pendingMembersFinalizer_findVirtualRouterMembers(t *testing.T) {
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-1",
			UID:  "uid-1",
		},
	}
	vrInMesh_1 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vr-1",
		},
		Spec: appmesh.VirtualRouterSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vrInMesh_2 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vr-2",
		},
		Spec: appmesh.VirtualRouterSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-1",
			},
		},
	}
	vrNotInMesh_1 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-3",
			Name:      "vr-3",
		},
		Spec: appmesh.VirtualRouterSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-2",
				UID:  "uid-2",
			},
		},
	}
	vrNotInMesh_2 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-4",
			Name:      "vr-4",
		},
		Spec: appmesh.VirtualRouterSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "mesh-1",
				UID:  "uid-2",
			},
		},
	}

	type env struct {
		virtualRouters []*appmesh.VirtualRouter
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    []*appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "found no virtualRouter",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualRouter{},
		},
		{
			name: "found virtualRouters that matches",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{
					vrInMesh_1, vrInMesh_2, vrNotInMesh_1, vrNotInMesh_2,
				},
			},
			args: args{
				ms: ms,
			},
			want: []*appmesh.VirtualRouter{vrInMesh_1, vrInMesh_2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              &log.NullLogger{},
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}

			for _, vr := range tt.env.virtualRouters {
				err := k8sClient.Create(ctx, vr.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.findVirtualRouterMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.SortSlices(compareVirtualRouter),
				}
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func Test_pendingMembersFinalizer_Finalize(t *testing.T) {
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "uid-1",
		},
	}
	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vs-1",
		},
		Spec: appmesh.VirtualServiceSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "uid-1",
			},
		},
	}
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "uid-1",
			},
		},
	}
	vr := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "vr-1",
		},
		Spec: appmesh.VirtualRouterSpec{
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "uid-1",
			},
		},
	}
	_ = vn
	_ = vr

	type env struct {
		virtualServices []*appmesh.VirtualService
		virtualNodes    []*appmesh.VirtualNode
		virtualRouters  []*appmesh.VirtualRouter
	}
	type args struct {
		ms *appmesh.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "when pending virtualService deletion",
			env: env{
				virtualServices: []*appmesh.VirtualService{vs},
			},
			args:    args{ms: ms},
			wantErr: errors.New("pending members deletion"),
		},
		{
			name: "when pending virtualRouter deletion",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vr},
			},
			args:    args{ms: ms},
			wantErr: errors.New("pending members deletion"),
		},
		{
			name: "when pending virtualService deletion",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vn},
			},
			args:    args{ms: ms},
			wantErr: errors.New("pending members deletion"),
		},
		{
			name:    "when pending no member deletion",
			env:     env{},
			args:    args{ms: ms},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              &log.NullLogger{},
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}
			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs)
				assert.NoError(t, err)
			}
			for _, vr := range tt.env.virtualRouters {
				err := k8sClient.Create(ctx, vr)
				assert.NoError(t, err)
			}
			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn)
				assert.NoError(t, err)
			}

			err := m.Finalize(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func compareVirtualService(a *appmesh.VirtualService, b *appmesh.VirtualService) bool {
	return k8s.NamespacedName(a).String() < k8s.NamespacedName(b).String()
}

func compareVirtualNode(a *appmesh.VirtualNode, b *appmesh.VirtualNode) bool {
	return k8s.NamespacedName(a).String() < k8s.NamespacedName(b).String()
}

func compareVirtualRouter(a *appmesh.VirtualRouter, b *appmesh.VirtualRouter) bool {
	return k8s.NamespacedName(a).String() < k8s.NamespacedName(b).String()
}
