package manifest

import (
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
)

type ManifestBuilder struct {
	Namespace            string
	ServiceDiscoveryType ServiceDiscoveryType

	// required when serviceDiscoveryType == CloudMapServiceDiscovery
	CloudMapNamespace string

	// If set, will not enable ipv6 on any of the pods. This is needed to handle
	// testing on github actions, where ipv6 is not supported -
	// https://github.com/actions/runner-images/issues/668.
	DisableIPv6 bool
}

type ContainerInfo struct {
	Name          string
	AppImage      string
	ContainerPort int32
	Command       []string
	Env           []corev1.EnvVar
	Args          []string
	VolumeMounts  []corev1.VolumeMount
}

type PodGroupInfo struct {
	GroupLabel  string
	MatchLabels map[string]string
	Pods        []*corev1.Pod
}

func (b *ManifestBuilder) BuildContainerSpec(containersInfo []ContainerInfo) []corev1.Container {

	var containers []corev1.Container
	for index, _ := range containersInfo {
		container := corev1.Container{
			Name:    containersInfo[index].Name,
			Image:   containersInfo[index].AppImage,
			Command: containersInfo[index].Command,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: containersInfo[index].ContainerPort,
				},
			},
			Env:          containersInfo[index].Env,
			Args:         containersInfo[index].Args,
			VolumeMounts: containersInfo[index].VolumeMounts,
		}
		containers = append(containers, container)
	}

	return containers
}

func (b *ManifestBuilder) BuildDeployment(instanceName string, replicas int32, containers []corev1.Container, annotations map[string]string) *appsv1.Deployment {
	labels := b.buildNodeSelectors(instanceName)
	dpName := b.buildNodeName(instanceName)
	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      dpName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: b.createDeploymentAnnotations(annotations),
				},
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes:    []corev1.Volume{{Name: "scripts-vol", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "scripts-configmap"}}}}},
				},
			},
		},
	}
	return dp
}

func (b *ManifestBuilder) createDeploymentAnnotations(annotations map[string]string) map[string]string {
	deploymentAnnotations := map[string]string{}
	if b.DisableIPv6 {
		deploymentAnnotations[inject.AppMeshIPV6Annotation] = "disabled"
	}
	for k, v := range annotations {
		deploymentAnnotations[k] = v
	}
	return deploymentAnnotations
}

func (b *ManifestBuilder) BuildPodGroup(containers []corev1.Container, podGroupName string, podCount int) PodGroupInfo {
	podGroupLabel := utils.RandomDNS1123LabelWithPrefix(podGroupName)
	podGroupMatchLabels := map[string]string{"app": podGroupLabel}
	podGroup := make([]*corev1.Pod, podCount)
	for i := 0; i < podCount; i++ {
		podGroup[i] = b.BuildPod(containers, podGroupMatchLabels)
	}

	return PodGroupInfo{
		GroupLabel:  podGroupLabel,
		MatchLabels: podGroupMatchLabels,
		Pods:        podGroup,
	}
}

func (b *ManifestBuilder) BuildPod(containers []corev1.Container, labels map[string]string) *corev1.Pod {
	name := utils.RandomDNS1123LabelWithPrefix("pod")
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   b.Namespace,
			Name:        name,
			Labels:      labels,
			Annotations: b.createPodAnnotations(),
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
}

func (b *ManifestBuilder) createPodAnnotations() map[string]string {
	deploymentAnnotations := map[string]string{}
	if b.DisableIPv6 {
		deploymentAnnotations[inject.AppMeshIPV6Annotation] = "disabled"
	}
	return deploymentAnnotations
}

func (b *ManifestBuilder) BuildServiceWithSelector(instanceName string, containerPort int32, targetPort int) *corev1.Service {
	labels := b.buildNodeSelectors(instanceName)
	svcName := b.buildNodeName(instanceName)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      svcName,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       containerPort,
					TargetPort: intstr.FromInt(targetPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildServiceWithoutSelector(instanceName string, containerPort int32, targetPort int) *corev1.Service {
	svcName := b.buildServiceName(instanceName)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      svcName,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       containerPort,
					TargetPort: intstr.FromInt(targetPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildK8SSecretsFromPemFile(pemFileBasePath string, tlsFiles []string,
	secretName string, f *framework.Framework) *corev1.Secret {
	certMap := make(map[string][]byte, len(tlsFiles))
	for _, tlsFile := range tlsFiles {
		pemFilePath := pemFileBasePath + tlsFile
		pemBytes, err := utils.ReadFileContents(pemFilePath)
		if err != nil {
			f.Logger.Error("Error while trying to read the PEM file: ", zap.Error(err))
		}
		certMap[tlsFile] = pemBytes
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "tls-e2e",
		},
		Immutable:  nil,
		Data:       certMap,
		StringData: nil,
		Type:       corev1.SecretTypeOpaque,
	}
	return secret
}

func (b *ManifestBuilder) buildNodeSelectors(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "timeout-app",
		"app.kubernetes.io/instance": instanceName,
	}
}

func (b *ManifestBuilder) buildNodeName(instanceName string) string {
	return instanceName
}

func (b *ManifestBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
