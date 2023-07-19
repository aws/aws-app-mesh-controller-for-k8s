package appmesh

import (
	"context"
	"reflect"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathValidateAppMeshGatewayRoute = "/validate-appmesh-k8s-aws-v1beta2-gatewayroute"

// NewGatewayRouteValidator returns a validator for GatewayRoute.
func NewGatewayRouteValidator() *gatewayRouteValidator {
	return &gatewayRouteValidator{}
}

var _ webhook.Validator = &gatewayRouteValidator{}

type gatewayRouteValidator struct {
}

func (v *gatewayRouteValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.GatewayRoute{}, nil
}

func validateInternal(spec appmesh.GatewayRouteSpec) error {
	numOfRoutes := getNumberOfRouteTypes(spec)
	if numOfRoutes == 0 {
		return errors.New("Missing GatewayRoute Type in GatewayRoute Spec")
	}

	if numOfRoutes > 1 {
		return errors.New("Found multiple types in GatewayRoute Spec. Only 1 (GRPC/HTTP/HTTP2) allowed")
	}

	if spec.GRPCRoute != nil {
		return validateGRPCGatewayRouteSpec(spec.GRPCRoute)
	}
	if spec.HTTPRoute != nil {
		return validateHTTPRouteSpec(spec.HTTPRoute)
	}
	// spec is for HTTP2Route
	return validateHTTPRouteSpec(spec.HTTP2Route)
}

func (v *gatewayRouteValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	currGR := obj.(*appmesh.GatewayRoute)
	spec := currGR.Spec
	return validateInternal(spec)
}

func getNumberOfRouteTypes(spec appmesh.GatewayRouteSpec) int {
	numOfRoutes := 0
	if spec.GRPCRoute != nil {
		numOfRoutes++
	}
	if spec.HTTPRoute != nil {
		numOfRoutes++
	}
	if spec.HTTP2Route != nil {
		numOfRoutes++
	}
	return numOfRoutes
}

func validateQueryParametersIfAny(queryParams []appmesh.HTTPQueryParameters) error {
	for _, queryParam := range queryParams {
		if queryParam.Match == nil {
			continue
		}
		if queryParam.Match.Exact == nil {
			return errors.New("Missing Match criteria for one or more exact query match block, don't specify match block if you dont need it")
		}
	}
	return nil
}

func validateHTTPHeadersIfAny(headers []appmesh.HTTPGatewayRouteHeader) error {
	for _, header := range headers {
		if header.Match == nil {
			continue
		}
		numOfMatchFilters := getNumOfHeaderMatchFilters(header.Match)
		if numOfMatchFilters > 1 {
			return errors.New("Too many Match Filters specified, only 1 allowed per header")
		}
		if numOfMatchFilters == 0 {
			return errors.New("Missing Match criteria for one or more header match block, don't specify match block if you dont need it")
		}
		if header.Match.Range != nil {
			return validateMatchRange(header.Match.Range)
		}
	}
	return nil
}

func validateMetadataIfAny(metadataList []appmesh.GRPCGatewayRouteMetadata) error {
	for _, metadata := range metadataList {
		if metadata.Match == nil {
			continue
		}
		numOfMatchFilters := getNumOfMetadataMatchFilters(metadata.Match)
		if numOfMatchFilters > 1 {
			return errors.New("Too many Match Filters specified, only 1 allowed per metadata")
		}
		if metadata.Match.Range != nil {
			return validateMatchRange(metadata.Match.Range)
		}
	}
	return nil
}

func validateMatchRange(matchRange *appmesh.MatchRange) error {
	if matchRange.Start >= matchRange.End {
		return errors.New("Invalid Match Range specified, end has to be greater than start")
	}
	return nil
}

func getNumOfHeaderMatchFilters(match *appmesh.HeaderMatchMethod) int {
	numOfFilter := 0
	if match.Exact != nil {
		numOfFilter++
	}
	if match.Prefix != nil {
		numOfFilter++
	}
	if match.Regex != nil {
		numOfFilter++
	}
	if match.Suffix != nil {
		numOfFilter++
	}
	if match.Range != nil {
		numOfFilter++
	}
	return numOfFilter
}

func getNumOfMetadataMatchFilters(match *appmesh.GRPCRouteMetadataMatchMethod) int {
	numOfFilter := 0
	if match.Exact != nil {
		numOfFilter++
	}
	if match.Prefix != nil {
		numOfFilter++
	}
	if match.Regex != nil {
		numOfFilter++
	}
	if match.Suffix != nil {
		numOfFilter++
	}
	if match.Range != nil {
		numOfFilter++
	}
	return numOfFilter
}

func (v *gatewayRouteValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	newGR := obj.(*appmesh.GatewayRoute)
	oldGR := oldObj.(*appmesh.GatewayRoute)
	if err := v.enforceFieldsImmutability(newGR, oldGR); err != nil {
		return err
	}
	return validateInternal(newGR.Spec)
}

func validateHTTPRouteSpec(currRoute *appmesh.HTTPGatewayRoute) error {
	if err := validateHTTPRouteMatch(currRoute.Match); err != nil {
		return err
	}

	if currRoute.Action != (appmesh.HTTPGatewayRouteAction{}) && currRoute.Action.Rewrite != nil {
		if err := validateHTTPRouteRewrite(currRoute.Action.Rewrite, currRoute.Match); err != nil {
			return err
		}
	}
	return nil
}

func validateHTTPRouteRewrite(rewrite *appmesh.HTTPGatewayRouteRewrite, match appmesh.HTTPGatewayRouteMatch) error {
	if rewrite.Prefix == nil && rewrite.Path == nil && rewrite.Hostname == nil {
		return errors.New("Either prefix, path or hostname for rewrite must be specified")
	}
	if rewrite.Prefix != nil && rewrite.Path != nil {
		return errors.New("Both prefix and path for rewrites cannot be specified. Only 1 allowed")
	}
	if rewrite.Prefix != nil {
		if match.Prefix != nil && !strings.HasPrefix(*match.Prefix, "/") {
			return errors.New("Prefix to be matched on must start with '/'")
		}
		return validatePrefixRewrite(rewrite.Prefix)
	}
	return nil
}

func validatePrefixRewrite(prefixRewrite *appmesh.GatewayRoutePrefixRewrite) error {
	if prefixRewrite.DefaultPrefix != nil && prefixRewrite.Value != nil {
		return errors.New("Cannot specify both defaultPrefix and prefix for rewrite, only 1 allowed")
	}
	if prefixRewrite.Value != nil {
		if !strings.HasSuffix(*prefixRewrite.Value, "/") || !strings.HasPrefix(*prefixRewrite.Value, "/") {
			return errors.New("New Rewrite Prefix must start and end with '/'")
		}
	}
	return nil
}

func validateHTTPRouteMatch(match appmesh.HTTPGatewayRouteMatch) error {
	if err := validatePrefix_Path_HostName(match); err != nil {
		return err
	}
	if match.Headers != nil && len(match.Headers) != 0 {
		return validateHTTPHeadersIfAny(match.Headers)
	}
	if match.QueryParameters != nil && len(match.QueryParameters) != 0 {
		return validateQueryParametersIfAny(match.QueryParameters)
	}
	return nil
}

func validateGRPCGatewayRouteSpec(currRoute *appmesh.GRPCGatewayRoute) error {
	if err := validateGRPCGatewayRouteMatch(currRoute.Match); err != nil {
		return err
	}
	return nil
}

func validateGRPCGatewayRouteMatch(currMatch appmesh.GRPCGatewayRouteMatch) error {
	if err := validateHostnameAndServicename(currMatch); err != nil {
		return err
	}

	if currMatch.Metadata != nil && len(currMatch.Metadata) != 0 {
		return validateMetadataIfAny(currMatch.Metadata)
	}
	return nil
}

func validateHostnameAndServicename(currMatch appmesh.GRPCGatewayRouteMatch) error {
	servicename := currMatch.ServiceName
	hostname := currMatch.Hostname
	if servicename == nil && hostname == nil {
		return errors.New("Either servicename or hostname must be specified")
	}
	if servicename == nil {
		if err := validateHostName(hostname); err != nil {
			return err
		}
	}
	return nil
}

func (v *gatewayRouteValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func validatePrefix_Path_HostName(match appmesh.HTTPGatewayRouteMatch) error {
	prefix := match.Prefix
	hostname := match.Hostname
	path := match.Path

	if prefix == nil && path == nil && hostname == nil {
		return errors.New("Either prefix or path or hostname must be specified")
	}

	if prefix != nil && path != nil {
		return errors.New("Both prefix and path cannot be specified. Only 1 allowed")
	}

	// Validate path
	if path != nil {
		if err := validatePathForGatewayRoute(path); err != nil {
			return err
		}
	}

	// Validate hosntname
	if hostname != nil {
		if err := validateHostName(hostname); err != nil {
			return err
		}
	}
	return nil
}

func validateHostName(hostname *appmesh.GatewayRouteHostnameMatch) error {
	exact := hostname.Exact
	suffix := hostname.Suffix

	if exact == nil && suffix == nil {
		return errors.New("Either exact or suffix match for hostname must be specified")
	}

	if exact != nil && suffix != nil {
		return errors.New("Both exact and suffix match for hostname are not allowed. Only one must be specified")
	}

	return nil
}

func validatePathForGatewayRoute(path *appmesh.HTTPPathMatch) error {
	exact := path.Exact
	regex := path.Regex

	if exact == nil && regex == nil {
		return errors.New("Either exact or regex for path must be specified")
	}

	if exact != nil && regex != nil {
		return errors.New("Both exact and regex for path are not allowed. Only one must be specified")
	}

	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *gatewayRouteValidator) enforceFieldsImmutability(newGR *appmesh.GatewayRoute, oldGR *appmesh.GatewayRoute) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(newGR.Spec.AWSName, oldGR.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(newGR.Spec.MeshRef, oldGR.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if !reflect.DeepEqual(newGR.Spec.VirtualGatewayRef, oldGR.Spec.VirtualGatewayRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.virtualGatewayRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "GatewayRoute", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-gatewayroute,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=gatewayroutes,verbs=create;update,versions=v1beta2,name=vgatewayroute.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (v *gatewayRouteValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshGatewayRoute, webhook.ValidatingWebhookForValidator(v))
}
