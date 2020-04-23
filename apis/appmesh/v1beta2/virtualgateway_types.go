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

type VirtualGatewayConditionType string

const (
	// VirtualGatewayActive is True when the AppMesh VirtualGateway has been created or found via the API
	VirtualGatewayActive VirtualGatewayConditionType = "VirtualGatewayActive"
)

// +kubebuilder:validation:Enum=grpc;http;http2
type VirtualGatewayPortProtocol string

// VirtualGatewayPortMapping refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayPortMapping struct {
	// The port used for the port mapping.
	Port PortNumber `json:"port"`
	// The protocol used for the port mapping.
	Protocol VirtualGatewayPortProtocol `json:"protocol"`
}

type VirtualGatewayCondition struct {
	// Type of VirtualGateway condition.
	Type VirtualGatewayConditionType `json:"type"`
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

// VirtualGatewayHealthCheckPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayHealthCheckPolicy struct {
	// The number of consecutive successful health checks that must occur before declaring listener healthy.
	// If unspecified, defaults to be 10
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	// +optional
	HealthyThreshold *int64 `json:"healthyThreshold,omitempty"`
	// The time period in milliseconds between each health check execution.
	// If unspecified, defaults to be 30000
	// +kubebuilder:validation:Minimum=5000
	// +kubebuilder:validation:Maximum=300000
	// +optional
	IntervalMillis *int64 `json:"intervalMillis,omitempty"`
	// The destination path for the health check request.
	// This value is only used if the specified protocol is http or http2. For any other protocol, this value is ignored.
	// +optional
	Path *string `json:"path,omitempty"`
	// The destination port for the health check request.
	// If unspecified, defaults to be same as port defined in the PortMapping for the listener.
	// +optional
	Port *PortNumber `json:"port,omitempty"`
	// The protocol for the health check request
	// If unspecified, defaults to be same as protocol defined in the PortMapping for the listener.
	// +optional
	Protocol *VirtualGatewayPortProtocol `json:"protocol,omitempty"`
	// The amount of time to wait when receiving a response from the health check, in milliseconds.
	// If unspecified, defaults to be 5000
	// +kubebuilder:validation:Minimum=2000
	// +kubebuilder:validation:Maximum=60000
	// +optional
	TimeoutMillis *int64 `json:"timeoutMillis,omitempty"`
	// The number of consecutive failed health checks that must occur before declaring a virtual Gateway unhealthy.
	// If unspecified, defaults to be 2
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	// +optional
	UnhealthyThreshold *int64 `json:"unhealthyThreshold,omitempty"`
}

// VirtualGatewayListenerTLSACMCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayListenerTLSACMCertificate struct {
	// The Amazon Resource Name (ARN) for the certificate.
	CertificateARN string `json:"certificateARN"`
}

// VirtualGatewayListenerTLSFileCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayListenerTLSFileCertificate struct {
	// The certificate chain for the certificate.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	CertificateChain string `json:"certificateChain"`
	// The private key for a certificate stored on the file system of the virtual Gateway.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	PrivateKey string `json:"privateKey"`
}

// VirtualGatewayListenerTLSCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayListenerTLSCertificate struct {
	// A reference to an object that represents an AWS Certificate Manager (ACM) certificate.
	// +optional
	ACM *VirtualGatewayListenerTLSACMCertificate `json:"acm,omitempty"`
	// A reference to an object that represents a local file certificate.
	// +optional
	File *VirtualGatewayListenerTLSFileCertificate `json:"file,omitempty"`
}

const (
	VirtualGatewayListenerTLSModeDisabled   VirtualGatewayListenerTLSMode = "DISABLED"
	VirtualGatewayListenerTLSModePermissive VirtualGatewayListenerTLSMode = "PERMISSIVE"
	VirtualGatewayListenerTLSModeStrict     VirtualGatewayListenerTLSMode = "STRICT"
)

// +kubebuilder:validation:Enum=DISABLED;PERMISSIVE;STRICT
type VirtualGatewayListenerTLSMode string

// VirtualGatewayListenerTLS refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayListenerTLS struct {
	// A reference to an object that represents a listener's TLS certificate.
	Certificate VirtualGatewayListenerTLSCertificate `json:"certificate"`
	// ListenerTLS mode
	Mode VirtualGatewayListenerTLSMode `json:"mode"`
}

// VirtualGatewayFileAccessLog refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayFileAccessLog struct {
	// The file path to write access logs to.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	Path string `json:"path"`
}

// VirtualGatewayAccessLog refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayAccessLog struct {
	// The file object to send virtual gateway access logs to.
	// +optional
	File *VirtualGatewayFileAccessLog `json:"file,omitempty"`
}

// VirtualGatewayLogging refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayLogging struct {
	// The access log configuration for a virtual Gateway.
	// +optional
	AccessLog *VirtualGatewayAccessLog `json:"accessLog,omitempty"`
}

// VirtualGatewayListener refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayListener struct {
	// The port mapping information for the listener.
	PortMapping VirtualGatewayPortMapping `json:"portMapping"`
	// The health check information for the listener.
	// +optional
	HealthCheck *VirtualGatewayHealthCheckPolicy `json:"healthCheck,omitempty"`
	// A reference to an object that represents the Transport Layer Security (TLS) properties for a listener.
	// +optional
	TLS *VirtualGatewayListenerTLS `json:"tls,omitempty"`
	// The inbound and outbound access logging information for the virtual gateway.
	// +optional
	Logging *VirtualGatewayLogging `json:"logging,omitempty"`
}

// VirtualGatewayTLSValidationContextACMTrust refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayTLSValidationContextACMTrust struct {
	// One or more ACM Amazon Resource Name (ARN)s.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=3
	CertificateAuthorityARNs []string `json:"certificateAuthorityARNs"`
}

// VirtualGatewayTLSValidationContextFileTrust refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayTLSValidationContextFileTrust struct {
	// The certificate trust chain for a certificate stored on the file system of the virtual Gateway.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	CertificateChain string `json:"certificateChain"`
}

// VirtualGatewayTLSValidationContextTrust refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayTLSValidationContextTrust struct {
	// A reference to an object that represents a TLS validation context trust for an AWS Certicate Manager (ACM) certificate.
	// +optional
	ACM *VirtualGatewayTLSValidationContextACMTrust `json:"acm,omitempty"`
	// An object that represents a TLS validation context trust for a local file.
	// +optional
	File *VirtualGatewayTLSValidationContextFileTrust `json:"file,omitempty"`
}

// VirtualGatewayTLSValidationContext refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayTLSValidationContext struct {
	// A reference to an object that represents a TLS validation context trust
	Trust VirtualGatewayTLSValidationContextTrust `json:"trust"`
}

// VirtualGatewayClientPolicyTLS refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayClientPolicyTLS struct {
	// Whether the policy is enforced.
	// If unspecified, default settings from AWS API will be applied. Refer to AWS Docs for default settings.
	// +optional
	Enforce *bool `json:"enforce,omitempty"`
	// The range of ports that the policy is enforced for.
	// +optional
	Ports []PortNumber `json:"ports,omitempty"`
	// A reference to an object that represents a TLS validation context.
	Validation VirtualGatewayTLSValidationContext `json:"validation"`
}

// VirtualGatewayClientPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayClientPolicy struct {
	// A reference to an object that represents a Transport Layer Security (TLS) client policy.
	// +optional
	TLS *VirtualGatewayClientPolicyTLS `json:"tls,omitempty"`
}

// VirtualGatewayBackendDefaults refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewayBackendDefaults struct {
	// A reference to an object that represents a client policy.
	// +optional
	ClientPolicy *VirtualGatewayClientPolicy `json:"clientPolicy,omitempty"`
}

// VirtualGatewaySpec defines the desired state of VirtualGateway
// refers to https://docs.aws.amazon.com/app-mesh/latest/userguide/virtual_gateways.html
type VirtualGatewaySpec struct {
	// AWSName is the AppMesh VirtualGateway object's name.
	// If unspecified or empty, it defaults to be "${name}_${namespace}" of k8s VirtualGateway
	// +optional
	AWSName *string `json:"awsName,omitempty"`
	// NamespaceSelector selects Namespaces using labels to designate GatewayRoute membership.
	// This field follows standard label selector semantics; if present but empty, it selects all namespaces.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// PodSelector selects Pods using labels to designate VirtualGateway membership.
	// if unspecified or empty, it selects no pods.
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// The listener that the virtual gateway is expected to receive inbound traffic from
	// +kubebuilder:validation:MinItems=0
	// +kubebuilder:validation:MaxItems=1
	Listeners []VirtualGatewayListener `json:"listeners,omitempty"`
	// A reference to an object that represents the defaults for backend GatewayRoutes.
	// +optional
	BackendDefaults *VirtualGatewayBackendDefaults `json:"backendDefaults,omitempty"`

	// A reference to k8s Mesh CR that this VirtualGateway belongs to.
	// The admission controller populates it using Meshes's selector, and prevents users from setting this field.
	//
	// Populated by the system.
	// Read-only.
	// +optional
	MeshRef *MeshReference `json:"meshRef,omitempty"`
}

// VirtualGatewayStatus defines the observed state of VirtualGateway
type VirtualGatewayStatus struct {
	// MeshARN is the AppMesh Mesh object's Amazon Resource Name
	// +optional
	MeshARN *string `json:"meshARN,omitempty"`
	// VirtualGatewayARN is the AppMesh VirtualGateway object's Amazon Resource Name
	// +optional
	VirtualGatewayARN *string `json:"virtualGatewayARN,omitempty"`
	// The current VirtualGateway status.
	// +optional
	Conditions []VirtualGatewayCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualGateway is the Schema for the virtualgateways API
type VirtualGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualGatewaySpec   `json:"spec,omitempty"`
	Status VirtualGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualGatewayList contains a list of VirtualGateway
type VirtualGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualGateway{}, &VirtualGatewayList{})
}
