package equality

import (
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func CompareOptionForHealthCheckPolicy() cmp.Option {
	return IgnoreLeftHandUnset(appmeshsdk.HealthCheckPolicy{}, "Port")
}

func CompareOptionForVirtualNodeSpec() cmp.Option {
	return cmp.Options{
		cmpopts.EquateEmpty(),
		CompareOptionForHealthCheckPolicy(),
	}
}
