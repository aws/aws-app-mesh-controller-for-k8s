package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func Test_membershipDesignator_Designate(t *testing.T) {
	testNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns",
		},
		Spec: corev1.NamespaceSpec{},
	}
	secondTestNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns2",
		},
		Spec: corev1.NamespaceSpec{},
	}
	vnWithNilPodSelector := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vn-with-nil-pod-selector",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: nil,
		},
	}
	vnWithEmptyPodSelector := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vn-with-empty-pod-selector",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{},
		},
		Status: appmesh.VirtualNodeStatus{},
	}
	vnWithPodSelectorPodX := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vn-with-pod-selector-pod-x",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pod-x": "true",
				},
			},
		},
	}
	vnWithPodSelectorPodY := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vn-with-pod-selector-pod-y",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pod-y": "true",
				},
			},
		},
	}

	vnWithPodSelectorPodXSecondNs := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secondTestNS.Name,
			Name:      "vn-with-pod-selector-pod-x",
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pod-x": "true",
				},
			},
		},
	}

	type env struct {
		namespaces   []*corev1.Namespace
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "[a single virtualNode with empty pod selector] pod without labels can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithEmptyPodSelector,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithEmptyPodSelector,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode with empty pod selector] pod with labels can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithEmptyPodSelector,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"any-key": "any-value",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithEmptyPodSelector,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode with nil pod selector] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithNilPodSelector,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode with nil pod selector] pod with labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithNilPodSelector,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"any-key": "any-value",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode selects pod with specific labels] pod with matching labels can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"pod-x": "true",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithPodSelectorPodX,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode selects pod with specific labels] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "[a single virtualNode selects pod with specific labels] pod with non-matching labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"some-key": "some-value",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "[multiple virtualNode - both selects namespace with different labels] pod with matching labels for one virtualNode can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodY,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"pod-x": "true",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithPodSelectorPodX,
			wantErr: nil,
		},
		{
			name: "[multiple virtualNode - both selects namespace with different labels] pod with matching labels for another virtualNode can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodY,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"pod-y": "true",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithPodSelectorPodY,
			wantErr: nil,
		},
		{
			name: "[multiple virtualNode - both selects namespace with different labels] pod with matching labels for both virtualNode cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodY,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"pod-x": "true",
							"pod-y": "true",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("found multiple matching VirtualNodes for pod awesome-ns/my-pod: awesome-ns/vn-with-pod-selector-pod-x,awesome-ns/vn-with-pod-selector-pod-y"),
		},
		{
			name: "[multiple virtualNode different namespaces with same name ] only the virtualNode for pod namespaces will be listed and used",
			env: env{
				namespaces: []*corev1.Namespace{testNS, secondTestNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodXSecondNs,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"pod-x": "true",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    vnWithPodSelectorPodX,
			wantErr: nil,
		},
		{
			name: "[multiple virtualNode - both selects namespace with different labels] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodY,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "[multiple virtualNode - both selects namespace with different labels] pod with non-matching labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualNodes: []*appmesh.VirtualNode{
					vnWithPodSelectorPodX,
					vnWithPodSelectorPodY,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS.Name,
						Name:      "my-pod",
						Labels: map[string]string{
							"some-key": "some-value",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			want:    nil,
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
			designator := NewMembershipDesignator(k8sClient)

			for _, ns := range tt.env.namespaces {
				err := k8sClient.Create(ctx, ns.DeepCopy())
				assert.NoError(t, err)
			}
			for _, mesh := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, mesh.DeepCopy())
				assert.NoError(t, err)
			}
			ctx = webhook.ContextWithAdmissionRequest(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{Namespace: "awesome-ns"},
			})

			got, err := designator.Designate(ctx, tt.args.pod)
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
