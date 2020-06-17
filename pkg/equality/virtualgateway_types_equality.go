// +build preview

package equality

import (
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func CompareOptionForVirtualGatewayHealthCheckPolicy() cmp.Option {
	return IgnoreLeftHandUnset(appmeshsdk.VirtualGatewayHealthCheckPolicy{}, "Port")
}

func CompareOptionForVirtualGatewaySpec() cmp.Option {
	return cmp.Options{
		cmpopts.EquateEmpty(),
		CompareOptionForVirtualGatewayHealthCheckPolicy(),
	}
}
