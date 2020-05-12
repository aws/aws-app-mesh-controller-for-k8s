package cloudmap

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func Test_defaultInstancesReconciler_matchDesiredInstancesAgainstExistingInstances(t *testing.T) {
	type args struct {
		desiredReadyInstancesAttrsByID    map[string]InstanceAttributes
		desiredNotReadyInstancesAttrsByID map[string]InstanceAttributes
		existingInstancesAttrsByID        map[string]InstanceAttributes
	}
	tests := []struct {
		name                           string
		args                           args
		wantInstancesToCreateOrUpdate  map[string]InstanceAttributes
		wantInstancesToDelete          []string
		wantInstancesToUpdateHealthy   []string
		wantInstancesToUpdateUnhealthy []string
	}{
		{
			name: "when all instances needs to be registered",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
					"192.168.1.2": {
						"AWS_INSTANCE_IPV4": "192.168.1.",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID:        nil,
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
				},
				"192.168.1.2": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
				},
			},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when all instances needs to be deregistered",
			args: args{
				desiredReadyInstancesAttrsByID:    nil,
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
					"192.168.1.2": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate:  map[string]InstanceAttributes{},
			wantInstancesToDelete:          []string{"192.168.1.1", "192.168.1.2"},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when some instances needs to be deregistered and some needs to be registered",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.2": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
				},
			},
			wantInstancesToDelete:          []string{"192.168.1.2"},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when some ready instances needs to be report healthCheck",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate:  map[string]InstanceAttributes{},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   []string{"192.168.1.1"},
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when some ready instances needs to be updated",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
						"extraKey":          "value",
					},
				},
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
					"extraKey":               "value",
				},
			},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   []string{"192.168.1.1"},
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when some ready instances needs to be updated - shouldn't change AWS_INIT_HEALTH_STATUS",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
						"extraKey":          "value",
					},
				},
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
					"extraKey":               "value",
				},
			},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   []string{"192.168.1.1"},
			wantInstancesToUpdateUnhealthy: nil,
		},
		{
			name: "when some unready instances needs to report healthCheck",
			args: args{
				desiredReadyInstancesAttrsByID: nil,
				desiredNotReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate:  map[string]InstanceAttributes{},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: []string{"192.168.1.1"},
		},
		{
			name: "when some unready instances needs to be updated",
			args: args{
				desiredReadyInstancesAttrsByID: nil,
				desiredNotReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
						"extraKey":          "value",
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
					"extraKey":               "value",
				},
			},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: []string{"192.168.1.1"},
		},
		{
			name: "when some unready instances needs to be updated - shouldn't change AWS_INIT_HEALTH_STATUS",
			args: args{
				desiredReadyInstancesAttrsByID: nil,
				desiredNotReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
						"extraKey":          "value",
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
					"extraKey":               "value",
				},
			},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: []string{"192.168.1.1"},
		},
		{
			name: "when desiredReadyInstancesAttrsByID and desiredNotReadyInstancesAttrsByID and existingInstancesAttrsByID are non-empty",
			args: args{
				desiredReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
						"k8s.io/pod":        "pod1",
						"k8s.io/namespace":  "pod-ns",
					},
					"192.168.1.5": {
						"AWS_INSTANCE_IPV4": "192.168.1.5",
						"k8s.io/pod":        "pod5",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				desiredNotReadyInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.3": {
						"AWS_INSTANCE_IPV4": "192.168.1.3",
						"k8s.io/pod":        "pod3",
						"k8s.io/namespace":  "pod-ns",
					},
					"192.168.1.6": {
						"AWS_INSTANCE_IPV4": "192.168.1.6",
						"k8s.io/pod":        "pod6",
						"k8s.io/namespace":  "pod-ns",
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
					"192.168.1.2": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.2",
						"k8s.io/pod":             "pod2",
						"k8s.io/namespace":       "pod-ns",
					},
					"192.168.1.3": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.3",
						"k8s.io/pod":             "pod3",
						"k8s.io/namespace":       "pod-ns",
					},
					"192.168.1.4": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.4",
						"k8s.io/pod":             "pod4",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]InstanceAttributes{
				"192.168.1.5": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.5",
					"k8s.io/pod":             "pod5",
					"k8s.io/namespace":       "pod-ns",
				},
			},
			wantInstancesToDelete:          []string{"192.168.1.2", "192.168.1.4"},
			wantInstancesToUpdateHealthy:   []string{"192.168.1.1"},
			wantInstancesToUpdateUnhealthy: []string{"192.168.1.3"},
		},
		{
			name: "when desiredReadyInstancesAttrsByID and desiredNotReadyInstancesAttrsByID and existingInstancesAttrsByID are empty",
			args: args{
				desiredReadyInstancesAttrsByID:    nil,
				desiredNotReadyInstancesAttrsByID: nil,
				existingInstancesAttrsByID:        nil,
			},
			wantInstancesToCreateOrUpdate:  map[string]InstanceAttributes{},
			wantInstancesToDelete:          []string{},
			wantInstancesToUpdateHealthy:   nil,
			wantInstancesToUpdateUnhealthy: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			gotInstancesToCreateOrUpdate, gotInstancesToDelete, gotInstancesToUpdateHealthy, gotInstancesToUpdateUnhealthy := r.matchDesiredInstancesAgainstExistingInstances(tt.args.desiredReadyInstancesAttrsByID, tt.args.desiredNotReadyInstancesAttrsByID, tt.args.existingInstancesAttrsByID)
			assert.Equal(t, tt.wantInstancesToCreateOrUpdate, gotInstancesToCreateOrUpdate)
			assert.Equal(t, tt.wantInstancesToDelete, gotInstancesToDelete)
			assert.Equal(t, tt.wantInstancesToUpdateHealthy, gotInstancesToUpdateHealthy)
			assert.Equal(t, tt.wantInstancesToUpdateUnhealthy, gotInstancesToUpdateUnhealthy)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceAttributesByID(t *testing.T) {
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: appmesh.VirtualNodeSpec{
			ServiceDiscovery: &appmesh.ServiceDiscovery{
				AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
			},
		},
	}
	type args struct {
		vn   *appmesh.VirtualNode
		pods []*corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want map[string]InstanceAttributes
	}{
		{
			name: "when there are multiple pod",
			args: args{
				vn: vn,
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod1",
							Labels: map[string]string{
								"app": "my-app",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.1.42",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "pod-ns",
							Name:      "pod2",
							Labels: map[string]string{
								"app": "my-app",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.42.1",
						},
					},
				},
			},
			want: map[string]InstanceAttributes{
				"192.168.1.42": {
					"app":               "my-app",
					"AWS_INSTANCE_IPV4": "192.168.1.42",
					"k8s.io/pod":        "pod1",
					"k8s.io/namespace":  "pod-ns",
				},
				"192.168.42.1": {
					"app":               "my-app",
					"AWS_INSTANCE_IPV4": "192.168.42.1",
					"k8s.io/pod":        "pod2",
					"k8s.io/namespace":  "pod-ns",
				},
			},
		},
		{
			name: "when there are no pods",
			args: args{
				vn:   vn,
				pods: nil,
			},
			want: map[string]InstanceAttributes{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceAttributesByID(tt.args.vn, tt.args.pods)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceProbes(t *testing.T) {
	type args struct {
		pods []*corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want []InstanceProbe
	}{
		{
			name: "when there are multiple pod",
			args: args{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod1",
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.1.42",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod2",
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.42.1",
						},
					},
				},
			},
			want: []InstanceProbe{
				{
					instanceID: "192.168.1.42",
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod1",
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.1.42",
						},
					},
				},
				{
					instanceID: "192.168.42.1",
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod2",
						},
						Status: corev1.PodStatus{
							PodIP: "192.168.42.1",
						},
					},
				},
			},
		},
		{
			name: "when there is no pods",
			args: args{
				pods: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceProbes(tt.args.pods)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesReconciler_buildInstanceAttributes(t *testing.T) {
	type args struct {
		vn  *appmesh.VirtualNode
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want InstanceAttributes
	}{
		{
			name: "attributes should have pod labels",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{},
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
			want: InstanceAttributes{
				"podLabelA":         "valueA",
				"podLabelB":         "valueB",
				"AWS_INSTANCE_IPV4": "192.168.1.42",
				"k8s.io/pod":        "pod-name",
				"k8s.io/namespace":  "pod-ns",
			},
		},
		{
			name: "attributes should have VirtualNode attributes",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
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
			want: InstanceAttributes{
				"attrKeyA":          "valueA",
				"attrKeyB":          "valueB",
				"AWS_INSTANCE_IPV4": "192.168.1.42",
				"k8s.io/pod":        "pod-name",
				"k8s.io/namespace":  "pod-ns",
			},
		},
		{
			name: "attributes should have both pod labels and VirtualNode attributes",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
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
			want: InstanceAttributes{
				"podLabelA":         "valueA",
				"podLabelB":         "valueB",
				"attrKeyA":          "valueA",
				"attrKeyB":          "valueB",
				"AWS_INSTANCE_IPV4": "192.168.1.42",
				"k8s.io/pod":        "pod-name",
				"k8s.io/namespace":  "pod-ns",
			},
		},
		{
			name: "when pod labels or virtualNode attributes contains core attributes, it should be overwritten",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: appmesh.VirtualNodeSpec{
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
			want: InstanceAttributes{
				"podLabelA":         "valueA",
				"attrKeyA":          "valueA",
				"AWS_INSTANCE_IPV4": "192.168.1.42",
				"k8s.io/pod":        "pod-name",
				"k8s.io/namespace":  "pod-ns",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			got := r.buildInstanceAttributes(tt.args.vn, tt.args.pod)
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

func Test_ignoreAttrAwsInitHealthStatus(t *testing.T) {
	tests := []struct {
		name          string
		desiredAttrs  map[string]string
		existingAttrs map[string]string
		wantEquals    bool
	}{
		{
			name: "when they equals except desiredAttrs contains AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value1",
			},
			existingAttrs: map[string]string{
				"key1": "value1",
			},
			wantEquals: true,
		},
		{
			name: "when they differs and desiredAttrs contains AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value1",
			},
			existingAttrs: map[string]string{
				"key1": "value2",
			},
			wantEquals: false,
		},
		{
			name: "when they equals except existingAttrs contains AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"key1": "value1",
			},
			existingAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value1",
			},
			wantEquals: true,
		},
		{
			name: "when they differs and existingAttrs contains AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"key1": "value1",
			},
			existingAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value2",
			},
			wantEquals: false,
		},
		{
			name: "when they equals except they contains different AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value1",
			},
			existingAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
				"key1":                   "value1",
			},
			wantEquals: true,
		},
		{
			name: "when they differs and they contains different AWS_INIT_HEALTH_STATUS",
			desiredAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "HEALTHY",
				"key1":                   "value1",
			},
			existingAttrs: map[string]string{
				"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
				"key1":                   "value2",
			},
			wantEquals: false,
		},
		{
			name: "when they both don't contain AWS_INIT_HEALTH_STATUS, and equals",
			desiredAttrs: map[string]string{
				"key1": "value1",
			},
			existingAttrs: map[string]string{
				"key1": "value1",
			},
			wantEquals: true,
		},
		{
			name: "when they both don't contain AWS_INIT_HEALTH_STATUS, and differs - caseA",
			desiredAttrs: map[string]string{
				"key1": "value1",
			},
			existingAttrs: map[string]string{
				"key1": "value2",
			},
			wantEquals: false,
		},
		{
			name: "when they both don't contain AWS_INIT_HEALTH_STATUS, and differs - caseB",
			desiredAttrs: map[string]string{
				"key1": "value1",
			},
			existingAttrs: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ignoreAttrAwsInitHealthStatus()
			gotEquals := cmp.Equal(tt.desiredAttrs, tt.existingAttrs, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}

func Test_defaultInstancesReconciler_filterExistingInstancesAttrsIDForVirtualNode(t *testing.T) {
	type args struct {
		vn                         *appmesh.VirtualNode
		existingInstancesAttrsByID map[string]InstanceAttributes
	}
	tests := []struct {
		name string
		args args
		want map[string]InstanceAttributes
	}{
		{
			name: "when contains instances from another namespace",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.VirtualNodeSpec{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "my-app",
							},
						},
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"app":                    "my-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "my-ns",
					},
					"192.168.1.2": {
						"app":                    "my-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.2",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod2",
						"k8s.io/namespace":       "another-ns",
					},
					"192.168.1.3": {
						"app":                    "my-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.3",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod3",
						"k8s.io/namespace":       "my-ns",
					},
				},
			},
			want: map[string]InstanceAttributes{
				"192.168.1.1": {
					"app":                    "my-app",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "my-ns",
				},
				"192.168.1.3": {
					"app":                    "my-app",
					"AWS_INSTANCE_IPV4":      "192.168.1.3",
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"k8s.io/pod":             "pod3",
					"k8s.io/namespace":       "my-ns",
				},
			},
		},
		{
			name: "when contains instances from another virtualnode",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
					},
					Spec: appmesh.VirtualNodeSpec{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "my-app",
							},
						},
					},
				},
				existingInstancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"app":                    "my-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "my-ns",
					},
					"192.168.1.2": {
						"app":                    "another-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.2",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod2",
						"k8s.io/namespace":       "my-ns",
					},
					"192.168.1.3": {
						"app":                    "my-app",
						"AWS_INSTANCE_IPV4":      "192.168.1.3",
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"k8s.io/pod":             "pod3",
						"k8s.io/namespace":       "my-ns",
					},
				},
			},
			want: map[string]InstanceAttributes{
				"192.168.1.1": {
					"app":                    "my-app",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "my-ns",
				},
				"192.168.1.3": {
					"app":                    "my-app",
					"AWS_INSTANCE_IPV4":      "192.168.1.3",
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"k8s.io/pod":             "pod3",
					"k8s.io/namespace":       "my-ns",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &defaultInstancesReconciler{}
			if got := r.filterExistingInstancesAttrsIDForVirtualNode(tt.args.vn, tt.args.existingInstancesAttrsByID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterExistingInstancesAttrsIDForVirtualNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
