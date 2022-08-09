package cloudmap

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_defaultInstancesReconciler_buildInstanceAttributes(t *testing.T) {
	type args struct {
		ms  *appmesh.Mesh
		vn  *appmesh.VirtualNode
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want instanceAttributes
	}{
		{
			name: "attributes should have pod labels",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels: map[string]string{
							"podLabelA": "valueA",
							"podLabelB": "valueB",
						},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: instanceAttributes{
				"podLabelA":                   "valueA",
				"podLabelB":                   "valueB",
				"AWS_INSTANCE_IPV4":           "192.168.1.42",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
		{
			name: "attributes should have VirtualNode attributes",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								Attributes: []appmesh.AWSCloudMapInstanceAttribute{
									{
										Key:   "attrKeyA",
										Value: "valueA",
									},
									{
										Key:   "attrKeyB",
										Value: "valueB",
									},
								},
							},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels:    map[string]string{},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: instanceAttributes{
				"attrKeyA":                    "valueA",
				"attrKeyB":                    "valueB",
				"AWS_INSTANCE_IPV4":           "192.168.1.42",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
		{
			name: "attributes should have both pod labels and VirtualNode attributes",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								Attributes: []appmesh.AWSCloudMapInstanceAttribute{
									{
										Key:   "attrKeyA",
										Value: "valueA",
									},
									{
										Key:   "attrKeyB",
										Value: "valueB",
									},
								},
							},
						},

						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels: map[string]string{
							"podLabelA": "valueA",
							"podLabelB": "valueB",
						},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: instanceAttributes{
				"podLabelA":                   "valueA",
				"podLabelB":                   "valueB",
				"attrKeyA":                    "valueA",
				"attrKeyB":                    "valueB",
				"AWS_INSTANCE_IPV4":           "192.168.1.42",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
		{
			name: "when pod labels or virtualNode attributes contains core attributes, it should be overwritten",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								Attributes: []appmesh.AWSCloudMapInstanceAttribute{
									{
										Key:   "attrKeyA",
										Value: "valueA",
									},
									{
										Key:   "AWS_INSTANCE_IPV4",
										Value: "valueB",
									},
								},
							},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels: map[string]string{
							"podLabelA":         "valueA",
							"AWS_INSTANCE_IPV4": "valueB",
						},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: instanceAttributes{
				"podLabelA":                   "valueA",
				"attrKeyA":                    "valueA",
				"AWS_INSTANCE_IPV4":           "192.168.1.42",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceAttributes(tt.args.ms, tt.args.vn, tt.args.pod, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceID(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal case",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: "192.168.1.42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceID(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceInfoByID(t *testing.T) {
	type args struct {
		ms   *appmesh.Mesh
		vn   *appmesh.VirtualNode
		pods []*corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want map[string]instanceInfo
	}{
		{
			name: "multiple pods",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod-name-1",
							Labels: map[string]string{
								"podLabelA": "valueA",
								"podLabelB": "valueB",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							PodIP: "192.168.1.42",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod-name-2",
							Labels: map[string]string{
								"podLabelA": "valueA",
								"podLabelB": "valueB",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							PodIP: "192.168.2.42",
						},
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.42": {
					attrs: instanceAttributes{
						"podLabelA":                   "valueA",
						"podLabelB":                   "valueB",
						"AWS_INSTANCE_IPV4":           "192.168.1.42",
						"AWS_INSTANCE_PORT":           "8080",
						"k8s.io/pod":                  "pod-name-1",
						"k8s.io/namespace":            "pod-ns",
						"appmesh.k8s.aws/mesh":        "my-mesh",
						"appmesh.k8s.aws/virtualNode": "my-vn",
					},
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod-name-1",
							Labels: map[string]string{
								"podLabelA": "valueA",
								"podLabelB": "valueB",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							PodIP: "192.168.1.42",
						},
					},
				},
				"192.168.2.42": {
					attrs: instanceAttributes{
						"podLabelA":                   "valueA",
						"podLabelB":                   "valueB",
						"AWS_INSTANCE_IPV4":           "192.168.2.42",
						"AWS_INSTANCE_PORT":           "8080",
						"k8s.io/pod":                  "pod-name-2",
						"k8s.io/namespace":            "pod-ns",
						"appmesh.k8s.aws/mesh":        "my-mesh",
						"appmesh.k8s.aws/virtualNode": "my-vn",
					},
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod-name-2",
							Labels: map[string]string{
								"podLabelA": "valueA",
								"podLabelB": "valueB",
							},
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							PodIP: "192.168.2.42",
						},
					},
				},
			},
		},
		{
			name: "nil pods",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pods: nil,
			},
			want: map[string]instanceInfo{},
		},
		{
			name: "empty pods",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pods: []*corev1.Pod{},
			},
			want: map[string]instanceInfo{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceInfoByID(tt.args.ms, tt.args.vn, tt.args.pods, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceAttributes_IPv6(t *testing.T) {
	type args struct {
		ms  *appmesh.Mesh
		vn  *appmesh.VirtualNode
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want instanceAttributes
	}{
		{
			name: "should have AWS_INSTANCE_IPV6 set to pod's IPv6 address",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels: map[string]string{
							"podLabelA": "valueA",
							"podLabelB": "valueB",
						},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "2001:4860:4860::8888",
					},
				},
			},
			want: instanceAttributes{
				"podLabelA":                   "valueA",
				"podLabelB":                   "valueB",
				"AWS_INSTANCE_IPV6":           "2001:4860:4860::8888",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{
				ipFamily: IPv6,
			}
			got := r.buildInstanceAttributes(tt.args.ms, tt.args.vn, tt.args.pod, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceAttributes_IPv4(t *testing.T) {
	type args struct {
		ms  *appmesh.Mesh
		vn  *appmesh.VirtualNode
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want instanceAttributes
	}{
		{
			name: "should have AWS_INSTANCE_IPV4 set to pod's IPv4 address",
			args: args{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
						},
						Listeners: []appmesh.Listener{{
							PortMapping: appmesh.PortMapping{
								Port: appmesh.PortNumber(8080),
							}},
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "pod-ns",
						Name:      "pod-name",
						Labels: map[string]string{
							"podLabelA": "valueA",
							"podLabelB": "valueB",
						},
					},
					Spec: corev1.PodSpec{},
					Status: corev1.PodStatus{
						PodIP: "192.168.1.42",
					},
				},
			},
			want: instanceAttributes{
				"podLabelA":                   "valueA",
				"podLabelB":                   "valueB",
				"AWS_INSTANCE_IPV4":           "192.168.1.42",
				"AWS_INSTANCE_PORT":           "8080",
				"k8s.io/pod":                  "pod-name",
				"k8s.io/namespace":            "pod-ns",
				"appmesh.k8s.aws/mesh":        "my-mesh",
				"appmesh.k8s.aws/virtualNode": "my-vn",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{
				ipFamily: IPv4,
			}
			got := r.buildInstanceAttributes(tt.args.ms, tt.args.vn, tt.args.pod, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
