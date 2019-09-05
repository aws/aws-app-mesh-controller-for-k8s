package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/appmesh/appmeshiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	"k8s.io/client-go/tools/cache"
)

type CloudAPI interface {
	AppMeshAPI
	CloudMapAPI
}

type Cloud struct {
	region string

	appmesh  appmeshiface.AppMeshAPI
	cloudmap servicediscoveryiface.ServiceDiscoveryAPI

	namespaceIDCache cache.Store
	serviceIDCache   cache.Store
}

type cloudmapServiceCacheItem struct {
	key   string
	value CloudMapServiceSummary
}

type CloudMapServiceSummary struct {
	NamespaceID string
	ServiceID   string
}

type cloudmapNamespaceCacheItem struct {
	key   string
	value CloudMapNamespaceSummary
}

type CloudMapNamespaceSummary struct {
	NamespaceID   string
	NamespaceType string
}

func NewCloud(opts CloudOptions) (CloudAPI, error) {
	cfg := &aws.Config{Region: aws.String(opts.Region)}

	session, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	metadata := ec2metadata.New(session)

	if len(aws.StringValue(cfg.Region)) == 0 {
		region, err := metadata.Region()
		if err != nil {
			return nil, fmt.Errorf("failed to get region from metadata, specify --aws-region instead if ec2Metadata is unavailable: %s", err)
		}
		cfg.Region = aws.String(region)
	}

	return &Cloud{
		region:   aws.StringValue(cfg.Region),
		appmesh:  appmesh.New(session, cfg),
		cloudmap: servicediscovery.New(session, cfg),
		namespaceIDCache: cache.NewTTLStore(func(obj interface{}) (string, error) {
			return obj.(*cloudmapNamespaceCacheItem).key, nil
		}, 60*time.Second),
		serviceIDCache: cache.NewTTLStore(func(obj interface{}) (string, error) {
			return obj.(*cloudmapServiceCacheItem).key, nil
		}, 60*time.Second),
	}, nil
}
