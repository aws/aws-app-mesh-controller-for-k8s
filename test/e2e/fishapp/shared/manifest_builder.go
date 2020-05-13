package shared

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	Namespace            string
	ServiceDiscoveryType ServiceDiscoveryType

	// required when serviceDiscoveryType == CloudMapServiceDiscovery
	CloudMapNamespace string
}

func (b *ManifestBuilder) BuildNodeDeployment(instanceName string, replicas int32) *appsv1.Deployment {
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
					Labels: labels,
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
					Port:       AppContainerPort,
					TargetPort: intstr.FromInt(AppContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildNodeVirtualNode(instanceName string, backendVirtualServices []types.NamespacedName) *appmesh.VirtualNode {
	labels := b.buildNodeSelectors(instanceName)
	vnName := b.buildNodeName(instanceName)
	var sd *appmesh.ServiceDiscovery
	switch b.ServiceDiscoveryType {
	case DNSServiceDiscovery:
		sd = b.buildNodeDNSServiceDiscovery(instanceName)
	case CloudMapServiceDiscovery:
		sd = b.buildNodeCloudMapServiceDiscovery(instanceName)
	}
	var backends []appmesh.Backend
	for _, backendVS := range backendVirtualServices {
		backends = append(backends, appmesh.Backend{
			VirtualService: appmesh.VirtualServiceBackend{
				VirtualServiceRef: appmesh.VirtualServiceReference{
					Namespace: aws.String(backendVS.Namespace),
					Name:      backendVS.Name,
				},
			},
		})
	}
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vnName,
		},
		Spec: appmesh.VirtualNodeSpec{
			PodSelector: &metav1.LabelSelector{MatchLabels: labels},
			Listeners: []appmesh.Listener{
				{
					PortMapping: appmesh.PortMapping{
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
	VirtualNode types.NamespacedName
	Weight      int64
}

func (b *ManifestBuilder) BuildServiceService(instanceName string) *corev1.Service {
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
					Port:       AppContainerPort,
					TargetPort: intstr.FromInt(AppContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return svc
}

func (b *ManifestBuilder) BuildServiceVirtualRouter(instanceName string, routeCfgs []RouteToWeightedVirtualNodes) *appmesh.VirtualRouter {
	vrName := b.buildServiceName(instanceName)
	var routes []appmesh.Route
	for index, routeCfg := range routeCfgs {
		var targets []appmesh.WeightedTarget
		for _, weightedTarget := range routeCfg.WeightedTargets {
			targets = append(targets, appmesh.WeightedTarget{
				VirtualNodeRef: appmesh.VirtualNodeReference{
					Namespace: aws.String(weightedTarget.VirtualNode.Namespace),
					Name:      weightedTarget.VirtualNode.Name,
				},
				Weight: weightedTarget.Weight,
			})
		}
		routes = append(routes, appmesh.Route{
			Name: fmt.Sprintf("path-%d", index),
			HTTPRoute: &appmesh.HTTPRoute{
				Match: appmesh.HTTPRouteMatch{
					Prefix: routeCfg.Path,
				},
				Action: appmesh.HTTPRouteAction{
					WeightedTargets: targets,
				},
			},
		})
	}
	vr := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vrName,
		},
		Spec: appmesh.VirtualRouterSpec{
			Listeners: []appmesh.VirtualRouterListener{
				{
					PortMapping: appmesh.PortMapping{
						Port:     AppContainerPort,
						Protocol: "http",
					},
				},
			},
			Routes: routes,
		},
	}
	return vr
}

func (b *ManifestBuilder) BuildServiceVirtualService(instanceName string) *appmesh.VirtualService {
	vsName := b.buildServiceName(instanceName)
	vrName := b.buildServiceName(instanceName)
	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vsName,
		},
		Spec: appmesh.VirtualServiceSpec{
			Provider: &appmesh.VirtualServiceProvider{
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterRef: appmesh.VirtualRouterReference{
						Namespace: aws.String(b.Namespace),
						Name:      vrName,
					},
				},
			},
		},
	}
	return vs
}

func (b *ManifestBuilder) buildNodeDNSServiceDiscovery(instanceName string) *appmesh.ServiceDiscovery {
	nodeServiceName := b.buildNodeName(instanceName)
	nodeServiceDNS := fmt.Sprintf("%s.%s.svc.cluster.local.", nodeServiceName, b.Namespace)
	return &appmesh.ServiceDiscovery{
		DNS: &appmesh.DNSServiceDiscovery{
			Hostname: nodeServiceDNS,
		},
	}
}

func (b *ManifestBuilder) buildNodeCloudMapServiceDiscovery(instanceName string) *appmesh.ServiceDiscovery {
	nodeServiceName := b.buildNodeName(instanceName)
	return &appmesh.ServiceDiscovery{
		AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
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

func (b *ManifestBuilder) buildNodeName(instanceName string) string {
	// I like to be explicit about implicit connections between Objects
	return instanceName
}

func (b *ManifestBuilder) buildServiceName(instanceName string) string {
	// I like to be explicit about implicit connections between Objects
	return instanceName
}
