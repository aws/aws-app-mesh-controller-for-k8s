package v1beta2

import "k8s.io/apimachinery/pkg/types"

// +kubebuilder:validation:Enum=s;ms
type DurationUnit string

const (
	DurationUnitS  DurationUnit = "s"
	DurationUnitMS DurationUnit = "ms"
)

type Duration struct {
	// A unit of time.
	Unit DurationUnit `json:"unit"`
	// A number of time units.
	// +kubebuilder:validation:Minimum=0
	Value int64 `json:"value"`
}

// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=65535
type PortNumber int64

// +kubebuilder:validation:Enum=grpc;http;http2;tcp
type PortProtocol string

const (
	PortProtocolGRPC  PortProtocol = "grpc"
	PortProtocolHTTP  PortProtocol = "http"
	PortProtocolHTTP2 PortProtocol = "http2"
	PortProtocolTCP   PortProtocol = "tcp"
)

// PortMapping refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_PortMapping.html
type PortMapping struct {
	// The port used for the port mapping.
	Port PortNumber `json:"port"`
	// The protocol used for the port mapping.
	Protocol PortProtocol `json:"protocol"`
}

// HealthCheckPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HealthCheckPolicy.html
type HealthCheckPolicy struct {
	// The number of consecutive successful health checks that must occur before declaring listener healthy.
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	// +optional
	HealthyThreshold *int64 `json:"healthyThreshold,omitempty"`
	// The time period in milliseconds between each health check execution.
	// +kubebuilder:validation:Minimum=5000
	// +kubebuilder:validation:Maximum=300000
	// +optional
	IntervalMillis *int64 `json:"intervalMillis,omitempty"`
	// The destination path for the health check request.
	// +optional
	Path *string `json:"path,omitempty"`
	// The destination port for the health check request.
	// +optional
	Port *PortNumber `json:"port,omitempty"`
	// The protocol for the health check request
	// +optional
	Protocol *PortProtocol `json:"protocol,omitempty"`
	// The amount of time to wait when receiving a response from the health check, in milliseconds.
	// +kubebuilder:validation:Minimum=2000
	// +kubebuilder:validation:Maximum=60000
	// +optional
	TimeoutMillis *int64 `json:"timeoutMillis,omitempty"`
	// The number of consecutive failed health checks that must occur before declaring a virtual node unhealthy.
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	// +optional
	UnhealthyThreshold *int64 `json:"unhealthyThreshold,omitempty"`
}

// TlsValidationContextAcmTrust refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextAcmTrust.html
type TlsValidationContextAcmTrust struct {
	// One or more ACM Amazon Resource Name (ARN)s.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=3
	CertificateAuthorityArns []string `json:"certificateAuthorityArns"`
}

// TlsValidationContextFileTrust refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextFileTrust.html
type TlsValidationContextFileTrust struct {
	// The certificate trust chain for a certificate stored on the file system of the virtual node that the proxy is running on.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	CertificateChain string `json:"certificateChain"`
}

// TlsValidationContextTrust refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextTrust.html
type TlsValidationContextTrust struct {
	// A reference to an object that represents a TLS validation context trust for an AWS Certicate Manager (ACM) certificate.
	// +optional
	ACM *TlsValidationContextAcmTrust `json:"acm,omitempty"`
	// An object that represents a TLS validation context trust for a local file.
	// +optional
	File *TlsValidationContextFileTrust `json:"file,omitempty"`
}

// TlsValidationContext refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContext.html
type TlsValidationContext struct {
	// A reference to an object that represents a TLS validation context trust
	Trust TlsValidationContextTrust `json:"trust"`
}

// ClientPolicyTls refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicyTls.html
type ClientPolicyTls struct {
	// Whether the policy is enforced.
	// +optional
	Enforce *bool `json:"enforce,omitempty"`
	// One or more ports that the policy is enforced for.
	// +optional
	Ports []PortNumber `json:"ports,omitempty"`
	// A reference to an object that represents a TLS validation context.
	Validation TlsValidationContext `json:"validation"`
}

// ClientPolicy refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicy.html
type ClientPolicy struct {
	// A reference to an object that represents a Transport Layer Security (TLS) client policy.
	// +optional
	TLS *ClientPolicyTls `json:"tls,omitempty"`
}

// ListenerTlsAcmCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsAcmCertificate.html
type ListenerTlsAcmCertificate struct {
	// The Amazon Resource Name (ARN) for the certificate.
	CertificateArn string `json:"certificateArn"`
}

// ListenerTlsFileCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsFileCertificate.html
type ListenerTlsFileCertificate struct {
	// The certificate chain for the certificate.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	CertificateChain string `json:"certificateChain"`
	// The private key for a certificate stored on the file system of the virtual node that the proxy is running on.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	PrivateKey string `json:"privateKey"`
}

// ListenerTlsCertificate refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsCertificate.html
type ListenerTlsCertificate struct {
	// A reference to an object that represents an AWS Certicate Manager (ACM) certificate.
	// +optional
	ACM *ListenerTlsAcmCertificate `json:"acm,omitempty"`
	// A reference to an object that represents a local file certificate.
	// +optional
	File *ListenerTlsFileCertificate `json:"file,omitempty"`
}

// +kubebuilder:validation:Enum=DISABLED;PERMISSIVE;STRICT
type ListenerTlsMode string

const (
	ListenerTlsModeDisabled   ListenerTlsMode = "DISABLED"
	ListenerTlsModePermissive ListenerTlsMode = "PERMISSIVE"
	ListenerTlsModeStrict     ListenerTlsMode = "STRICT"
)

// ListenerTls refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTls.html
type ListenerTls struct {
	// A reference to an object that represents a listener's TLS certificate.
	Certificate ListenerTlsCertificate `json:"certificate"`
	// ListenerTls mode
	Mode ListenerTlsMode `json:"mode"`
}

// Listener refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Listener.html
type Listener struct {
	// The port mapping information for the listener.
	PortMapping PortMapping `json:"portMapping"`
	// The health check information for the listener.
	// +optional
	HealthCheck *HealthCheckPolicy `json:"healthCheck,omitempty"`
	// A reference to an object that represents the Transport Layer Security (TLS) properties for a listener.
	// +optional
	TLS *ListenerTls `json:"tls,omitempty"`
}

// VirtualNodeReference holds a reference to VirtualNode.appmesh.k8s.aws
type VirtualNodeReference struct {
	// Namespace is the namespace of VirtualNode CR.
	// If unspecified, defaults to the referencing object's namespace
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Name is the name of VirtualNode CR
	Name string `json:"name"`
}

// VirtualServiceReference holds a reference to VirtualService.appmesh.k8s.aws
type VirtualServiceReference struct {
	// Namespace is the namespace of VirtualService CR.
	// If unspecified, defaults to the referencing object's namespace
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Name is the name of VirtualService CR
	Name string `json:"name"`
}

// VirtualRouterReference holds a reference to VirtualRouter.appmesh.k8s.aws
type VirtualRouterReference struct {
	// Namespace is the namespace of VirtualRouter CR.
	// If unspecified, defaults to the referencing object's namespace
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Name is the name of VirtualRouter CR
	Name string `json:"name"`
}

// MeshReference holds a reference to Mesh.appmesh.k8s.aws
type MeshReference struct {
	// Name is the name of Mesh CR
	Name string `json:"name"`
	// UID is the UID of Mesh CR
	UID types.UID `json:"uid"`
}
