package inject

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func getConfig(fp func(Config) Config) Config {
	conf := Config{
		IgnoredIPs:                  "169.254.169.254",
		LogLevel:                    "debug",
		Preview:                     false,
		SidecarImage:                "111345817488.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-envoy:latest",
		InitImage:                   "111345817488.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v2",
		SidecarMemory:               "32Mi",
		SidecarCpu:                  "10m",
		EnableIAMForServiceAccounts: true,
	}
	if fp != nil {
		conf = fp(conf)
	}
	return conf
}

func getPod(annotations map[string]string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
			Annotations: map[string]string{
				"some-key": "some-value",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "bar",
				Image: "bar:v1",
				Ports: []corev1.ContainerPort{{ContainerPort: 80}, {ContainerPort: 443}},
			},
			},
		},
		Status: corev1.PodStatus{},
	}
	if annotations != nil {
		for k, v := range annotations {
			pod.Annotations[k] = v
		}
	}
	return pod
}

func getVn(ports []int) *appmesh.VirtualNode {
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "awesome-ns",
			Name:      "my-vn",
		},
		Spec: appmesh.VirtualNodeSpec{
			AWSName: aws.String(""),
			MeshRef: &appmesh.MeshReference{
				Name: "my-mesh",
				UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
			},
		},
	}
	for _, p := range ports {
		listener := appmesh.Listener{PortMapping: appmesh.PortMapping{Port: appmesh.PortNumber(p)}}
		vn.Spec.Listeners = append(vn.Spec.Listeners, listener)
	}
	return vn
}

func getMesh() *appmesh.Mesh {
	return &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
		},
		Spec: appmesh.MeshSpec{
			AWSName: aws.String("my-mesh"),
		},
	}
}

func Test_InjectEnvoyContainer(t *testing.T) {
	type args struct {
		ms  *appmesh.Mesh
		vn  *appmesh.VirtualNode
		pod *corev1.Pod
	}
	type expected struct {
		init       int
		containers int
		xray       bool
		proxyinit  bool
	}
	tests := []struct {
		name    string
		conf    Config
		args    args
		want    expected
		wantErr error
	}{
		{
			name: "Inject Envoy container with proxyinit",
			conf: getConfig(nil),
			args: args{
				ms:  getMesh(),
				vn:  getVn(nil),
				pod: getPod(nil),
			},
			want: expected{
				init:       1,
				containers: 2,
				xray:       false,
				proxyinit:  true,
			},
		},
		{
			name: "Inject Envoy container with xray",
			conf: getConfig(func(cnf Config) Config {
				cnf.EnableXrayTracing = true
				return cnf
			}),
			args: args{
				ms:  getMesh(),
				vn:  getVn(nil),
				pod: getPod(nil),
			},
			want: expected{
				init:       1,
				containers: 3,
				xray:       true,
				proxyinit:  true,
			},
		},
		{
			name: "AppMesh CNI Enabled",
			conf: getConfig(nil),
			args: args{
				ms: getMesh(),
				vn: getVn(nil),
				pod: getPod(map[string]string{
					AppMeshCNIAnnotation: "enabled",
				}),
			},
			want: expected{
				init:       0,
				containers: 2,
				xray:       false,
				proxyinit:  false,
			},
		},
		{
			name: "Enable Jaeger Tracing",
			conf: getConfig(func(cnf Config) Config {
				cnf.EnableJaegerTracing = true
				cnf.JaegerAddress = "addr"
				cnf.JaegerPort = "1234"
				return cnf
			}),
			args: args{
				ms:  getMesh(),
				vn:  getVn(nil),
				pod: getPod(nil),
			},
			want: expected{
				init:       2,
				containers: 2,
				xray:       false,
				proxyinit:  true,
			},
		},
		{
			name: "Enable Datadog Tracing",
			conf: getConfig(func(cnf Config) Config {
				cnf.EnableDatadogTracing = true
				cnf.DatadogAddress = "addr"
				cnf.DatadogPort = "1234"
				return cnf
			}),
			args: args{
				ms:  getMesh(),
				vn:  getVn(nil),
				pod: getPod(nil),
			},
			want: expected{
				init:       2,
				containers: 2,
				xray:       false,
				proxyinit:  true,
			},
		},
		{
			name: "With Pod Annotations",
			conf: getConfig(nil),
			args: args{
				ms: getMesh(),
				vn: getVn(nil),
				pod: getPod(map[string]string{
					AppMeshCPURequestAnnotation:         "20m",
					AppMeshMemoryRequestAnnotation:      "64Mi",
					AppMeshPreviewAnnotation:            "0",
					AppMeshEgressIgnoredPortsAnnotation: "33",
				}),
			},
			want: expected{
				init:       1,
				containers: 2,
				xray:       false,
				proxyinit:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := NewSidecarInjector(tt.conf, "us-west-2", nil, nil, nil)
			pod := tt.args.pod
			inj.injectAppMeshPatches(tt.args.ms, tt.args.vn, pod)
			assert.Equal(t, tt.want.init, len(pod.Spec.InitContainers), "Numbers of init containers mismatch")
			assert.Equal(t, tt.want.containers, len(pod.Spec.Containers), "Numbers of containers mismatch")
			if tt.want.xray {
				found := false
				for _, v := range pod.Spec.Containers {
					if v.Name == "xray-daemon" {
						found = true
					}
				}
				assert.True(t, found, "X-ray container not found")
			}
			if tt.want.proxyinit {
				found := false
				for _, v := range pod.Spec.InitContainers {
					if v.Name == "proxyinit" {
						found = true
					}
				}
				assert.True(t, found, "Proxyinit container not found")
			}
		})
	}
}

func TestSidecarInjector_determineSidecarInjectMode(t *testing.T) {
	nsEnabledSidecarInject := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns",
			Labels: map[string]string{
				"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
			},
		},
	}
	nsDisabledSidecarInject := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awesome-ns",
			Labels: map[string]string{
				"appmesh.k8s.aws/sidecarInjectorWebhook": "disabled",
			},
		},
	}
	nsUnspecifiedSidecarInject := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "awesome-ns",
			Labels: map[string]string{},
		},
	}
	podEnabledSidecarInject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "awesome-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
			},
		},
	}
	podDisabledSidecarInject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "awesome-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/sidecarInjectorWebhook": "disabled",
			},
		},
	}
	podUnspecifiedSidecarInject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "awesome-ns",
			Name:        "my-pod",
			Annotations: map[string]string{},
		},
	}
	// see https://github.com/kubernetes/kubernetes/issues/76680
	podWithUnspecifiedNameAndNamespace := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	type env struct {
		namespaces []*corev1.Namespace
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    sidecarInjectMode
		wantErr error
	}{
		{
			name: "when pod enable sidecar inject",
			env: env{
				namespaces: []*corev1.Namespace{nsUnspecifiedSidecarInject},
			},
			args: args{
				pod: podEnabledSidecarInject,
			},
			want: sidecarInjectModeEnabled,
		},
		{
			name: "when pod disable sidecar inject",
			env: env{
				namespaces: []*corev1.Namespace{nsUnspecifiedSidecarInject},
			},
			args: args{
				pod: podDisabledSidecarInject,
			},
			want: sidecarInjectModeDisabled,
		},
		{
			name: "when pod unspecified sidecar inject",
			env: env{
				namespaces: []*corev1.Namespace{nsUnspecifiedSidecarInject},
			},
			args: args{
				pod: podUnspecifiedSidecarInject,
			},
			want: sidecarInjectModeUnspecified,
		},
		{
			name: "when pod namespace enabled sidecar inject - case 1",
			env: env{
				namespaces: []*corev1.Namespace{nsEnabledSidecarInject},
			},
			args: args{
				pod: podUnspecifiedSidecarInject,
			},
			want: sidecarInjectModeEnabled,
		},
		{
			name: "when pod namespace enabled sidecar inject - case 2",
			env: env{
				namespaces: []*corev1.Namespace{nsEnabledSidecarInject},
			},
			args: args{
				pod: podWithUnspecifiedNameAndNamespace,
			},
			want: sidecarInjectModeEnabled,
		},
		{
			name: "when pod namespace disabled sidecar inject",
			env: env{
				namespaces: []*corev1.Namespace{nsDisabledSidecarInject},
			},
			args: args{
				pod: podUnspecifiedSidecarInject,
			},
			want: sidecarInjectModeDisabled,
		},
		{
			name: "when pod namespace unspecified sidecar inject",
			env: env{
				namespaces: []*corev1.Namespace{nsUnspecifiedSidecarInject},
			},
			args: args{
				pod: podUnspecifiedSidecarInject,
			},
			want: sidecarInjectModeUnspecified,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := &SidecarInjector{
				k8sClient: k8sClient,
			}

			for _, ns := range tt.env.namespaces {
				err := k8sClient.Create(ctx, ns.DeepCopy())
				assert.NoError(t, err)
			}
			ctx = webhook.ContextWithAdmissionRequest(ctx, admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{Namespace: "awesome-ns"},
			})
			got, err := m.determineSidecarInjectMode(ctx, tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
