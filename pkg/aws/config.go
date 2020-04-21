package aws

import "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"

type CloudOptions struct {
	Region               string
	AWSAPIThrottleConfig *throttle.ServiceOperationsThrottleConfig
}
