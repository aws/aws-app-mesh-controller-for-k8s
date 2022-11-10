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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackendGroupSpec defines the desired state of BackendGroup
type BackendGroupSpec struct {
	// VirtualServices defines the set of virtual services in this BackendGroup.
	VirtualServices []VirtualServiceReference `json:"virtualservices,omitempty"`

	// A reference to k8s Mesh CR that this BackendGroup belongs to.
	// The admission controller populates it using Meshes's selector, and prevents users from setting this field.
	//
	// Populated by the system.
	// Read-only.
	// +optional
	MeshRef *MeshReference `json:"meshRef,omitempty"`
}

// BackendGroupStatus defines the observed state of BackendGroup
type BackendGroupStatus struct {
}

// BackendGroupReference holds a reference to BackendGroup.appmesh.k8s.aws
type BackendGroupReference struct {
	// Namespace is the namespace of BackendGroup CR.
	// If unspecified, defaults to the referencing object's namespace
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Name is the name of BackendGroup CR
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=all
// +kubebuilder:subresource:status
// +kubebuilder:pruning:PreserveUnknownFields
// BackendGroup is the Schema for the backendgroups API
type BackendGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackendGroupSpec   `json:"spec,omitempty"`
	Status BackendGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackendGroupList contains a list of BackendGroup
type BackendGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackendGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackendGroup{}, &BackendGroupList{})
}
