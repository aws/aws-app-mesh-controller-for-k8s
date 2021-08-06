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

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PodConverter implements the interface to convert k8s pod object to a stripped down
// version of pod to save on memory utilized
type podConverter struct {
	podResource     string
	podResourceType runtime.Object
}

// NewPodConverter returns podConverter object
func NewPodConverter() *podConverter {
	return &podConverter{
		podResource:     "pods",
		podResourceType: &v1.Pod{},
	}
}

// ConvertObject converts original pod object to stripped down pod object
func (c *podConverter) ConvertObject(originalObj interface{}) (convertedObj interface{}, err error) {
	pod, ok := originalObj.(*v1.Pod)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to pod")
	}
	return c.stripDownPod(pod), nil
}

// ConvertList converts the original pod list to stripped down list of pod objects
func (c *podConverter) ConvertList(originalList interface{}) (convertedList interface{}, err error) {
	podList, ok := originalList.(*v1.PodList)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to pod list")
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
func (c *podConverter) Resource() string {
	return c.podResource
}

// ResourceType to watch and list
func (c *podConverter) ResourceType() runtime.Object {
	return c.podResourceType
}

// StripDownPod removes all the extra details from pod that are not
// required by the controller.
func (c *podConverter) stripDownPod(pod *v1.Pod) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:              pod.Name,
			Namespace:         pod.Namespace,
			Labels:            pod.Labels,
			DeletionTimestamp: pod.DeletionTimestamp,
		},
		Spec: v1.PodSpec{
			NodeName:      pod.Spec.NodeName,
			RestartPolicy: pod.Spec.RestartPolicy,
		},
		Status: v1.PodStatus{
			Conditions: pod.Status.Conditions,
			Phase:      pod.Status.Phase,
			PodIP:      pod.Status.PodIP,
		},
	}
}
