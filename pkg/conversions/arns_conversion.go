package conversions

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/conversion"
	"regexp"
)

func Convert_CRD_VirtualNodeARN_To_SDK_VirtualNodeName(vnARN *string, vnName *string, scope conversion.Scope) error {
	parsedARN, err := arn.Parse(*vnARN)
	if err != nil {
		return errors.Wrapf(err, "invalid arn")
	}
	_, resourceType, resourceName, err := parseAppMeshARNResource(parsedARN.Resource)
	if err != nil {
		return err
	}
	if resourceType != "virtualNode" {
		return errors.Errorf("expects %v ARN, got %v", "virtualNode", resourceType)
	}
	*vnName = resourceName
	return nil
}

func Convert_CRD_VirtualServiceARN_To_SDK_VirtualServiceName(vsARN *string, vsName *string, scope conversion.Scope) error {
	parsedARN, err := arn.Parse(*vsARN)
	if err != nil {
		return errors.Wrapf(err, "invalid arn")
	}
	_, resourceType, resourceName, err := parseAppMeshARNResource(parsedARN.Resource)
	if err != nil {
		return err
	}
	if resourceType != "virtualService" {
		return errors.Errorf("expects %v ARN, got %v", "virtualService", resourceType)
	}
	*vsName = resourceName
	return nil
}

func Convert_CRD_VirtualRouterARN_To_SDK_VirtualRouterName(vrARN *string, vrName *string, scope conversion.Scope) error {
	parsedARN, err := arn.Parse(*vrARN)
	if err != nil {
		return errors.Wrapf(err, "invalid arn")
	}
	_, resourceType, resourceName, err := parseAppMeshARNResource(parsedARN.Resource)
	if err != nil {
		return err
	}
	if resourceType != "virtualRouter" {
		return errors.Errorf("expects %v ARN, got %v", "virtualRouter", resourceType)
	}
	*vrName = resourceName
	return nil
}

var appMeshARNResourcePattern = regexp.MustCompile("^mesh/([^/]+)/([^/]+)/([^/]+)$")

// parseAppMeshARNResource parses the resource part for an appmesh resource's ARN
func parseAppMeshARNResource(arnResource string) (augmentedMeshName string, resourceType string, resourceName string, err error) {
	subExps := appMeshARNResourcePattern.FindStringSubmatch(arnResource)
	if len(subExps) != 4 {
		return "", "", "", errors.Errorf("invalid resource in appMesh ARN: %v", arnResource)
	}
	return subExps[1], subExps[2], subExps[3], nil
}
