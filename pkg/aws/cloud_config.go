package aws

import (
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"github.com/spf13/pflag"
)

const (
	flagAWSRegion      = "aws-region"
	flagAWSAccountID   = "aws-account-id"
	flagAWSAPIThrottle = "aws-api-throttle"
	flagAWSSTSEndpoint = "aws-sts-endpoint"
)

type CloudConfig struct {
	// AWS Region for the kubernetes cluster
	Region string
	// AccountID for the kubernetes cluster
	AccountID string
	// AWS STS Endpoint override for the controller
	STSEndpoint string
	// Throttle settings for aws APIs
	ThrottleConfig *throttle.ServiceOperationsThrottleConfig
}

func (cfg *CloudConfig) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&cfg.Region, flagAWSRegion, "", "AWS Region for the kubernetes cluster")
	fs.StringVar(&cfg.AccountID, flagAWSAccountID, "", "AWS AccountID for the kubernetes cluster")
	fs.StringVar(&cfg.STSEndpoint, flagAWSSTSEndpoint, "", "AWS STS endpoint override for the controller")
	fs.Var(cfg.ThrottleConfig, flagAWSAPIThrottle, "throttle settings for AWS APIs, format: serviceID1:operationRegex1=rate:burst,serviceID2:operationRegex2=rate:burst")
}
