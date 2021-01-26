package conversions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertObj(t *testing.T) {
	podConverter := NewPodConverter()

	annotations := make(map[string]string)
	annotations["random"] = "TestValue"
	annotations["appmesh.k8s.aws/cpuLimit"] = "60"

	labels := make(map[string]string)
	labels["app"] = "TestApp"
	labels["role"] = "front"

	commands := []string{"sh", "-c", "echo Container 1 is Running; sleep 360000"}

	container := corev1.Container{
		Name:            "busybox",
		Image:           "busybox",
		ImagePullPolicy: "IfNotPresent",
		Command:         commands,
	}

	containers := make([]corev1.Container, 0)
	containers = append(containers, container)

	oldPod := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "TestPod",
			Namespace:   "TestNameSpace",
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: v1.PodSpec{
			NodeName:   "TestNode",
			Containers: containers,
		},
	}

	convertedObj, err := podConverter.ConvertObject(oldPod)
	if err != nil {
		t.Error(err)
	}

	convertedPod, ok := convertedObj.(*corev1.Pod)
	if !ok {
		t.Error("Conversion Failed")
	}

	assert.Equal(t, convertedPod.Spec.NodeName, "TestNode", "NodeName must be excluded/empty")
	assert.Equal(t, convertedPod.ObjectMeta.Name, oldPod.ObjectMeta.Name, "Pod Name mismatch")
	assert.Equal(t, convertedPod.ObjectMeta.Namespace, oldPod.ObjectMeta.Namespace, "Pod Namespace mismatch")
	assert.Equal(t, len(convertedPod.ObjectMeta.Annotations), 0, "Annotations must be excluded/empty")
	assert.Equal(t, len(convertedPod.ObjectMeta.Labels), 2, "Labels must be excluded/empty")
	assert.Equal(t, len(convertedPod.Spec.Containers), 0, "Container should be excluded/empty")
}

func TestConvertList(t *testing.T) {
	podConverter := NewPodConverter()

	annotations := make(map[string]string)
	annotations["random"] = "TestValue"
	annotations["appmesh.k8s.aws/cpuLimit"] = "60"

	pod1 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "TestPod1",
			Namespace:   "TestNameSpace",
			Annotations: annotations,
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode1",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "TestPod2",
			Namespace:   "TestNameSpace",
			Annotations: annotations,
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode2",
		},
	}

	expectedpod1 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "TestPod1",
			Namespace: "TestNameSpace",
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode1",
		},
	}

	expectedpod2 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "TestPod2",
			Namespace: "TestNameSpace",
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode2",
		},
	}

	podList := &corev1.PodList{
		Items: []corev1.Pod{
			*pod1,
			*pod2,
		},
	}

	convertedList, err := podConverter.ConvertList(podList)
	if err != nil {
		t.Error(err)
	}
	assert.NotNil(t, convertedList, "Converted List cannot be Nil")
	assert.Equal(t, len(convertedList.(*corev1.PodList).Items), 2, "Length mismatch")
	convertedPods := convertedList.(*v1.PodList).Items
	assert.Equal(t, convertedPods[0].Spec.NodeName, expectedpod1.Spec.NodeName, "Nodename mismatch for pod1")
	assert.Equal(t, convertedPods[1].Spec.NodeName, expectedpod2.Spec.NodeName, "Nodename mismatch for pod2")
	assert.Equal(t, convertedPods[0].Name, expectedpod1.Name, "Name mismatch for pod 1")
	assert.Equal(t, convertedPods[0].Namespace, expectedpod1.Namespace, "Namespace mismatch for pod1")
	assert.Equal(t, convertedPods[1].ObjectMeta.Name, expectedpod2.Name, "Name mismatch for pod 2")
	assert.Equal(t, convertedPods[1].ObjectMeta.Namespace, expectedpod2.Namespace, "Namespace mismatch for pod2")
	assert.Equal(t, len(convertedPods[0].Annotations), 0, "Annotations should be excluded")
	assert.Equal(t, len(convertedPods[1].Annotations), 0, "Annotations should be excluded")
}
