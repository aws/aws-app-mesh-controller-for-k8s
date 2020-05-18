package cloudmap

import (
	"context"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/cache"
	"testing"
	"time"
)

func Test_defaultInstancesCache_cloneInstanceAttributesByID(t *testing.T) {
	type args struct {
		instanceAttrsByID map[string]instanceAttributes
	}
	tests := []struct {
		name string
		args args
		want map[string]instanceAttributes
	}{
		{
			name: "when it's non-empty",
			args: args{
				instanceAttrsByID: map[string]instanceAttributes{
					"192.168.1.1": {
						"AWS_INIT_HEALTH_STATUS": "HEALTHY",
						"AWS_INSTANCE_IPV4":      "192.168.1.1",
						"k8s.io/pod":             "pod1",
						"k8s.io/namespace":       "pod-ns",
					},
				},
			},
			want: map[string]instanceAttributes{
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
				instanceAttrsByID: nil,
			},
			want: map[string]instanceAttributes{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultInstancesCache{}
			got := c.cloneInstanceAttributesByID(tt.args.instanceAttrsByID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultInstancesCache_recordSuccessfulRegisterInstanceOperation(t *testing.T) {
	now := time.Now()
	oneSecAfterNow := now.Add(1 * time.Second)
	oneSecBeforeNow := now.Add(-1 * time.Second)
	type fields struct {
		instancesAttrsCacheItemByService map[string]*instancesAttrsCacheItem
	}
	type args struct {
		serviceID  string
		instanceID string
		attrs      instanceAttributes
		operation  *servicediscovery.Operation
	}
	tests := []struct {
		name                       string
		fields                     fields
		args                       args
		wantInstanceAttrsByService map[string]map[string]instanceAttributes
	}{
		{
			name: "when cache for service presents and updateTime pasts lastUpdateTime, should update cache",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.1",
				attrs: instanceAttributes{
					attrAWSInstanceIPV4: "192.168.1.1",
					"k":                 "v",
				},
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecAfterNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.1",
						"k":                 "v",
					},
				},
			},
		},
		{
			name: "when cache for service presents and lastUpdateTime empty, should update cache",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
						},
						lastUpdatedTimeByID: make(map[string]time.Time),
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.1",
				attrs: instanceAttributes{
					attrAWSInstanceIPV4: "192.168.1.1",
					"k":                 "v",
				},
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecAfterNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.1",
						"k":                 "v",
					},
				},
			},
		},
		{
			name: "when cache for service presents and updateTime before lastUpdateTime, should be no-op",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.1",
				attrs: instanceAttributes{
					attrAWSInstanceIPV4: "192.168.1.1",
					"k":                 "v",
				},
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecBeforeNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.1",
					},
				},
			},
		},
		{
			name: "when cache for service presents and updateTime pasts lastUpdateTime, should add new items into cache",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.2",
				attrs: instanceAttributes{
					attrAWSInstanceIPV4: "192.168.1.2",
				},
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecAfterNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.1",
					},
					"192.168.1.2": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.2",
					},
				},
			},
		},
		{
			name: "when cache for service absents, should be no-op",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
						},
					},
				},
			},
			args: args{
				serviceID:  "service-B",
				instanceID: "192.168.1.1",
				attrs: instanceAttributes{
					attrAWSInstanceIPV4: "192.168.1.1",
					"k":                 "v",
				},
				operation: nil,
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": instanceAttributes{
						attrAWSInstanceIPV4: "192.168.1.1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instancesAttrsCache := cache.NewLRUExpireCache(defaultInstanceAttrsCacheSize)
			c := &defaultInstancesCache{
				instancesAttrsCache: instancesAttrsCache,
			}
			for serviceID, cacheItem := range tt.fields.instancesAttrsCacheItemByService {
				instancesAttrsCache.Add(serviceID, cacheItem, defaultInstanceAttrsCacheTTL)
			}
			c.recordSuccessfulRegisterInstanceOperation(tt.args.serviceID, tt.args.instanceID, tt.args.attrs, tt.args.operation)
			for serviceID, wantInstanceAttrsByID := range tt.wantInstanceAttrsByService {
				gotInstanceAttrsByID, err := c.ListInstances(context.Background(), serviceID)
				assert.NoError(t, err)
				assert.Equal(t, wantInstanceAttrsByID, gotInstanceAttrsByID)
			}
		})
	}
}

func Test_defaultInstancesCache_recordSuccessfulDeregisterInstanceOperation(t *testing.T) {
	now := time.Now()
	oneSecAfterNow := now.Add(1 * time.Second)
	oneSecBeforeNow := now.Add(-1 * time.Second)
	type fields struct {
		instancesAttrsCacheItemByService map[string]*instancesAttrsCacheItem
	}
	type args struct {
		serviceID  string
		instanceID string
		operation  *servicediscovery.Operation
	}
	tests := []struct {
		name                       string
		fields                     fields
		args                       args
		wantInstanceAttrsByService map[string]map[string]instanceAttributes
	}{
		{
			name: "when cache for service presents and updateTime pasts lastUpdateTime, should remove from cache",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
							"192.168.1.2": {
								attrAWSInstanceIPV4: "192.168.1.2",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
							"192.168.1.2": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.1",
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecAfterNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.2": {
						attrAWSInstanceIPV4: "192.168.1.2",
					},
				},
			},
		},
		{
			name: "when cache for service presents and updateTime before lastUpdateTime, should be no-op",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
							"192.168.1.2": {
								attrAWSInstanceIPV4: "192.168.1.2",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
							"192.168.1.2": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-A",
				instanceID: "192.168.1.1",
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecBeforeNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": {
						attrAWSInstanceIPV4: "192.168.1.1",
					},
					"192.168.1.2": {
						attrAWSInstanceIPV4: "192.168.1.2",
					},
				},
			},
		},
		{
			name: "when cache for service absents, should be no-op",
			fields: fields{
				instancesAttrsCacheItemByService: map[string]*instancesAttrsCacheItem{
					"service-A": {
						instanceAttrsByID: map[string]instanceAttributes{
							"192.168.1.1": {
								attrAWSInstanceIPV4: "192.168.1.1",
							},
							"192.168.1.2": {
								attrAWSInstanceIPV4: "192.168.1.2",
							},
						},
						lastUpdatedTimeByID: map[string]time.Time{
							"192.168.1.1": now,
							"192.168.1.2": now,
						},
					},
				},
			},
			args: args{
				serviceID:  "service-B",
				instanceID: "192.168.1.1",
				operation: &servicediscovery.Operation{
					UpdateDate: &oneSecAfterNow,
				},
			},
			wantInstanceAttrsByService: map[string]map[string]instanceAttributes{
				"service-A": {
					"192.168.1.1": {
						attrAWSInstanceIPV4: "192.168.1.1",
					},
					"192.168.1.2": {
						attrAWSInstanceIPV4: "192.168.1.2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instancesAttrsCache := cache.NewLRUExpireCache(defaultInstanceAttrsCacheSize)
			c := &defaultInstancesCache{
				instancesAttrsCache: instancesAttrsCache,
			}
			for serviceID, cacheItem := range tt.fields.instancesAttrsCacheItemByService {
				instancesAttrsCache.Add(serviceID, cacheItem, defaultInstanceAttrsCacheTTL)
			}
			c.recordSuccessfulDeregisterInstanceOperation(tt.args.serviceID, tt.args.instanceID, tt.args.operation)
			for serviceID, wantInstanceAttrsByID := range tt.wantInstanceAttrsByService {
				gotInstanceAttrsByID, err := c.ListInstances(context.Background(), serviceID)
				assert.NoError(t, err)
				assert.Equal(t, wantInstanceAttrsByID, gotInstanceAttrsByID)
			}
		})
	}
}
