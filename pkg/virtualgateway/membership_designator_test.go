package virtualgateway

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

func Test_virtualGatewayMembershipDesignator_DesignateForGatewayRoute(t *testing.T) {
	vgWithNilNSSelector := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-nil-ns-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: nil,
		},
	}
	vgWithEmptyNSSelector := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-empty-ns-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{},
		},
	}
	vgWithNSSelectorVgX := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-ns-selector-vg-x",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"vg": "x",
				},
			},
		},
	}
	vgWithNSSelectorVgY := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-ns-selector-vg-y",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"vg": "y",
				},
			},
		},
	}

	type env struct {
		virtualGateways []*appmesh.VirtualGateway
		namespaces      []*corev1.Namespace
	}
	type args struct {
		obj *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualGateway
		wantErr error
	}{
		{
			name: "[a single virtualGateway with empty namespace selector] namespace without labels can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyNSSelector,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[a single virtualGateway with empty namespace selector] namespace with labels can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyNSSelector,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[a single virtualGateway with nil namespace selector] namespace without labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNilNSSelector,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single virtualGateway with nil namespace selector] namespace with labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNilNSSelector,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single virtualGateway selects namespace with specific labels] namespace with matching labels can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"vg": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithNSSelectorVgX,
			wantErr: nil,
		},
		{
			name: "[a single virtualGateway selects namespace with specific labels] namespace without labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[a single virtualGateway selects namespace with specific labels] namespace with non-matching labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[multiple virtualGateways - one with empty namespace selector, another selects namespace with specific labels] namespaces without labels can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyNSSelector,
					vgWithNSSelectorVgX,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - one with empty namespace selector, another selects namespace with specific labels] namespaces with non-matching labels can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyNSSelector,
					vgWithNSSelectorVgX,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithEmptyNSSelector,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - one with empty namespace selector, another selects namespace with specific labels] namespaces with matching labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyNSSelector,
					vgWithNSSelectorVgX,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"vg": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("found multiple matching virtualGateways for namespace: awesome-ns, expecting 1 but found 2: vg-with-empty-ns-selector,vg-with-ns-selector-vg-x"),
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] namespaces with matching labels for one virtualGateway can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
					vgWithNSSelectorVgY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"vg": "x",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithNSSelectorVgX,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] namespaces with matching labels for another virtualGateway can be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
					vgWithNSSelectorVgY,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"vg": "y",
							},
						},
					},
				},
			},
			args: args{
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithNSSelectorVgY,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] namespaces without labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
					vgWithNSSelectorVgY,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
		},
		{
			name: "[multiple virtualGateways - both select namespaces with different labels] namespaces with non-matching labels cannot be selected",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorVgX,
					vgWithNSSelectorVgY,
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
				obj: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gateway-route",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    nil,
			wantErr: errors.New("failed to find matching virtualGateway for namespace: awesome-ns, expecting 1 but found 0"),
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

			for _, vg := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, vg.DeepCopy())
				assert.NoError(t, err)
			}
			for _, ns := range tt.env.namespaces {
				err := k8sClient.Create(ctx, ns.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := designator.DesignateForGatewayRoute(context.Background(), tt.args.obj)
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

func Test_virtualGatewayMembershipDesignator_DesignateForPod(t *testing.T) {
	testNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns",
		},
		Spec: corev1.NamespaceSpec{},
	}
	vgWithNilPodSelector := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vg-with-nil-pod-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			PodSelector: nil,
		},
	}
	vgWithEmptyPodSelector := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vg-with-empty-pod-selector",
		},
		Spec:   appmesh.VirtualGatewaySpec{},
		Status: appmesh.VirtualGatewayStatus{},
	}
	vgWithPodSelectorPodX := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vg-with-pod-selector-pod-x",
		},
		Spec: appmesh.VirtualGatewaySpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pod-x": "true",
				},
			},
		},
	}
	vgWithPodSelectorPodY := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS.Name,
			Name:      "vg-with-pod-selector-pod-y",
		},
		Spec: appmesh.VirtualGatewaySpec{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pod-y": "true",
				},
			},
		},
	}

	type env struct {
		namespaces      []*corev1.Namespace
		virtualGateways []*appmesh.VirtualGateway
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualGateway
		wantErr error
	}{
		{
			name: "[a single virtuaGateway with empty pod selector] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyPodSelector,
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
			name: "[a single virtualGateway with empty pod selector] pod with labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithEmptyPodSelector,
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
			name: "[a single virtualGateway with nil pod selector] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNilPodSelector,
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
			name: "[a single virtualGateway with nil pod selector] pod with labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNilPodSelector,
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
			name: "[a single virtualGateway selects pod with specific labels] pod with matching labels can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
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
			want:    vgWithPodSelectorPodX,
			wantErr: nil,
		},
		{
			name: "[a single virtualGateway selects pod with specific labels] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
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
			name: "[a single virtualGateway selects pod with specific labels] pod with non-matching labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
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
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for one virtualNode can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodY,
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
			want:    vgWithPodSelectorPodX,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for another virtualNode can be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodY,
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
			want:    vgWithPodSelectorPodY,
			wantErr: nil,
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for both virtualNode cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodY,
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
			wantErr: errors.New("found multiple matching VirtualGateways for pod awesome-ns/my-pod: awesome-ns/vg-with-pod-selector-pod-x,awesome-ns/vg-with-pod-selector-pod-y"),
		},
		{
			name: "[multiple virtualGateways - both select namespace with different labels] pod without labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodY,
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
			name: "[multiple virtualGateways - both select namespace with different labels] pod with non-matching labels cannot be selected",
			env: env{
				namespaces: []*corev1.Namespace{testNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodY,
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
			for _, vg := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, vg.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := designator.DesignateForPod(context.Background(), tt.args.pod)
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
