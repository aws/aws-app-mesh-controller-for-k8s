package equality

import (
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCompareOptionForHealthCheckPolicy(t *testing.T) {
	tests := []struct {
		name       string
		argLeft    *appmeshsdk.HealthCheckPolicy
		argRight   *appmeshsdk.HealthCheckPolicy
		wantEquals bool
	}{
		{
			name: "when port equals",
			argLeft: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(80),
			},
			argRight: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(80),
			},
			wantEquals: true,
		},
		{
			name: "when port differs",
			argLeft: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(80),
			},
			argRight: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(443),
			},
			wantEquals: false,
		},
		{
			name: "when left hand port is nil",
			argLeft: &appmeshsdk.HealthCheckPolicy{
				Port: nil,
			},
			argRight: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(443),
			},
			wantEquals: true,
		},
		{
			name: "when left hand port is nil but other fields differs",
			argLeft: &appmeshsdk.HealthCheckPolicy{
				Port: nil,
				Path: aws.String("/p1"),
			},
			argRight: &appmeshsdk.HealthCheckPolicy{
				Port: aws.Int64(443),
				Path: aws.String("/p2"),
			},
			wantEquals: false,
		},
		{
			name:       "nil left hand arg",
			argLeft:    nil,
			argRight:   &appmeshsdk.HealthCheckPolicy{},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CompareOptionForHealthCheckPolicy()
			gotEquals := cmp.Equal(tt.argLeft, tt.argRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}

func TestCompareOptionForVirtualNodeSpec(t *testing.T) {
	tests := []struct {
		name       string
		argLeft    *appmeshsdk.VirtualNodeSpec
		argRight   *appmeshsdk.VirtualNodeSpec
		wantEquals bool
	}{
		{
			name: "when left hand arg have nil backends, while right hand arg have empty backends",
			argLeft: &appmeshsdk.VirtualNodeSpec{
				Backends: nil,
			},
			argRight: &appmeshsdk.VirtualNodeSpec{
				Backends: []*appmeshsdk.Backend{},
			},
			wantEquals: true,
		},
		{
			name: "when left hand arg have nil backends, while right hand arg have empty backends, and healthCheck equals",
			argLeft: &appmeshsdk.VirtualNodeSpec{
				Backends: nil,
				Listeners: []*appmeshsdk.Listener{
					{
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							Path: aws.String("p1"),
							Port: nil,
						},
					},
				},
			},
			argRight: &appmeshsdk.VirtualNodeSpec{
				Backends: []*appmeshsdk.Backend{},
				Listeners: []*appmeshsdk.Listener{
					{
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							Path: aws.String("p1"),
							Port: aws.Int64(80),
						},
					},
				},
			},
			wantEquals: true,
		},
		{
			name: "when left hand arg have nil backends, while right hand arg have empty backends, but other fields differs",
			argLeft: &appmeshsdk.VirtualNodeSpec{
				Backends: nil,
				Listeners: []*appmeshsdk.Listener{
					{
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							Path: aws.String("p1"),
						},
					},
				},
			},
			argRight: &appmeshsdk.VirtualNodeSpec{
				Backends: []*appmeshsdk.Backend{},
				Listeners: []*appmeshsdk.Listener{
					{
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
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
			argRight:   &appmeshsdk.VirtualNodeSpec{},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CompareOptionForVirtualNodeSpec()
			gotEquals := cmp.Equal(tt.argLeft, tt.argRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}
