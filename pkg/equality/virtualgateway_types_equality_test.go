package equality

import (
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCompareOptionForVirtualGatewayHealthCheckPolicy(t *testing.T) {
	tests := []struct {
		name       string
		argLeft    *appmeshsdk.VirtualGatewayHealthCheckPolicy
		argRight   *appmeshsdk.VirtualGatewayHealthCheckPolicy
		wantEquals bool
	}{
		{
			name: "when port equals",
			argLeft: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(80),
			},
			argRight: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(80),
			},
			wantEquals: true,
		},
		{
			name: "when port differs",
			argLeft: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(80),
			},
			argRight: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(443),
			},
			wantEquals: false,
		},
		{
			name: "when left hand port is nil",
			argLeft: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: nil,
			},
			argRight: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(443),
			},
			wantEquals: true,
		},
		{
			name: "when left hand port is nil but other fields differs",
			argLeft: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: nil,
				Path: aws.String("/p1"),
			},
			argRight: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				Port: aws.Int64(443),
				Path: aws.String("/p2"),
			},
			wantEquals: false,
		},
		{
			name:       "nil left hand arg",
			argLeft:    nil,
			argRight:   &appmeshsdk.VirtualGatewayHealthCheckPolicy{},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CompareOptionForVirtualGatewayHealthCheckPolicy()
			gotEquals := cmp.Equal(tt.argLeft, tt.argRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}

func TestCompareOptionForVirtualGatewaySpec(t *testing.T) {
	tests := []struct {
		name       string
		argLeft    *appmeshsdk.VirtualGatewaySpec
		argRight   *appmeshsdk.VirtualGatewaySpec
		wantEquals bool
	}{
		{
			name: "when left hand arg have nil listeners, while right hand arg have empty listeners",
			argLeft: &appmeshsdk.VirtualGatewaySpec{
				Listeners: nil,
			},
			argRight: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{},
			},
			wantEquals: true,
		},
		{
			name: "when healthCheck is equal except the port field",
			argLeft: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							Path: aws.String("p1"),
							Port: nil,
						},
					},
				},
			},
			argRight: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							Path: aws.String("p1"),
							Port: aws.Int64(80),
						},
					},
				},
			},
			wantEquals: true,
		},
		{
			name: "when healthcheck differs",
			argLeft: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							Path: aws.String("p1"),
						},
					},
				},
			},
			argRight: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							Path: aws.String("p2"),
						},
					},
				},
			},
			wantEquals: false,
		},
		{
			name:       "nil left hand arg",
			argLeft:    nil,
			argRight:   &appmeshsdk.VirtualGatewaySpec{},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CompareOptionForVirtualGatewaySpec()
			gotEquals := cmp.Equal(tt.argLeft, tt.argRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}
