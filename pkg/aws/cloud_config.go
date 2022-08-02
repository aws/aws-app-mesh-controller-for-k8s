package aws

import (
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"regexp"
	"strings"
)

const (
	flagAWSRegion      = "aws-region"
	flagAWSAccountID   = "aws-account-id"
	flagAWSAPIThrottle = "aws-api-throttle"
)

type CloudConfig struct {
	// AWS Region for the kubernetes cluster
	Region string
	// AccountID for the kubernetes cluster
	AccountID string
	// Throttle settings for aws APIs
	ThrottleConfig *throttle.ServiceOperationsThrottleConfig
}

func (cfg *CloudConfig) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&cfg.Region, flagAWSRegion, "", "AWS Region for the kubernetes cluster")
	fs.StringVar(&cfg.AccountID, flagAWSAccountID, "", "AWS AccountID for the kubernetes cluster")
	fs.Var(cfg.ThrottleConfig, flagAWSAPIThrottle, "throttle settings for AWS APIs, format: serviceID1:operationRegex1=rate:burst,serviceID2:operationRegex2=rate:burst")
}

//function to check if aws accountId got converted to scientific notation, and convert back
func (cfg *CloudConfig) HandleAccountID(log logr.Logger) {
	matched, _ := regexp.MatchString("^([0-9]{1}[.])([0-9]{11})(e\\+11)", cfg.AccountID)
	if matched {
		log.Error(nil, "The following AWS Account ID is not formatted correctly: "+cfg.AccountID)
		cfg.AccountID = cfg.AccountID[0:13]
		cfg.AccountID = strings.Replace(cfg.AccountID, ".", "", 1)
		log.Error(nil, "Using the converted AWS Account ID: "+cfg.AccountID)
	}
}
