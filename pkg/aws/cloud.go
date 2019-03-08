package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/appmesh/appmeshiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
)

type CloudAPI interface {
	AppMeshAPI
}

type Cloud struct {
	region string

	appmesh  appmeshiface.AppMeshAPI
	cloudmap servicediscoveryiface.ServiceDiscoveryAPI
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
		aws.StringValue(cfg.Region),
		appmesh.New(session, cfg),
		servicediscovery.New(session, cfg),
	}, nil
}
