package aws

import (
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
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

	stats *metrics.Recorder
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

func NewCloud(opts CloudOptions, stats *metrics.Recorder) (CloudAPI, error) {
	cfg := &aws.Config{Region: aws.String(opts.Region)}

	session, err := newAWSSession(cfg, stats, opts.AWSAPIThrottleConfig)
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
		stats: stats,
	}, nil
}

func newAWSSession(cfg *aws.Config, stats *metrics.Recorder, throttleCfg *throttle.ServiceOperationsThrottleConfig) (*session.Session, error) {
	session, err := session.NewSession(cfg)
	if err != nil {
		stats.RecordAWSAPIRequestError("session", "NewSession", getAWSErrorCode(err))
		return nil, err
	}

	if throttleCfg != nil {
		throttler := throttle.NewThrottler(throttleCfg)
		throttler.InjectHandlers(&session.Handlers)
	}

	session.Handlers.Send.PushFront(func(r *request.Request) {
		stats.RecordAWSAPIRequestCount(r.ClientInfo.ServiceName, r.Operation.Name)
	})

	session.Handlers.Complete.PushFront(func(r *request.Request) {
		if r.Error != nil {
			stats.RecordAWSAPIRequestError(r.ClientInfo.ServiceName, r.Operation.Name, getAWSErrorCode(r.Error))
		}
	})

	return session, nil
}

func getAWSErrorCode(err error) string {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return aerr.Code()
		}
		return "internal"
	}
	return ""
}
