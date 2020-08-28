package manifest

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ServiceDiscoveryType string

const (
	DNSServiceDiscovery      ServiceDiscoveryType = "DNS"
	CloudMapServiceDiscovery ServiceDiscoveryType = "CloudMap"
)

type VNBuilder struct {
	Namespace            string
	ServiceDiscoveryType ServiceDiscoveryType

	// required when serviceDiscoveryType == CloudMapServiceDiscovery
	CloudMapNamespace string
}

func (b *VNBuilder) BuildVirtualNode(instanceName string, backendVirtualServices []types.NamespacedName,
	listeners []appmesh.Listener, backendDefaults *appmesh.BackendDefaults) *appmesh.VirtualNode {
	labels := b.buildSelectors(instanceName)
	vnName := b.buildName(instanceName)

	var sd *appmesh.ServiceDiscovery

	switch b.ServiceDiscoveryType {
	case DNSServiceDiscovery:
		sd = b.buildDNSServiceDiscovery(instanceName)
	case CloudMapServiceDiscovery:
		sd = b.buildCloudMapServiceDiscovery(instanceName)
	}

	var backends []appmesh.Backend

	for _, backendVS := range backendVirtualServices {
		backends = append(backends, appmesh.Backend{
			VirtualService: appmesh.VirtualServiceBackend{
				VirtualServiceRef: &appmesh.VirtualServiceReference{
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
			PodSelector:      &metav1.LabelSelector{MatchLabels: labels},
			Listeners:        listeners,
			ServiceDiscovery: sd,
			Backends:         backends,
			BackendDefaults:  backendDefaults,
		},
	}
	return vn
}

func (b *VNBuilder) BuildListener(protocol appmesh.PortProtocol, port appmesh.PortNumber) appmesh.Listener {
	return appmesh.Listener{
		PortMapping: appmesh.PortMapping{
			Port:     port,
			Protocol: protocol,
		},
	}
}

func (b *VNBuilder) BuildListenerWithTimeout(protocol appmesh.PortProtocol, port appmesh.PortNumber, timeout int64, unit appmesh.DurationUnit) appmesh.Listener {
	return appmesh.Listener{
		PortMapping: appmesh.PortMapping{
			Port:     port,
			Protocol: protocol,
		},
		Timeout: &appmesh.ListenerTimeout{
			HTTP: &appmesh.HTTPTimeout{
				PerRequest: &appmesh.Duration{
					Unit:  unit,
					Value: timeout,
				},
				Idle: nil,
			},
		},
	}
}

func (b *VNBuilder) BuildListenerWithTLS(protocol appmesh.PortProtocol, port appmesh.PortNumber, listenerTLS *appmesh.ListenerTLS) appmesh.Listener {
	return appmesh.Listener{
		PortMapping: appmesh.PortMapping{
			Port:     port,
			Protocol: protocol,
		},
		TLS: listenerTLS,
	}
}

func (b *VNBuilder) buildDNSServiceDiscovery(instanceName string) *appmesh.ServiceDiscovery {
	nodeServiceName := b.buildName(instanceName)
	nodeServiceDNS := fmt.Sprintf("%s.%s.svc.cluster.local", nodeServiceName, b.Namespace)
	return &appmesh.ServiceDiscovery{
		DNS: &appmesh.DNSServiceDiscovery{
			Hostname: nodeServiceDNS,
		},
	}
}

func (b *VNBuilder) buildCloudMapServiceDiscovery(instanceName string) *appmesh.ServiceDiscovery {
	nodeServiceName := b.buildName(instanceName)
	return &appmesh.ServiceDiscovery{
		AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
			NamespaceName: b.CloudMapNamespace,
			ServiceName:   nodeServiceName,
		},
	}
}

func (b *VNBuilder) buildSelectors(instanceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": instanceName,
	}
}

func (b *VNBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *VNBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
