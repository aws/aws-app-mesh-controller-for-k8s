package aws

import (
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestCloudConfig(t *testing.T) {
	cloudConfig := CloudConfig{
		Region:         "",
		AccountID:      "",
		ThrottleConfig: nil,
	}
	setupLog := ctrl.Log.WithName("setup")

	type args struct {
		accountId string
	}
	tests := []struct {
		name          string
		accountId     string
		wantAccountID string
	}{
		{
			name:          "normal case",
			accountId:     "123456789012",
			wantAccountID: "123456789012",
		},
		{
			name:          "properID with characters at end",
			accountId:     "123456789012abc",
			wantAccountID: "123456789012abc",
		},
		{
			name:          "Scientific Notation case formatted as expected to handle",
			accountId:     "1.23456789012e+11",
			wantAccountID: "123456789012",
		},
		{
			name:          "Scientific Notation case extra letters at end",
			accountId:     "1.23456789012e+1123",
			wantAccountID: "1.23456789012e+1123",
		},
		{
			name:          "Scientific Notation case: Too many digits",
			accountId:     "1.234567890123e+11",
			wantAccountID: "1.234567890123e+11",
		},
		{
			name:          "Scientific Notation case: Wrong ending",
			accountId:     "1.234567890123e+12",
			wantAccountID: "1.234567890123e+12",
		},
		{
			name:          "Scientific Notation case: Too few digits",
			accountId:     "1.234567890e+11",
			wantAccountID: "1.234567890e+11",
		},
		{
			name:          "No AccountID Provided",
			accountId:     "",
			wantAccountID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloudConfig.AccountID = tt.accountId
			cloudConfig.HandleAccountID(setupLog)
			assert.Equal(t, tt.wantAccountID, cloudConfig.AccountID)
		})
	}
}

func TestGetAwsDualStackEndpoint(t *testing.T) {
	c := CloudConfig{
		UseAwsDualStackEndpoint: false,
	}
	var tests = []struct {
		name                    string
		useAwsDualStackEndpoint bool
		want                    int
	}{
		{
			name:                    "enabling Aws Service DualStackEndpoint",
			useAwsDualStackEndpoint: true,
			want:                    1,
		},
		{
			name:                    "disabling Aws Service DualStackEndpoint",
			useAwsDualStackEndpoint: false,
			want:                    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.UseAwsDualStackEndpoint = tt.useAwsDualStackEndpoint
			got := c.GetAwsDualStackEndpoint()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAwsFIPSEndpoint(t *testing.T) {
	c := CloudConfig{
		UseAwsFIPSEndpoint: false,
	}
	var tests = []struct {
		name               string
		useAwsFIPSEndpoint bool
		want               int
	}{
		{
			name:               "enabling Aws Service FipsEndpoint",
			useAwsFIPSEndpoint: true,
			want:               1,
		},
		{
			name:               "disabling Aws Service FipsEndpoint",
			useAwsFIPSEndpoint: false,
			want:               0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.UseAwsFIPSEndpoint = tt.useAwsFIPSEndpoint
			got := c.GetAwsFIPSEndpoint()
			assert.Equal(t, tt.want, got)
		})
	}
}
