package appmesh

import (
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_gatewayRouteValidator_validateGRPCGatewayRouteSpec(t *testing.T) {
	tests := []struct {
		name    string
		grSpec  appmesh.GRPCGatewayRoute
		wantErr error
	}{
		{
			name: "ValidCase: No Match filter for one of the metadata",
			grSpec: appmesh.GRPCGatewayRoute{
				Match: appmesh.GRPCGatewayRouteMatch{
					ServiceName: aws.String("color.ColorService"),
					Metadata: []appmesh.GRPCGatewayRouteMetadata{
						{
							Name: aws.String("client"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Exact: aws.String("mobile"),
							},
						},
						{
							Name: aws.String("service_type"),
						},
					},
				},
				Action: appmesh.GRPCGatewayRouteAction{},
			},
			wantErr: nil,
		},
		{
			name: "Invalid MatchRange for metadata",
			grSpec: appmesh.GRPCGatewayRoute{
				Match: appmesh.GRPCGatewayRouteMatch{
					ServiceName: aws.String("user.ProfileService"),
					Metadata: []appmesh.GRPCGatewayRouteMetadata{
						{
							Name: aws.String("userId"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Range: &appmesh.MatchRange{
									Start: 30,
									End:   20,
								},
							},
						},
					},
				},
				Action: appmesh.GRPCGatewayRouteAction{},
			},
			wantErr: errors.New("Invalid Match Range specified, end has to be greater than start"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGRPCGatewayRouteSpec(&tt.grSpec)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func Test_gatewayRouteValidator_validateQueryParamsIfAny(t *testing.T) {
	tests := []struct {
		name    string
		currGR  *appmesh.HTTPGatewayRouteMatch
		wantErr error
	}{
		{
			name: "Valid case",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				QueryParameters: []appmesh.HTTPQueryParameters{
					{
						Name: aws.String("user"),
						Match: &appmesh.QueryMatchMethod{
							Exact: aws.String("Test"),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "No matching filters for single query parameter",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				QueryParameters: []appmesh.HTTPQueryParameters{
					{
						Name:  aws.String("user"),
						Match: &appmesh.QueryMatchMethod{},
					},
				},
			},
			wantErr: errors.New("Missing Match criteria for one or more exact query match block, don't specify match block if you dont need it"),
		},
		{
			name: "QueryParams with only name and no match block specified",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				QueryParameters: []appmesh.HTTPQueryParameters{
					{
						Name: aws.String("user"),
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryParametersIfAny(tt.currGR.QueryParameters)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validateHeadersIfAny(t *testing.T) {
	tests := []struct {
		name    string
		currGR  *appmesh.HTTPGatewayRouteMatch
		wantErr error
	}{
		{
			name: "Multiple matching filters for single header",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				Headers: []appmesh.HTTPGatewayRouteHeader{
					{
						Name: "user",
						Match: &appmesh.HeaderMatchMethod{
							Exact:  aws.String("Test"),
							Prefix: aws.String("App"),
						},
						Invert: aws.Bool(false),
					},
				},
			},
			wantErr: errors.New("Too many Match Filters specified, only 1 allowed per header"),
		},
		{
			name: "No matching filters for single header",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				Headers: []appmesh.HTTPGatewayRouteHeader{
					{
						Name:  "user",
						Match: &appmesh.HeaderMatchMethod{},
					},
				},
			},
			wantErr: errors.New("Missing Match criteria for one or more header match block, don't specify match block if you dont need it"),
		},
		{
			name: "Header with only name and no match block specified",
			currGR: &appmesh.HTTPGatewayRouteMatch{
				Headers: []appmesh.HTTPGatewayRouteHeader{
					{
						Name: "user",
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPHeadersIfAny(tt.currGR.Headers)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validateMetadataIfAny(t *testing.T) {
	tests := []struct {
		name    string
		currGR  *appmesh.GRPCGatewayRouteMatch
		wantErr error
	}{
		{
			name: "Multiple Match Filters for single metadata",
			currGR: &appmesh.GRPCGatewayRouteMatch{
				Metadata: []appmesh.GRPCGatewayRouteMetadata{
					{
						Name: aws.String("service_name"),
						Match: &appmesh.GRPCRouteMetadataMatchMethod{
							Exact:  aws.String("checkout"),
							Suffix: aws.String("/item"),
						},
					},
				},
			},
			wantErr: errors.New("Too many Match Filters specified, only 1 allowed per metadata"),
		},
		{
			name: "Invalid Match Range",
			currGR: &appmesh.GRPCGatewayRouteMatch{
				Metadata: []appmesh.GRPCGatewayRouteMetadata{
					{
						Name: aws.String("service_name"),
						Match: &appmesh.GRPCRouteMetadataMatchMethod{
							Range: &appmesh.MatchRange{
								Start: 4,
								End:   2,
							},
						},
					},
				},
			},
			wantErr: errors.New("Invalid Match Range specified, end has to be greater than start"),
		},
		{
			name: "Valid Case",
			currGR: &appmesh.GRPCGatewayRouteMatch{
				Metadata: []appmesh.GRPCGatewayRouteMetadata{
					{
						Name: aws.String("service_name"),
						Match: &appmesh.GRPCRouteMetadataMatchMethod{
							Range: &appmesh.MatchRange{
								Start: 4,
								End:   8,
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetadataIfAny(tt.currGR.Metadata)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validatePathForHTTPGatewayRoute(t *testing.T) {
	tests := []struct {
		name    string
		path    *appmesh.HTTPPathMatch
		wantErr error
	}{
		{
			name: "Only Exact path specified",
			path: &appmesh.HTTPPathMatch{
				Exact: aws.String("/paths/green"),
			},
			wantErr: nil,
		},
		{
			name: "Only Regex specified",
			path: &appmesh.HTTPPathMatch{
				Regex: aws.String("/pathss/green.*test"),
			},
			wantErr: nil,
		},
		{
			name: "Both Exact and Regex specified",
			path: &appmesh.HTTPPathMatch{
				Exact: aws.String("/paths/green"),
				Regex: aws.String("/pathss/green.*test"),
			},
			wantErr: errors.New("Both exact and regex for path are not allowed. Only one must be specified"),
		},
		{
			name:    "Neither Exact nor Regex specified",
			path:    &appmesh.HTTPPathMatch{},
			wantErr: errors.New("Either exact or regex for path must be specified"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathForGatewayRoute(tt.path)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func Test_gatewayRouteValidator_validateHTTPRouteRewrite(t *testing.T) {
	tests := []struct {
		name    string
		match   appmesh.HTTPGatewayRouteMatch
		rewrite appmesh.HTTPGatewayRouteRewrite
		wantErr error
	}{
		{
			name: "Missing Rewrite filters",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix:   nil,
				Path:     nil,
				Hostname: nil,
			},
			wantErr: errors.New("Either prefix, path or hostname for rewrite must be specified"),
		},
		{
			name: "Both Prefix and Path Rewrite specified",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					Value: aws.String("test-prefix"),
				},
				Path: &appmesh.GatewayRoutePathRewrite{
					Exact: aws.String("test-prefix.com/payment"),
				},
			},
			wantErr: errors.New("Both prefix and path for rewrites cannot be specified. Only 1 allowed"),
		},
		{
			name: "Incorrect Prefix Rewrite format",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					Value: aws.String("/test"),
				},
			},
			wantErr: errors.New("New Rewrite Prefix must start and end with '/'"),
		},
		{
			name: "Incorrect Prefix Match format - case2",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					Value: aws.String("/test/"),
				},
			},
			wantErr: errors.New("Prefix to be matched on must start with '/'"),
		},
		{
			name: "Incorrect Prefix Rewrite format - case2",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					Value: aws.String("test/"),
				},
			},
			wantErr: errors.New("New Rewrite Prefix must start and end with '/'"),
		},
		{
			name: "Both DefaultPrefix and Prefix for Rewrite specified",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					DefaultPrefix: aws.String("DISABLED"),
					Value:         aws.String("/test/"),
				},
			},
			wantErr: errors.New("Cannot specify both defaultPrefix and prefix for rewrite, only 1 allowed"),
		},
		{
			name: "ValidCase: Only DefaultPrefix specified",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Prefix: &appmesh.GatewayRoutePrefixRewrite{
					DefaultPrefix: aws.String("DISABLED"),
				},
			},
			wantErr: nil,
		},
		{
			name: "Only DefaultTargetHostname specified",
			match: appmesh.HTTPGatewayRouteMatch{
				Prefix: aws.String("/red/"),
			},
			rewrite: appmesh.HTTPGatewayRouteRewrite{
				Hostname: &appmesh.GatewayRouteHostnameRewrite{
					DefaultTargetHostname: aws.String("DISABLED"),
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPRouteRewrite(&tt.rewrite, tt.match)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validateHostnameAndServicename(t *testing.T) {
	tests := []struct {
		name      string
		currMatch *appmesh.GRPCGatewayRouteMatch
		wantErr   error
	}{
		{
			name:      "Hostname and Servicename both nil",
			currMatch: &appmesh.GRPCGatewayRouteMatch{},
			wantErr:   errors.New("Either servicename or hostname must be specified"),
		},
		{
			name: "Invalid Hostname with both Exact and Suffix specified",
			currMatch: &appmesh.GRPCGatewayRouteMatch{
				Hostname: &appmesh.GatewayRouteHostnameMatch{
					Exact:  aws.String("www.github.com"),
					Suffix: aws.String(".github.com"),
				},
			},
			wantErr: errors.New("Both exact and suffix match for hostname are not allowed. Only one must be specified"),
		},
		{
			name: "Invalid Hostname with neither Exact nor Suffix specified",
			currMatch: &appmesh.GRPCGatewayRouteMatch{
				Hostname: &appmesh.GatewayRouteHostnameMatch{
					Exact:  nil,
					Suffix: nil,
				},
			},
			wantErr: errors.New("Either exact or suffix match for hostname must be specified"),
		},
		{
			name: "Valid Hostname Case",
			currMatch: &appmesh.GRPCGatewayRouteMatch{
				Hostname: &appmesh.GatewayRouteHostnameMatch{
					Exact: aws.String("www.github.com"),
				},
			},
			wantErr: nil,
		},
		{
			name: "Valid Servicename Case",
			currMatch: &appmesh.GRPCGatewayRouteMatch{
				ServiceName: aws.String("test-service"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostnameAndServicename(*tt.currMatch)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validatePrefixOrPathOrHostnameOrServicename(t *testing.T) {
	tests := []struct {
		name    string
		currGR  *appmesh.GatewayRoute
		wantErr error
	}{
		{
			name: "GRPCGateway Route SevericName specified",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							ServiceName: aws.String("my-service"),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "GRPCGateway Route with Valid Hostname specified",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							Hostname: &appmesh.GatewayRouteHostnameMatch{
								Exact: aws.String("example.com"),
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "GRPCGateway Route with Missing Servicename and Hostname",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{},
					},
				},
			},
			wantErr: errors.New("Either servicename or hostname must be specified"),
		},
		{
			name: "HTTPGateway Route with Missing Prefix, Path and Hostname",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{},
					},
				},
			},
			wantErr: errors.New("Either prefix or path or hostname must be specified"),
		},
		{
			name: "HTTPGateway Route with Prefix and Path specified",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("/payment"),
							Path:   &appmesh.HTTPPathMatch{Exact: aws.String("/payment/items")},
						},
					},
				},
			},
			wantErr: errors.New("Both prefix and path cannot be specified. Only 1 allowed"),
		},
		{
			name: "HTTPGateway Route with Prefix or Path specified with no Hostname",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("/payment/"),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "HTTPGateway Route with Prefix or Path specified with Hostname",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Hostname: &appmesh.GatewayRouteHostnameMatch{Exact: aws.String("www.hotels.com")},
							Prefix:   aws.String("/payment/"),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "HTTPGateway Route with valid Prefix",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("/payment/"),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "HTTPGateway with Exact and Suffix both specified for Hostname",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Hostname: &appmesh.GatewayRouteHostnameMatch{
								Exact:  aws.String("www.hotels.com"),
								Suffix: aws.String("/payment"),
							},
						},
					},
				},
			},
			wantErr: errors.New("Both exact and suffix match for hostname are not allowed. Only one must be specified"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			spec := tt.currGR.Spec
			if spec.GRPCRoute != nil {
				err = validateHostnameAndServicename(spec.GRPCRoute.Match)
			} else if spec.HTTPRoute != nil {
				err = validatePrefix_Path_HostName(spec.HTTPRoute.Match)
			} else {
				err = validatePrefix_Path_HostName(spec.HTTP2Route.Match)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_gatewayRouteValidator_validateNumOfRoutes(t *testing.T) {
	tests := []struct {
		name        string
		currGR      *appmesh.GatewayRoute
		numOfRoutes int
	}{
		{
			name: "Only HTTPRoute specified",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("/paths"),
						},
					},
				},
			},
			numOfRoutes: 1,
		},
		{
			name: "HTTPRoute and GRPCRoute specified",
			currGR: &appmesh.GatewayRoute{
				Spec: appmesh.GatewayRouteSpec{
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("/paths"),
						},
					},
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							ServiceName: aws.String("payment"),
						},
					},
				},
			},
			numOfRoutes: 2,
		},
		{
			name:        "No route specified",
			currGR:      &appmesh.GatewayRoute{},
			numOfRoutes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numOfRoutes := getNumberOfRouteTypes(tt.currGR.Spec)
			assert.Equal(t, tt.numOfRoutes, numOfRoutes)
		})
	}
}

func Test_gatewayRouteValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		newGR *appmesh.GatewayRoute
		oldGR *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "GatewayRoute immutable fields didn't change",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "GatewayRoute field awsName changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.awsName"),
		},
		{
			name: "GatewayRoute field meshRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.meshRef"),
		},
		{
			name: "GatewayRoute field virtualGatewayRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "another-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.virtualGatewayRef"),
		},
		{
			name: "GatewayRoute fields awsName, meshRef and virtualGatewayRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "another-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.awsName,spec.meshRef,spec.virtualGatewayRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &gatewayRouteValidator{}
			err := v.enforceFieldsImmutability(tt.args.newGR, tt.args.oldGR)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
