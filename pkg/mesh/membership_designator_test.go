package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_meshSelectorDesignator_Designate(t *testing.T) {
	meshWithNilNSSelector := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-with-nil-ns-selector",
		},
		Spec: appmesh.MeshSpec{
			NamespaceSelector: nil,
		},
	}
	meshWithEmptyNSSelector := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-with-empty-ns-selector",
		},
		Spec: appmesh.MeshSpec{
			NamespaceSelector: &metav1.LabelSelector{},
		},
	}
	meshWithNSSelectorMeshX := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-with-ns-selector-mesh-x",
		},
		Spec: appmesh.MeshSpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"mesh": "x",
				},
			},
		},
	}
	meshWithNSSelectorMeshY := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-with-ns-selector-mesh-y",
		},
		Spec: appmesh.MeshSpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"mesh": "y",
				},
			},
		},
	}

	type env struct {
		meshes     []*appmesh.Mesh
		namespaces []*corev1.Namespace
	}
	type args struct {
		obj metav1.Object
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "[a single mesh with empty namespace selector] namespace without labels can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithEmptyNSSelector,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[a single mesh with empty namespace selector] namespace with labels can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithEmptyNSSelector,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"any-key": "any-value",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[a single mesh with nil namespace selector] namespace without labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNilNSSelector,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single mesh with nil namespace selector] namespace with labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNilNSSelector,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"any-key": "any-value",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single mesh selects namespace with specific labels] namespace with matching labels can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"mesh": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithNSSelectorMeshX,
			wantErr: nil,
		},
		{
			name: "[a single mesh selects namespace with specific labels] namespace without labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single mesh selects namespace with specific labels] namespace with non-matching labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"some-key": "some-value",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[multiple mesh - one with empty namespace selector, another selects namespace with specific labels] namespaces without labels can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithEmptyNSSelector,
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[multiple mesh - one with empty namespace selector, another selects namespace with specific labels] namespaces with non-matching labels can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithEmptyNSSelector,
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"some-key": "some-value",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[multiple mesh - one with empty namespace selector, another selects namespace with specific labels] namespaces with matching labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithEmptyNSSelector,
					meshWithNSSelectorMeshX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"mesh": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("found multiple matching meshes for namespace: awesome-ns, expecting 1 but found 2: mesh-with-empty-ns-selector,mesh-with-ns-selector-mesh-x"),
		},
		{
			name: "[multiple mesh - both selects namespace with different labels] namespaces with matching labels for one mesh can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
					meshWithNSSelectorMeshY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"mesh": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithNSSelectorMeshX,
			wantErr: nil,
		},
		{
			name: "[multiple mesh - both selects namespace with different labels] namespaces with matching labels for another mesh can be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
					meshWithNSSelectorMeshY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"mesh": "y",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    meshWithNSSelectorMeshY,
			wantErr: nil,
		},
		{
			name: "[multiple mesh - both selects namespace with different labels] namespaces without labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
					meshWithNSSelectorMeshY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[multiple mesh - both selects namespaces with different labels] namespaces with non-matching labels cannot be selected",
			env: env{
				meshes: []*appmesh.Mesh{
					meshWithNSSelectorMeshX,
					meshWithNSSelectorMeshY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"some-key": "some-value",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-node",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching mesh for namespace: awesome-ns, expecting 1 but found 0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			designator := NewMembershipDesignator(k8sClient)

			for _, mesh := range tt.env.meshes {
				err := k8sClient.Create(ctx, mesh.DeepCopy())
				assert.NoError(t, err)
			}
			for _, ns := range tt.env.namespaces {
				err := k8sClient.Create(ctx, ns.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := designator.Designate(context.Background(), tt.args.obj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}
