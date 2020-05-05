package shared

import (
	"fmt"
	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultAppImage       = "970805265562.dkr.ecr.us-west-2.amazonaws.com/colorteller:latest"
	defaultHTTPProxyImage = "abhinavsingh/proxy.py:latest"
)

const (
	AppContainerPort       = 9080
	HttpProxyContainerPort = 8899
)

type ServiceDiscoveryType string

const (
	DNSServiceDiscovery      ServiceDiscoveryType = "DNS"
	CloudMapServiceDiscovery ServiceDiscoveryType = "CloudMap"
)

type ManifestBuilder struct {
	MeshName             string
	Namespace            string
	ServiceDiscoveryType ServiceDiscoveryType

	// required when serviceDiscoveryType == CloudMapServiceDiscovery
	CloudMapNamespace string
}

func (b *ManifestBuilder) BuildNodeDeployment(instanceName string, replicas int32) *appsv1.Deployment {
	labels := b.buildNodeSelectors(instanceName)
	dpName := instanceName
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
					Labels: labels,
					Annotations: map[string]string{
						"appmesh.k8s.aws/mesh":  b.MeshName,
						"appmesh.k8s.aws/ports": fmt.Sprintf("%d", AppContainerPort),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: defaultAppImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: AppContainerPort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SERVER_PORT",
									Value: fmt.Sprintf("%d", AppContainerPort),
								},
								{
									Name:  "COLOR",
									Value: instanceName,
								},
							},
						},
						{
							Name:  "http-proxy",
							Image: defaultHTTPProxyImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: HttpProxyContainerPort,
								},
							},
							Args: []string{
								"--hostname=0.0.0.0",
								fmt.Sprintf("--port=%d", HttpProxyContainerPort),
							},
						},
					},
				},
			},
		},
	}
	return dp
}

func (b *ManifestBuilder) BuildNodeService(instanceName string) *corev1.Service {
	labels := b.buildNodeSelectors(instanceName)
	svcName := b.buildNodeServiceName(instanceName)
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
					Port:       AppContainerPort,
					TargetPort: intstr.FromInt(AppContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildNodeVirtualNode(instanceName string, backendVirtualServices []string) *appmeshv1beta1.VirtualNode {
	vnName := instanceName
	var sd *appmeshv1beta1.ServiceDiscovery
	switch b.ServiceDiscoveryType {
	case DNSServiceDiscovery:
		sd = b.buildNodeDNSServiceDiscovery(instanceName)
	case CloudMapServiceDiscovery:
		sd = b.buildNodeCloudMapServiceDiscovery(instanceName)
	}
	var backends []appmeshv1beta1.Backend
	for _, backendVS := range backendVirtualServices {
		backends = append(backends, appmeshv1beta1.Backend{
			VirtualService: appmeshv1beta1.VirtualServiceBackend{
				VirtualServiceName: backendVS,
			},
		})
	}
	vn := &appmeshv1beta1.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vnName,
		},
		Spec: appmeshv1beta1.VirtualNodeSpec{
			MeshName: b.MeshName,
			Listeners: []appmeshv1beta1.Listener{
				{
					PortMapping: appmeshv1beta1.PortMapping{
						Port:     AppContainerPort,
						Protocol: "http",
					},
				},
			},
			ServiceDiscovery: sd,
			Backends:         backends,
		},
	}
	return vn
}

type RouteToWeightedVirtualNodes struct {
	Path            string
	WeightedTargets []WeightedVirtualNode
}

// WeightedVirtualNode is virtual node with weight
type WeightedVirtualNode struct {
	VirtualNodeName string
	Weight          int64
}

func (b *ManifestBuilder) BuildServiceService(instanceName string) *corev1.Service {
	svcName := instanceName
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      svcName,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       AppContainerPort,
					TargetPort: intstr.FromInt(AppContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildServiceVirtualService(instanceName string, routeCfgs []RouteToWeightedVirtualNodes) *appmeshv1beta1.VirtualService {
	svcName := b.buildServiceServiceName(instanceName)
	svcDNS := fmt.Sprintf("%s.%s", svcName, b.Namespace)
	var routes []appmeshv1beta1.Route
	for index, routeCfg := range routeCfgs {
		var targets []appmeshv1beta1.WeightedTarget
		for _, weightedTarget := range routeCfg.WeightedTargets {
			targets = append(targets, appmeshv1beta1.WeightedTarget{
				VirtualNodeName: weightedTarget.VirtualNodeName,
				Weight:          weightedTarget.Weight,
			})
		}
		routes = append(routes, appmeshv1beta1.Route{
			Name: fmt.Sprintf("path-%d", index),
			Http: &appmeshv1beta1.HttpRoute{
				Match: appmeshv1beta1.HttpRouteMatch{
					Prefix: routeCfg.Path,
				},
				Action: appmeshv1beta1.HttpRouteAction{
					WeightedTargets: targets,
				},
			},
		})
	}
	vs := &appmeshv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      svcDNS,
		},
		Spec: appmeshv1beta1.VirtualServiceSpec{
			MeshName: b.MeshName,
			VirtualRouter: &appmeshv1beta1.VirtualRouter{
				Listeners: []appmeshv1beta1.VirtualRouterListener{
					{
						PortMapping: appmeshv1beta1.PortMapping{
							Port:     AppContainerPort,
							Protocol: "http",
						},
					},
				},
			},
			Routes: routes,
		},
	}
	return vs
}

func (b *ManifestBuilder) buildNodeDNSServiceDiscovery(instanceName string) *appmeshv1beta1.ServiceDiscovery {
	nodeServiceName := b.buildNodeServiceName(instanceName)
	nodeServiceDNS := fmt.Sprintf("%s.%s", nodeServiceName, b.Namespace)
	return &appmeshv1beta1.ServiceDiscovery{
		Dns: &appmeshv1beta1.DnsServiceDiscovery{
			HostName: nodeServiceDNS,
		},
	}
}

func (b *ManifestBuilder) buildNodeCloudMapServiceDiscovery(instanceName string) *appmeshv1beta1.ServiceDiscovery {
	nodeServiceName := b.buildNodeServiceName(instanceName)
	return &appmeshv1beta1.ServiceDiscovery{
		CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
			NamespaceName: b.CloudMapNamespace,
			ServiceName:   nodeServiceName,
		},
	}
}

func (b *ManifestBuilder) buildNodeSelectors(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "fish-app",
		"app.kubernetes.io/instance": instanceName,
	}
}

func (b *ManifestBuilder) buildNodeServiceName(instanceName string) string {
	// I like to be explicit about implicit connections between Objects
	return instanceName
}

func (b *ManifestBuilder) buildServiceServiceName(instanceName string) string {
	// I like to be explicit about implicit connections between Objects
	return instanceName
}
