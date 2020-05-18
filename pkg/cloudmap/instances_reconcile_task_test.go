package cloudmap

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ignoreAttrAWSInitHealthStatus(t *testing.T) {
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
			opts := ignoreAttrAWSInitHealthStatus()
			gotEquals := cmp.Equal(tt.desiredAttrs, tt.existingAttrs, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}

func Test_instancesReconcileTask_matchDesiredInstancesAgainstExistingInstances(t *testing.T) {
	type args struct {
		desiredReadyInstanceInfoByID    map[string]instanceInfo
		desiredNotReadyInstanceInfoByID map[string]instanceInfo
		existingInstanceAttrsByID       map[string]instanceAttributes
	}
	tests := []struct {
		name                          string
		args                          args
		wantInstancesToCreateOrUpdate map[string]instanceInfo
		wantInstancesToDelete         []string
	}{
		{
			name: "when all instances needs to be registered",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
					"192.168.1.2": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID:       nil,
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
				"192.168.1.2": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToDelete: []string{},
		},
		{
			name: "when all instances needs to be deregistered",
			args: args{
				desiredReadyInstanceInfoByID:    nil,
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID: map[string]instanceAttributes{
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
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{},
			wantInstancesToDelete:         []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name: "when some instances needs to be deregistered and some needs to be registered",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.2": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToDelete: []string{"192.168.1.2"},
		},
		{
			name: "when some ready instances needs to be report healthCheck",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{},
			wantInstancesToDelete:         []string{},
		},
		{
			name: "when some ready instances needs to be updated",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
							"extraKey":          "value",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
						"extraKey":               "value",
					},
				},
			},
			wantInstancesToDelete: []string{},
		},
		{
			name: "when some ready instances needs to be updated - shouldn't change AWS_INIT_HEALTH_STATUS",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
							"extraKey":          "value",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
						"extraKey":               "value",
					},
				},
			},
			wantInstancesToDelete: []string{},
		},
		{
			name: "when some unready instances needs to report healthCheck",
			args: args{
				desiredReadyInstanceInfoByID: nil,
				desiredNotReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{},
			wantInstancesToDelete:         []string{},
		},
		{
			name: "when some unready instances needs to be updated",
			args: args{
				desiredReadyInstanceInfoByID: nil,
				desiredNotReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
							"extraKey":          "value",
						},
					},
				},
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "UNHEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
						"extraKey":               "value",
					},
				},
			},
			wantInstancesToDelete: []string{},
		},
		{
			name: "when some unready instances needs to be updated - shouldn't change AWS_INIT_HEALTH_STATUS",
			args: args{
				desiredReadyInstanceInfoByID: nil,
				desiredNotReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
							"extraKey":          "value",
						},
					},
				},
				existingInstanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.1": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
						"extraKey":               "value",
					},
				},
			},
			wantInstancesToDelete: []string{},
		},
		{
			name: "when desiredReadyInstancesAttrsByID and desiredNotReadyInstancesAttrsByID and existingInstancesAttrsByID are non-empty",
			args: args{
				desiredReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.1",
							"k8s.io/pod":        "pod1",
							"k8s.io/namespace":  "pod-ns",
						},
					},
					"192.168.1.5": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.5",
							"k8s.io/pod":        "pod5",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				desiredNotReadyInstanceInfoByID: map[string]instanceInfo{
					"192.168.1.3": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.3",
							"k8s.io/pod":        "pod3",
							"k8s.io/namespace":  "pod-ns",
						},
					},
					"192.168.1.6": {
						attrs: instanceAttributes{
							"AWS_INSTANCE_IPV4": "192.168.1.6",
							"k8s.io/pod":        "pod6",
							"k8s.io/namespace":  "pod-ns",
						},
					},
				},
				existingInstanceAttrsByID: map[string]instanceAttributes{
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
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{
				"192.168.1.5": {
					attrs: instanceAttributes{
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.5",
						"k8s.io/pod":             "pod5",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			wantInstancesToDelete: []string{"192.168.1.2", "192.168.1.4"},
		},
		{
			name: "when desiredReadyInstancesAttrsByID and desiredNotReadyInstancesAttrsByID and existingInstancesAttrsByID are empty",
			args: args{
				desiredReadyInstanceInfoByID:    nil,
				desiredNotReadyInstanceInfoByID: nil,
				existingInstanceAttrsByID:       nil,
			},
			wantInstancesToCreateOrUpdate: map[string]instanceInfo{},
			wantInstancesToDelete:         []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &instancesReconcileTask{}
			gotInstancesToCreateOrUpdate, gotInstancesToDelete := r.matchDesiredInstancesAgainstExistingInstances(tt.args.desiredReadyInstanceInfoByID, tt.args.desiredNotReadyInstanceInfoByID, tt.args.existingInstanceAttrsByID)
			assert.Equal(t, tt.wantInstancesToCreateOrUpdate, gotInstancesToCreateOrUpdate)
			assert.Equal(t, tt.wantInstancesToDelete, gotInstancesToDelete)
		})
	}
}
