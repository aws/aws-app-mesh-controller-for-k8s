package cloudmap

import (
	"context"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/cache"
	"sync"
	"testing"
	"time"
)

func Test_defaultInstancesCache_recordSuccessOperation(t *testing.T) {

	type fields struct {
		serviceInstanceAttributes map[string]map[string]InstanceAttributes
	}
	type args struct {
		instanceWithinServiceID instanceWithinServiceID
		operation               operationInfo
		sdkOperation            *servicediscovery.Operation
	}
	tests := []struct {
		name                          string
		fields                        fields
		args                          args
		wantServiceInstanceAttributes map[string]map[string]InstanceAttributes
	}{
		{
			name: "success registerInstance operation should add to instance cache",
			fields: fields{
				serviceInstanceAttributes: map[string]map[string]InstanceAttributes{
					"srv-xxxx": {
						"192.168.1.1": {
							"AWS_INSTANCE_IPV4": "192.168.1.1",
						},
					},
				},
			},
			args: args{
				instanceWithinServiceID: instanceWithinServiceID{
					serviceID:  "srv-xxxx",
					instanceID: "192.168.1.2",
				},
				operation: operationInfo{
					operationID:   "operation-a",
					operationType: servicediscovery.OperationTypeRegisterInstance,
					instanceAttrs: map[string]string{
						"AWS_INSTANCE_IPV4": "192.168.1.2",
					},
				},
			},
			wantServiceInstanceAttributes: map[string]map[string]InstanceAttributes{
				"srv-xxxx": {
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
					},
					"192.168.1.2": {
						"AWS_INSTANCE_IPV4": "192.168.1.2",
					},
				},
			},
		},
		{
			name: "success deregisterInstance operation should remove from cache",
			fields: fields{
				serviceInstanceAttributes: map[string]map[string]InstanceAttributes{
					"srv-xxxx": {
						"192.168.1.1": {
							"AWS_INSTANCE_IPV4": "192.168.1.1",
						},
						"192.168.1.2": {
							"AWS_INSTANCE_IPV4": "192.168.1.2",
						},
					},
				},
			},
			args: args{
				instanceWithinServiceID: instanceWithinServiceID{
					serviceID:  "srv-xxxx",
					instanceID: "192.168.1.2",
				},
				operation: operationInfo{
					operationID:   "operation-b",
					operationType: servicediscovery.OperationTypeDeregisterInstance,
					instanceAttrs: nil,
				},
			},
			wantServiceInstanceAttributes: map[string]map[string]InstanceAttributes{
				"srv-xxxx": {
					"192.168.1.1": {
						"AWS_INSTANCE_IPV4": "192.168.1.1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instancesAttrsCache := cache.NewLRUExpireCache(1024)
			for serviceID, instanceAttrs := range tt.fields.serviceInstanceAttributes {
				instanceAttrsCache := &instancesAttrsCacheItem{
					instancesAttrsByID: make(map[string]InstanceAttributes),
					mutex:              sync.RWMutex{},
				}
				for instanceID, attrs := range instanceAttrs {
					instanceAttrsCache.instancesAttrsByID[instanceID] = attrs
				}
				instancesAttrsCache.Add(serviceID, instanceAttrsCache, 100*time.Second)
			}
			c := &defaultInstancesCache{instancesAttrsCache: instancesAttrsCache}
			c.recordSuccessOperation(context.Background(), tt.args.instanceWithinServiceID, tt.args.operation, tt.args.sdkOperation)
			for serviceID, instanceAttrs := range tt.wantServiceInstanceAttributes {
				cacheValue, exists := instancesAttrsCache.Get(serviceID)
				assert.True(t, exists)
				instancesAttrsCacheItem := cacheValue.(*instancesAttrsCacheItem)
				assert.Equal(t, instanceAttrs, instancesAttrsCacheItem.instancesAttrsByID)
			}
		})
	}
}

func Test_defaultInstancesCache_cloneInstanceAttributesByID(t *testing.T) {
	type args struct {
		instancesAttrsByID map[string]InstanceAttributes
	}
	tests := []struct {
		name string
		args args
		want map[string]InstanceAttributes
	}{
		{
			name: "when it's non-empty",
			args: args{
				instancesAttrsByID: map[string]InstanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			want: map[string]InstanceAttributes{
				"192.168.1.1": {
					"AWS_INIT_HEALTH_STATUS": "HEALTHY",
					"AWS_INSTANCE_IPV4":      "192.168.1.1",
					"k8s.io/pod":             "pod1",
					"k8s.io/namespace":       "pod-ns",
				},
			},
		},
		{
			name: "when it's nil",
			args: args{
				instancesAttrsByID: nil,
			},
			want: map[string]InstanceAttributes{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultInstancesCache{}
			got := c.cloneInstanceAttributesByID(tt.args.instancesAttrsByID)
			assert.Equal(t, tt.want, got)
		})
	}
}
