package throttle

import (
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"golang.org/x/time/rate"
	"regexp"
)

// NewDefaultServiceOperationsThrottleConfig returns a ServiceOperationsThrottleConfig with default settings.
func NewDefaultServiceOperationsThrottleConfig() *ServiceOperationsThrottleConfig {
	return &ServiceOperationsThrottleConfig{
		value: map[string][]throttleConfig{
			appmesh.ServiceID: {
				{
					operationPtn: regexp.MustCompile("^Describe|List"),
					r:            rate.Limit(40),
					burst:        5,
				},
				{
					operationPtn: regexp.MustCompile("^Create|Update|Delete"),
					r:            rate.Limit(8),
					burst:        5,
				},
			},
			servicediscovery.ServiceID: {
				{
					operationPtn: regexp.MustCompile("^ListNamespaces"),
					r:            rate.Limit(1),
					burst:        8,
				},
				{
					operationPtn: regexp.MustCompile("^ListServices"),
					r:            rate.Limit(1),
					burst:        8,
				},
				{
					operationPtn: regexp.MustCompile("^CreateService"),
					r:            rate.Limit(8),
					burst:        80,
				},
				{
					operationPtn: regexp.MustCompile("^ListInstances"),
					r:            rate.Limit(40),
					burst:        400,
				},
				{
					operationPtn: regexp.MustCompile("^RegisterInstance|DeregisterInstance"),
					r:            rate.Limit(4),
					burst:        80,
				},
			},
		},
	}
}
