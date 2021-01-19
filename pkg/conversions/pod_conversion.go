// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package conversions

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	AppMeshPrefix = "appmesh.k8s.aws"
)

// PodConverter implements the interface to convert k8s pod object to a stripped down
// version of pod to save on memory utilized
type PodConverter struct {
	K8sResource     string
	K8sResourceType runtime.Object
}

// ConvertObject converts original pod object to stripped down pod object
func (c *PodConverter) ConvertObject(originalObj interface{}) (convertedObj interface{}, err error) {
	pod, ok := originalObj.(*v1.Pod)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to pod")
	}
	return c.stripDownPod(pod), nil
}

// ConvertList converts the original pod list to stripped down list of pod objects
func (c *PodConverter) ConvertList(originalList interface{}) (convertedList interface{}, err error) {
	podList, ok := originalList.(*v1.PodList)
	if !ok {
		return nil, fmt.Errorf("faield to convert object to pod list")
	}
	// We need to set continue in order to allow the pagination to work on converted
	// pod list object
	strippedPodList := v1.PodList{
		ListMeta: metaV1.ListMeta{
			Continue:        podList.Continue,
			ResourceVersion: podList.ResourceVersion,
		},
	}
	for _, pod := range podList.Items {
		strippedPod := c.stripDownPod(&pod)
		strippedPodList.Items = append(strippedPodList.Items, *strippedPod)
	}
	return &strippedPodList, nil
}

// Resource to watch and list
func (c *PodConverter) Resource() string {
	return c.K8sResource
}

// ResourceType to watch and list
func (c *PodConverter) ResourceType() runtime.Object {
	return c.K8sResourceType
}

// StripDownPod removes all the extra details from pod that are not
// required by the controller.
func (c *PodConverter) stripDownPod(pod *v1.Pod) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
			// Annotations and Labels can be stripped down further
			Annotations:       getFilteredAnnotations(pod.Annotations),
			Labels:            pod.Labels,
			DeletionTimestamp: pod.DeletionTimestamp,
		},
		Spec: v1.PodSpec{
			NodeName:         pod.Spec.NodeName,
			SecurityContext:  pod.Spec.SecurityContext,
			RestartPolicy:    pod.Spec.RestartPolicy,
			ReadinessGates:   pod.Spec.ReadinessGates,
			ImagePullSecrets: pod.Spec.ImagePullSecrets,
			Volumes:          pod.Spec.Volumes,
			InitContainers:   getStrippedInitContainers(pod.Spec.InitContainers),
			Containers:       getStrippedContainers(pod.Spec.Containers),
		},
		Status: v1.PodStatus{
			Conditions: pod.Status.Conditions,
			Phase:      pod.Status.Phase,
			PodIP:      pod.Status.PodIP,
		},
	}
}

func getStrippedContainers(containers []v1.Container) []v1.Container {
	strippedContainers := make([]v1.Container, len(containers))
	for i, container := range containers {
		strippedContainers[i].Name = container.Name
		strippedContainers[i].Env = container.Env
		strippedContainers[i].Ports = container.Ports
		strippedContainers[i].ReadinessProbe = container.ReadinessProbe
	}
	return strippedContainers
}

func getStrippedInitContainers(initContainers []v1.Container) []v1.Container {
	strippedContainers := make([]v1.Container, len(initContainers))
	for i, container := range initContainers {
		strippedContainers[i].Name = container.Name
	}
	return strippedContainers
}

// Filters annotations based on AppMeshPrefix
func getFilteredAnnotations(annotations map[string]string) map[string]string {
	strippedAnnotations := make(map[string]string)
	for k, v := range annotations {
		if strings.HasPrefix(k, AppMeshPrefix) {
			strippedAnnotations[k] = v
		}
	}
	return strippedAnnotations
}
