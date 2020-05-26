package cloudmap

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	corev1 "k8s.io/api/core/v1"
)

// serviceSummary contains summary of cloudMap service information.
type serviceSummary struct {
	serviceID               string
	healthCheckCustomConfig *servicediscovery.HealthCheckCustomConfig
}

// serviceSubset represents a subset of cloudMap service
// see virtualNodeServiceSubset for more information.
type serviceSubset interface {
	// SubsetID returns a unique identifier for a specific cloudMap service's subset.
	SubsetID() string
	// Contains checks whether specified instance belongs to this subset.
	Contains(instanceID string, attrs instanceAttributes) bool
}

// serviceSubsetID represents ID for specific subset within specific cloudMap service
type serviceSubsetID struct {
	serviceID string
	subsetID  string
}

// instanceAttributes is the attributes for a cloudMap service instance.
type instanceAttributes map[string]string

// instanceInfo contains information for a cloudMap instance.
type instanceInfo struct {
	// instance's attribute
	attrs instanceAttributes
	// instance's corresponding k8s pod
	pod *corev1.Pod
}

type nodeAttributes struct {
	region           string
	availabilityZone string
}

var _ serviceSubset = &virtualNodeServiceSubset{}

// virtualNodeServiceSubset presents a subset of cloudMap service that should be managed by specific virtualNode.
type virtualNodeServiceSubset struct {
	ms *appmesh.Mesh
	vn *appmesh.VirtualNode
}

func (s *virtualNodeServiceSubset) SubsetID() string {
	return fmt.Sprintf("%s/%s", aws.StringValue(s.ms.Spec.AWSName), aws.StringValue(s.vn.Spec.AWSName))
}

func (s *virtualNodeServiceSubset) Contains(instanceID string, attrs instanceAttributes) bool {
	return attrs[attrAppMeshMesh] == aws.StringValue(s.ms.Spec.AWSName) && attrs[attrAppMeshVirtualNode] == aws.StringValue(s.vn.Spec.AWSName)
}
