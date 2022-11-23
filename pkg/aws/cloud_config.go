package aws

import (
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"regexp"
	"strings"
)

const (
	flagAWSRegion               = "aws-region"
	flagAWSAccountID            = "aws-account-id"
	flagAWSAPIThrottle          = "aws-api-throttle"
	flagUseAwsFipsEndpoint      = "use-aws-fips-endpoint"
	flagUseAwsDualStackEndpoint = "use-aws-dual-stack-endpoint"
)

type CloudConfig struct {
	// AWS Region for the kubernetes cluster
	Region string
	// AccountID for the kubernetes cluster
	AccountID string
	// Throttle settings for aws APIs
	ThrottleConfig *throttle.ServiceOperationsThrottleConfig
	// DualStackEndpoint flag for aws APIs
	UseAwsDualStackEndpoint bool
	// FipsEndpoint flag for aws APIs
	UseAwsFIPSEndpoint bool
}

func (cfg *CloudConfig) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&cfg.Region, flagAWSRegion, "", "AWS Region for the kubernetes cluster")
	fs.StringVar(&cfg.AccountID, flagAWSAccountID, "", "AWS AccountID for the kubernetes cluster")
	fs.Var(cfg.ThrottleConfig, flagAWSAPIThrottle, "throttle settings for AWS APIs, format: serviceID1:operationRegex1=rate:burst,serviceID2:operationRegex2=rate:burst")
	fs.BoolVar(&cfg.UseAwsFIPSEndpoint, flagUseAwsFipsEndpoint, true, "To use FIPS Endpoint for AWS Services")
	fs.BoolVar(&cfg.UseAwsDualStackEndpoint, flagUseAwsDualStackEndpoint, false, "To use Dual Stack Endpoint for AWS Services")
}

// function to check if aws accountId got converted to scientific notation, and convert back
// silently log any improperly formatted ids
func (cfg *CloudConfig) HandleAccountID(log logr.Logger) {
	properIDMatched, _ := regexp.MatchString("^(\\d{12})$", cfg.AccountID)

	if properIDMatched || cfg.AccountID == "" {
		return
	}
	log.Error(nil, "The following AWS Account ID is not formatted correctly: "+cfg.AccountID)

	scientificMatched, _ := regexp.MatchString("^(\\d[.])(\\d{11})(e\\+11)$", cfg.AccountID)
	if scientificMatched {
		cfg.AccountID = cfg.AccountID[0:13]
		cfg.AccountID = strings.Replace(cfg.AccountID, ".", "", 1)
		log.Error(nil, fmt.Sprintf("Using the converted AWS Account ID: %s", cfg.AccountID))
	}
}

// converts boolean values of Fips Endpoint into Integer which are used for AWS Services
func (cfg *CloudConfig) GetAwsDualStackEndpoint() int {
	if cfg.UseAwsDualStackEndpoint {
		return 1
	} else {
		return 0
	}
}

// converts boolean values of Fips Endpoint into Integer which are used for AWS Services
func (cfg *CloudConfig) GetAwsFIPSEndpoint() int {
	if cfg.UseAwsFIPSEndpoint {
		return 1
	} else {
		return 0
	}
}
