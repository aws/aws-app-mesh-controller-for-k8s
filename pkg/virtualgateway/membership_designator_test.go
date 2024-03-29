package virtualgateway

import (
	"context"
	"testing"

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
)

var (
	noMatchingVGWError = "failed to find matching virtualGateway for gatewayRoute: %s, expecting 1 but found 0"

	vgWithNilNSSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-nil-ns-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: nil,
		},
	}

	vgWithEmptyNSSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-empty-ns-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{},
		},
	}

	vgWithNSSelectorVgX = &appmesh.VirtualGateway{
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
	vgWithNSSelectorVgY = &appmesh.VirtualGateway{
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

	vgWithNSSelectorForBackendApps = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-ns-selector-app-backend",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "backend",
				},
			},
			GatewayRouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"vg": "x",
				},
			},
		},
	}

	vgWithEmptyGWRouteSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-empty-gwRoute-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			GatewayRouteSelector: &metav1.LabelSelector{},
		},
	}

	vgWithNilGWRouteSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-nil-gwRoute-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			GatewayRouteSelector: nil,
		},
	}

	vgWithTestGWRouteSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-test-gwroute-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "backend",
				},
			},
			GatewayRouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"gw": "testGW",
				},
			},
		},
	}

	vgWithProdGWRouteSelector = &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vg-with-test-gwroute-selector",
		},
		Spec: appmesh.VirtualGatewaySpec{
			GatewayRouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"gw": "prodGW",
				},
			},
		},
	}
)

func Test_virtualGatewayMembershipDesignator_matchesNamespaceSelector(t *testing.T) {
	testNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns",
			Labels: map[string]string{
				"app": "backend",
			},
		},
	}

	tests := []struct {
		name           string
		virtualGateway *appmesh.VirtualGateway
		objNS          corev1.Namespace
		want           bool
		wantErr        error
	}{
		{
			name:           "VirtualGateway with Empty namespace selector",
			virtualGateway: vgWithEmptyNSSelector,
			objNS:          testNamespace,
			want:           true,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with Nil namespace selector",
			virtualGateway: vgWithNilNSSelector,
			objNS:          testNamespace,
			want:           false,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with Matching namespace selector",
			virtualGateway: vgWithNSSelectorForBackendApps,
			objNS:          testNamespace,
			want:           true,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with non-matching namespace selector",
			virtualGateway: vgWithNSSelectorVgX,
			objNS:          testNamespace,
			want:           false,
			wantErr:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchesNamespaceSelector(tt.objNS, tt.virtualGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_virtualGatewayMembershipDesignator_matchesGatewayRouteSelector(t *testing.T) {
	testGatewayRoute := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "awesome-ns",
			Name:      "my-gateway-route",
			Labels: map[string]string{
				"gw": "testGW",
			},
		},
	}

	tests := []struct {
		name           string
		virtualGateway *appmesh.VirtualGateway
		gatewayRoute   *appmesh.GatewayRoute
		want           bool
		wantErr        error
	}{
		{
			name:           "VirtualGateway with Empty namespace selector",
			virtualGateway: vgWithEmptyGWRouteSelector,
			gatewayRoute:   testGatewayRoute,
			want:           true,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with Nil namespace selector",
			virtualGateway: vgWithNilGWRouteSelector,
			gatewayRoute:   testGatewayRoute,
			want:           true,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with Matching namespace selector",
			virtualGateway: vgWithTestGWRouteSelector,
			gatewayRoute:   testGatewayRoute,
			want:           true,
			wantErr:        nil,
		},
		{
			name:           "VirtualGateway with non-matching namespace selector",
			virtualGateway: vgWithProdGWRouteSelector,
			gatewayRoute:   testGatewayRoute,
			want:           false,
			wantErr:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchesGatewayRouteSelector(tt.gatewayRoute, tt.virtualGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_virtualGatewayMembershipDesignator_DesignateForGatewayRoute(t *testing.T) {
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
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
			wantErr: errors.New("found multiple matching virtualGateways for gatewayRoute: my-gateway-route, expecting 1 but found 2: vg-with-empty-ns-selector,vg-with-ns-selector-vg-x"),
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
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
			wantErr: errors.Errorf(noMatchingVGWError, "my-gateway-route"),
		},
		{
			name: "virtualgateway with valid gatewayroute selector",
			env: env{
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithNSSelectorForBackendApps,
					vgWithTestGWRouteSelector,
				},
				namespaces: []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "awesome-ns",
							Labels: map[string]string{
								"app": "backend",
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
						Labels: map[string]string{
							"gw": "testGW",
						},
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want:    vgWithTestGWRouteSelector,
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

			for _, vg := range tt.env.virtualGateways {
				err := k8sClient.Create(ctx, vg.DeepCopy())
				assert.NoError(t, err)
			}
			for _, ns := range tt.env.namespaces {
				err := k8sClient.Create(ctx, ns.DeepCopy())
				assert.NoError(t, err)
			}

			ctx = webhook.ContextWithAdmissionRequest(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{Namespace: "awesome-ns"},
			})
			got, err := designator.DesignateForGatewayRoute(ctx, tt.args.obj)
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

	secondTestNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns2",
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
	vgWithPodSelectorPodXSecondNs := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secondTestNS.Name,
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
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for one virtualGateway can be selected",
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
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for another virtualGateway can be selected",
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
			name: "[multiple virtualGateways - both select namespace with different labels] pod with matching labels for both virtualGateways cannot be selected",
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
		{
			name: "[multiple virtualGateways different namespaces with same name ] only the virtualGateway for pod namespaces will be listed and used",
			env: env{
				namespaces: []*corev1.Namespace{testNS, secondTestNS},
				virtualGateways: []*appmesh.VirtualGateway{
					vgWithPodSelectorPodX,
					vgWithPodSelectorPodXSecondNs,
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

			ctx = webhook.ContextWithAdmissionRequest(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{Namespace: "awesome-ns"},
			})

			got, err := designator.DesignateForPod(ctx, tt.args.pod)
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
