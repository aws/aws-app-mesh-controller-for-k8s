/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayRouteVirtualService refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GatewayRouteVirtualService struct {
	// The virtual service reference to associate with the gateway route virtual service target.
	VirtualServiceRef VirtualServiceReference `json:"virtualServiceRef"`
}

// GatewayRouteTarget refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GatewayRouteTarget struct {
	// The virtual service to associate with the gateway route target.
	VirtualService GatewayRouteVirtualService `json:"virtualService"`
}

// GRPCGatewayRouteMatch refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GRPCGatewayRouteMatch struct {
	// The fully qualified domain name for the service to match from the request.
	// +optional
	ServiceName *string `json:"serviceName,omitempty"`
}

// GRPCGatewayRouteAction refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GRPCGatewayRouteAction struct {
	// An object that represents the target that traffic is routed to when a request matches the route.
	Target GatewayRouteTarget `json:"target"`
}

// GRPCGatewayRoute refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GRPCGatewayRoute struct {
	// An object that represents the criteria for determining a request match.
	Match GRPCGatewayRouteMatch `json:"match"`
	// An object that represents the action to take if a match is determined.
	Action GRPCGatewayRouteAction `json:"action"`
}

// HTTPGatewayRouteMatch refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type HTTPGatewayRouteMatch struct {
	// Specifies the path to match requests with
	Prefix string `json:"prefix"`
}

// HTTPGatewayRouteAction refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type HTTPGatewayRouteAction struct {
	// An object that represents the target that traffic is routed to when a request matches the route.
	Target GatewayRouteTarget `json:"target"`
}

// HTTPGatewayRoute refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type HTTPGatewayRoute struct {
	// An object that represents the criteria for determining a request match.
	Match HTTPGatewayRouteMatch `json:"match"`
	// An object that represents the action to take if a match is determined.
	Action HTTPGatewayRouteAction `json:"action"`
}

// GatewayRouteSpec defines the desired state of GatewayRoute
// refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type GatewayRouteSpec struct {
	// AWSName is the AppMesh GatewayRoute object's name.
	// If unspecified or empty, it defaults to be "${name}_${namespace}" of k8s GatewayRoute
	// +optional
	AWSName *string `json:"awsName,omitempty"`
	// An object that represents the specification of a gRPC gatewayRoute.
	// +optional
	GRPCRoute *GRPCGatewayRoute `json:"grpcRoute,omitempty"`
	// An object that represents the specification of an HTTP gatewayRoute.
	// +optional
	HTTPRoute *HTTPGatewayRoute `json:"httpRoute,omitempty"`
	// An object that represents the specification of an HTTP/2 gatewayRoute.
	// +optional
	HTTP2Route *HTTPGatewayRoute `json:"http2Route,omitempty"`
	// A reference to k8s VirtualGateway CR that this GatewayRoute belongs to.
	// The admission controller populates it using VirtualGateway's selector, and prevents users from setting this field.
	//
	// Populated by the system.
	// Read-only.
	// +optional
	VirtualGatewayRef *VirtualGatewayReference `json:"virtualGatewayRef,omitempty"`
	// A reference to k8s Mesh CR that this GatewayRoute belongs to.
	// The admission controller populates it using Meshes's selector, and prevents users from setting this field.
	//
	// Populated by the system.
	// Read-only.
	// +optional
	MeshRef *MeshReference `json:"meshRef,omitempty"`
}

type GatewayRouteConditionType string

const (
	// GatewayRouteActive is True when the AppMesh GatewayRoute has been created or found via the API
	GatewayRouteActive GatewayRouteConditionType = "GatewayRouteActive"
)

type GatewayRouteCondition struct {
	// Type of GatewayRoute condition.
	Type GatewayRouteConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason *string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message *string `json:"message,omitempty"`
}

// GatewayRouteStatus defines the observed state of GatewayRoute
type GatewayRouteStatus struct {
	// GatewayRouteARNs is a map of AppMesh GatewayRoute objects' Amazon Resource Names, indexed by gatewayRoute name.
	// +optional
	GatewayRouteARN *string `json:"gatewayRouteARN,omitempty"`
	// The current GatewayRoute status.
	// +optional
	Conditions []GatewayRouteCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayRoute is the Schema for the gatewayroutes API
type GatewayRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewayRouteSpec   `json:"spec,omitempty"`
	Status GatewayRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayRouteList contains a list of GatewayRoute
type GatewayRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatewayRoute{}, &GatewayRouteList{})
}