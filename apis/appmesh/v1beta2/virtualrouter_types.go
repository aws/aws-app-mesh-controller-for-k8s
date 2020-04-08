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

// VirtualRouterListener refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterListener.html
type VirtualRouterListener struct {
	// The port mapping information for the listener.
	PortMapping PortMapping `json:"portMapping"`
}

// WeightedTarget refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_WeightedTarget.html
type WeightedTarget struct {
	// The virtual node to associate with the weighted target.
	VirtualNodeRef VirtualNodeReference `json:"virtualNodeRef"`
	// The relative weight of the weighted target.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight int64 `json:"weight"`
}

type MatchRange struct {
	// The start of the range.
	// +optional
	Start int64 `json:"start"`
	// The end of the range.
	// +optional
	End int64 `json:"end"`
}

// HeaderMatchMethod refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HeaderMatchMethod.html
type HeaderMatchMethod struct {
	// The value sent by the client must match the specified value exactly.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Exact *string `json:"exact,omitempty"`
	// The value sent by the client must begin with the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Prefix *string `json:"prefix,omitempty"`
	// An object that represents the range of values to match on.
	// +optional
	Range *MatchRange `json:"range,omitempty"`
	// The value sent by the client must include the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Regex *string `json:"regex,omitempty"`
	// The value sent by the client must end with the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Suffix *string `json:"suffix,omitempty"`
}

// HttpRouteHeader refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteHeader.html
type HttpRouteHeader struct {
	// A name for the HTTP header in the client request that will be matched on.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=50
	Name string `json:"name"`
	// The HeaderMatchMethod object.
	// +optional
	Match *HeaderMatchMethod `json:"match,omitempty"`
	// Specify True to match anything except the match criteria. The default value is False.
	// +optional
	Invert *bool `json:"invert,omitempty"`
}

// HttpRouteMatch refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteMatch.html
type HttpRouteMatch struct {
	// An object that represents the client request headers to match on.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Headers []HttpRouteHeader `json:"headers,omitempty"`
	// The client request method to match on.
	// +kubebuilder:validation:Enum=CONNECT;DELETE;GET;HEAD;OPTIONS;PATCH;POST;PUT;TRACE
	// +optional
	Method *string `json:"method,omitempty"`
	// Specifies the path to match requests with
	Prefix string `json:"prefix"`
	// The client request scheme to match on
	// +kubebuilder:validation:Enum=http;https
	// +optional
	Scheme *string `json:"scheme,omitempty"`
}

// HttpRouteAction refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteAction.html
type HttpRouteAction struct {
	// An object that represents the targets that traffic is routed to when a request matches the route.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	WeightedTargets []WeightedTarget `json:"weightedTargets"`
}

// +kubebuilder:validation:Enum=server-error;gateway-error;client-error;stream-error
type HttpRetryPolicyEvent string

// +kubebuilder:validation:Enum=connection-error
type TcpRetryPolicyEvent string

// +kubebuilder:validation:Enum=cancelled;deadline-exceeded;internal;resource-exhausted;unavailable
type GrpcRetryPolicyEvent string

// HttpRetryPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRetryPolicy.html
type HttpRetryPolicy struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=25
	// +optional
	HTTPRetryEvents []HttpRetryPolicyEvent `json:"httpRetryEvents,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1
	// +optional
	TCPRetryEvents []TcpRetryPolicyEvent `json:"tcpRetryEvents,omitempty"`
	// The maximum number of retry attempts.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxRetries *int64 `json:"maxRetries,omitempty"`
	// +optional
	PerRetryTimeout *Duration `json:"perRetryTimeout,omitempty"`
}

// HttpRoute refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRoute.html
type HttpRoute struct {
	// An object that represents the criteria for determining a request match.
	Match HttpRouteMatch `json:"match"`
	// An object that represents the action to take if a match is determined.
	Action HttpRouteAction `json:"action"`
	// An object that represents a retry policy.
	// +optional
	RetryPolicy *HttpRetryPolicy `json:"retryPolicy,omitempty"`
}

// TcpRouteAction refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRouteAction.html
type TcpRouteAction struct {
	// An object that represents the targets that traffic is routed to when a request matches the route.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	WeightedTargets []WeightedTarget `json:"weightedTargets"`
}

// TcpRoute refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRoute.html
type TcpRoute struct {
	// The action to take if a match is determined.
	Action TcpRouteAction `json:"action"`
}

type GrpcRouteMetadataMatchMethod struct {
	// The value sent by the client must match the specified value exactly.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Exact *string `json:"exact,omitempty"`
	// The value sent by the client must begin with the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Prefix *string `json:"prefix,omitempty"`
	// An object that represents the range of values to match on
	// +optional
	Range *MatchRange `json:"range,omitempty"`
	// The value sent by the client must include the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Regex *string `json:"regex,omitempty"`
	// The value sent by the client must end with the specified characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Suffix *string `json:"suffix,omitempty"`
}

// GrpcRouteMetadata refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMetadata.html
type GrpcRouteMetadata struct {
	// The name of the route.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=50
	Name string `json:"name"`
	// An object that represents the data to match from the request.
	// +optional
	Match *GrpcRouteMetadataMatchMethod `json:"match,omitempty"`
	// Specify True to match anything except the match criteria. The default value is False.
	// +optional
	Invert *bool `json:"invert,omitempty"`
}

// GrpcRouteMatch refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMatch.html
type GrpcRouteMatch struct {
	// The method name to match from the request. If you specify a name, you must also specify a serviceName.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=50
	// +optional
	MethodName *string `json:"methodName,omitempty"`
	// The fully qualified domain name for the service to match from the request.
	// +optional
	ServiceName *string `json:"serviceName,omitempty"`
	// An object that represents the data to match from the request.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Metadata []GrpcRouteMetadata `json:"metadata,omitempty"`
}

// GrpcRouteAction refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteAction.html
type GrpcRouteAction struct {
	// An object that represents the targets that traffic is routed to when a request matches the route.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	WeightedTargets []WeightedTarget `json:"weightedTargets"`
}

// GrpcRetryPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRetryPolicy.html
type GrpcRetryPolicy struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=5
	// +optional
	GRPCRetryEvents []GrpcRetryPolicyEvent `json:"grpcRetryEvents,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=25
	// +optional
	HTTPRetryEvents []HttpRetryPolicyEvent `json:"httpRetryEvents,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1
	// +optional
	TCPRetryEvents []TcpRetryPolicyEvent `json:"tcpRetryEvents,omitempty"`
	// The maximum number of retry attempts.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxRetries *int64 `json:"maxRetries,omitempty"`
	// +optional
	PerRetryTimeout *Duration `json:"perRetryTimeout,omitempty"`
}

// GrpcRoute refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRoute.html
type GrpcRoute struct {
	// An object that represents the criteria for determining a request match.
	Match GrpcRouteMatch `json:"match"`
	// An object that represents the action to take if a match is determined.
	Action GrpcRouteAction `json:"action"`
	// An object that represents a retry policy.
	// +optional
	RetryPolicy *GrpcRetryPolicy `json:"retryPolicy,omitempty"`
}

// Route refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_RouteSpec.html
type Route struct {
	// Route's name
	// +optional
	Name string `json:"name,omitempty"`
	// An object that represents the specification of a gRPC route.
	// +optional
	GRPCRoute *GrpcRoute `json:"grpcRoute,omitempty"`
	// An object that represents the specification of an HTTP route.
	// +optional
	HTTPRoute *HttpRoute `json:"httpRoute,omitempty"`
	// An object that represents the specification of an HTTP/2 route.
	// +optional
	HTTP2Route *HttpRoute `json:"http2Route,omitempty"`
	// An object that represents the specification of a TCP route.
	// +optional
	TCPRoute *TcpRoute `json:"tcpRoute,omitempty"`
	// The priority for the route.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	// +optional
	Priority *int64 `json:"priority,omitempty"`
}

type VirtualRouterConditionType string

const (
	// VirtualRouterActive is True when the AppMesh VirtualRouter has been created or found via the API
	VirtualRouterActive VirtualRouterConditionType = "VirtualRouterActive"
)

type VirtualRouterCondition struct {
	// Type of VirtualRouter condition.
	Type VirtualRouterConditionType `json:"type"`
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

// VirtualRouterSpec defines the desired state of VirtualRouter
// refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterSpec.html
type VirtualRouterSpec struct {
	// AWSName is the AppMesh VirtualRouter object's name.
	// If unspecified, it defaults to be "${name}_${namespace}" of k8s VirtualRouter
	// +optional
	AWSName *string `json:"awsName,omitempty"`
	// The listeners that the virtual router is expected to receive inbound traffic from
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1
	Listeners []VirtualRouterListener `json:"listeners,omitempty"`

	// The routes associated with VirtualRouter
	// +optional
	Routes []Route `json:"routes,omitempty"`
}

// VirtualRouterStatus defines the observed state of VirtualRouter
type VirtualRouterStatus struct {
	// MeshArn is the AppMesh Mesh object's Amazon Resource Name
	// +optional
	MeshArn *string `json:"meshArn,omitempty"`
	// VirtualRouterArn is the AppMesh VirtualRouter object's Amazon Resource Name.
	// +optional
	VirtualRouterArn *string `json:"virtualRouterArn,omitempty"`
	// RouteArns is a map of AppMesh Route objects' Amazon Resource Names, indexed by route name.
	// +optional
	RouteArns map[string]string `json:"routeArns,omitempty"`
	// The current VirtualRouter status.
	// +optional
	Conditions []VirtualRouterCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualRouter is the Schema for the virtualrouters API
type VirtualRouter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualRouterSpec   `json:"spec,omitempty"`
	Status VirtualRouterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualRouterList contains a list of VirtualRouter
type VirtualRouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualRouter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualRouter{}, &VirtualRouterList{})
}
