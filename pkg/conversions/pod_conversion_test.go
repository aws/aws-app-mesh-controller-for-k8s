package conversions

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertObj(t *testing.T) {
	podConverter := PodConverter{
		K8sResource:     "pods",
		K8sResourceType: &corev1.Pod{},
	}

	annotations := make(map[string]string)
	annotations["random"] = "TestValue"
	annotations[AppMeshPrefix+"/cpuLimit"] = "60"

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
			Name:      "TestPod",
			Namespace: "TestNameSpace",
			// Annotations and Labels can be stripped down further
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

	assert.Equal(t, convertedPod.Spec.NodeName, oldPod.Spec.NodeName, "NodeName mismatch")
	assert.Equal(t, convertedPod.ObjectMeta.Name, oldPod.ObjectMeta.Name, "Pod Name mismatch")
	assert.Equal(t, convertedPod.ObjectMeta.Namespace, oldPod.ObjectMeta.Namespace, "Pod Namespace mismatch")
	assert.Equal(t, convertedPod.Annotations["random"], "", "Annotations with this prefix should return empty string")
	assert.Equal(t, convertedPod.Annotations[AppMeshPrefix+"/cpuLimit"], "60", "Annotation mismatch")
	assert.True(t, reflect.DeepEqual(convertedPod.ObjectMeta.Labels, oldPod.ObjectMeta.Labels), "Labels mismatch")
	assert.Nil(t, convertedPod.Spec.Containers[0].Command, "Container Command should not be present")
}

func TestConvertList(t *testing.T) {
	podConverter := PodConverter{
		K8sResource:     "pods",
		K8sResourceType: &corev1.Pod{},
	}

	pod1 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "TestPod1",
			Namespace: "TestNameSpace",
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "TestPod2",
			Namespace: "TestNameSpace",
		},
		Spec: v1.PodSpec{
			NodeName: "TestNode",
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
}
