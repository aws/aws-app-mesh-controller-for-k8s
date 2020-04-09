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

// VirtualServiceBackend refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceBackend.html
type VirtualServiceBackend struct {
	// The VirtualService that is acting as a virtual node backend.
	VirtualServiceRef VirtualServiceReference `json:"virtualServiceRef"`
	// A reference to an object that represents the client policy for a backend.
	// +optional
	ClientPolicy *ClientPolicy `json:"clientPolicy,omitempty"`
}

// Backend refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Backend.html
type Backend struct {
	// Specifies a virtual service to use as a backend for a virtual node.
	VirtualService VirtualServiceBackend `json:"virtualService"`
}

// BackendDefaults refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_BackendDefaults.html
type BackendDefaults struct {
	// A reference to an object that represents a client policy.
	// +optional
	ClientPolicy *ClientPolicy `json:"clientPolicy,omitempty"`
}

// AwsCloudMapInstanceAttribute refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapInstanceAttribute.html
type AwsCloudMapInstanceAttribute struct {
	// The name of an AWS Cloud Map service instance attribute key.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	Key string `json:"key"`
	// The value of an AWS Cloud Map service instance attribute key.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Value string `json:"value"`
}

// AwsCloudMapServiceDiscovery refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapServiceDiscovery.html
type AwsCloudMapServiceDiscovery struct {
	// The name of the AWS Cloud Map namespace to use.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	NamespaceName string `json:"namespaceName"`
	// The name of the AWS Cloud Map service to use.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	ServiceName string `json:"serviceName"`
	// A string map that contains attributes with values that you can use to filter instances by any custom attribute that you specified when you registered the instance
	// +optional
	Attributes []AwsCloudMapInstanceAttribute `json:"attributes,omitempty"`
}

// DnsServiceDiscovery refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_DnsServiceDiscovery.html
type DnsServiceDiscovery struct {
	// Specifies the DNS service discovery hostname for the virtual node.
	Hostname string `json:"hostname"`
}

// ServiceDiscovery refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ServiceDiscovery.html
type ServiceDiscovery struct {
	// Specifies any AWS Cloud Map information for the virtual node.
	// +optional
	AWSCloudMap *AwsCloudMapServiceDiscovery `json:"awsCloudMap,omitempty"`
	// Specifies the DNS information for the virtual node.
	// +optional
	DNS *DnsServiceDiscovery `json:"dns,omitempty"`
}

// FileAccessLog refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_FileAccessLog.html
type FileAccessLog struct {
	// The file path to write access logs to.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	Path string `json:"path"`
}

// AccessLog refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AccessLog.html
type AccessLog struct {
	// The file object to send virtual node access logs to.
	// +optional
	File *FileAccessLog `json:"file,omitempty"`
}

// Logging refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Logging.html
type Logging struct {
	// The access log configuration for a virtual node.
	// +optional
	AccessLog *AccessLog `json:"accessLog,omitempty"`
}

type VirtualNodeConditionType string

const (
	// VirtualNodeActive is True when the AppMesh VirtualNode has been created or found via the API
	VirtualNodeActive VirtualNodeConditionType = "VirtualNodeActive"
)

type VirtualNodeCondition struct {
	// Type of VirtualNode condition.
	Type VirtualNodeConditionType `json:"type"`
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

// AwsCloudMapServiceStatus is AWS CloudMap Service object's info
type AwsCloudMapServiceStatus struct {
	// NamespaceID is AWS CloudMap Service object's namespace Id
	// +optional
	NamespaceID *string `json:"namespaceID,omitempty"`
	// ServiceID is AWS CloudMap Service object's Id
	// +optional
	ServiceID *string `json:"serviceID,omitempty"`
}

// VirtualNodeSpec defines the desired state of VirtualNode
// refers to https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceSpec.html
type VirtualNodeSpec struct {
	// AWSName is the AppMesh VirtualNode object's name.
	// If unspecified, it defaults to be "${name}_${namespace}" of k8s VirtualNode
	// +optional
	AWSName *string `json:"awsName,omitempty"`
	// The listener that the virtual node is expected to receive inbound traffic from
	// +kubebuilder:validation:MinItems=0
	// +kubebuilder:validation:MaxItems=1
	// +optional
	Listeners []Listener `json:"listeners,omitempty"`
	// The service discovery information for the virtual node.
	// +optional
	ServiceDiscovery *ServiceDiscovery `json:"serviceDiscovery,omitempty"`
	// The backends that the virtual node is expected to send outbound traffic to.
	// +optional
	Backends []Backend `json:"backends,omitempty"`
	// A reference to an object that represents the defaults for backends.
	// +optional
	BackendDefaults *BackendDefaults `json:"backendDefaults,omitempty"`
	// The inbound and outbound access logging information for the virtual node.
	// +optional
	Logging *Logging `json:"logging,omitempty"`

	// A reference to k8s Mesh CR that this VirtualNode belongs to.
	// The admission controller populates it using Meshes's selector, and prevents users from setting this field.
	//
	// Populated by the system.
	// Read-only.
	// +optional
	MeshRef *MeshReference `json:"meshRef,omitempty"`
}

// VirtualNodeStatus defines the observed state of VirtualNode
type VirtualNodeStatus struct {
	// MeshArn is the AppMesh Mesh object's Amazon Resource Name
	// +optional
	MeshArn *string `json:"meshArn,omitempty"`
	// VirtualNodeArn is the AppMesh VirtualNode object's Amazon Resource Name
	// +optional
	VirtualNodeArn *string `json:"virtualNodeArn,omitempty"`
	// The current VirtualNode status.
	// +optional
	Conditions []VirtualNodeCondition `json:"conditions,omitempty"`
	// AWSCloudMapServiceStatus is AWS CloudMap Service object's info
	// +optional
	AWSCloudMapServiceStatus *AwsCloudMapServiceStatus `json:"awsCloudMapServiceStatus,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualNode is the Schema for the virtualnodes API
type VirtualNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualNodeSpec   `json:"spec,omitempty"`
	Status VirtualNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualNodeList contains a list of VirtualNode
type VirtualNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualNode{}, &VirtualNodeList{})
}
