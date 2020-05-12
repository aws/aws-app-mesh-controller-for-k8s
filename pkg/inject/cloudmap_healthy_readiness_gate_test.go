package inject

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_cloudMapHealthyReadinessGate_mutate(t *testing.T) {
	vnWithCloudMapServiceDiscovery := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vn-1",
		},
		Spec: appmesh.VirtualNodeSpec{
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "cm-ns",
					ServiceName:   "cm-svc",
				},
			},
		},
	}
	vnWithDNSServiceDiscovery := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vn-2",
		},
		Spec: appmesh.VirtualNodeSpec{
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				DNS: &appmesh.DNSServiceDiscovery{
					Hostname: "www.example.com",
				},
			},
		},
	}
	vnWithNoServiceDiscovery := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vn-3",
		},
		Spec: appmesh.VirtualNodeSpec{},
	}

	type fields struct {
		vn *appmesh.VirtualNode
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPod *corev1.Pod
		wantErr error
	}{
		{
			name: "should add readinessGate if absent",
			fields: fields{
				vn: vnWithCloudMapServiceDiscovery,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ReadinessGates: []corev1.PodReadinessGate{
							{
								ConditionType: "condition-A",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: "condition-A",
						},
						{
							ConditionType: "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
						},
					},
				},
			},
		},
		{
			name: "shouldn't re-add readinessGate if present",
			fields: fields{
				vn: vnWithCloudMapServiceDiscovery,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ReadinessGates: []corev1.PodReadinessGate{
							{
								ConditionType: "condition-A",
							},
							{
								ConditionType: "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: "condition-A",
						},
						{
							ConditionType: "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
						},
					},
				},
			},
		},
		{
			name: "shouldn't add readinessGate if vn is using DNS serviceDiscovery",
			fields: fields{
				vn: vnWithDNSServiceDiscovery,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ReadinessGates: []corev1.PodReadinessGate{
							{
								ConditionType: "condition-A",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: "condition-A",
						},
					},
				},
			},
		},
		{
			name: "shouldn't add readinessGate if vn is using None serviceDiscovery",
			fields: fields{
				vn: vnWithNoServiceDiscovery,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ReadinessGates: []corev1.PodReadinessGate{
							{
								ConditionType: "condition-A",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: "condition-A",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &cloudMapHealthyReadinessGate{
				vn: tt.fields.vn,
			}
			pod := tt.args.pod.DeepCopy()
			err := m.mutate(pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPod, pod)
			}
		})
	}
}
