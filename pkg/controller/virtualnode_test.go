package controller

import (
	"testing"

	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
)

// newAWSVirtualNode is a helper function to generate an Kubernetes Custom Resource API object.
// Ports and protocols should be arrays of the same length.
func newAPIVirtualNode(ports []int64, protocols []string, backends []string, hostname string) *appmeshv1alpha1.VirtualNode {
	vn := appmeshv1alpha1.VirtualNode{
		Spec: &appmeshv1alpha1.VirtualNodeSpec{},
	}

	if len(ports) != len(protocols) {
		panic("ports and protocols are different lengths")
	}

	if len(ports) > 0 {
		listeners := []appmeshv1alpha1.Listener{}
		for i := range ports {
			listeners = append(listeners, appmeshv1alpha1.Listener{
				PortMapping: appmeshv1alpha1.PortMapping{
					Port:     ports[i],
					Protocol: protocols[i],
				},
			})
		}
		vn.Spec.Listeners = listeners
	}

	if len(backends) > 0 {
		bes := []appmeshv1alpha1.Backend{}
		for i := range backends {
			bes = append(bes, appmeshv1alpha1.Backend{
				VirtualService: appmeshv1alpha1.VirtualServiceBackend{
					VirtualServiceName: backends[i],
				},
			})
		}
		vn.Spec.Backends = bes
	}

	if hostname != "" {
		vn.Spec.ServiceDiscovery = &appmeshv1alpha1.ServiceDiscovery{
			Dns: &appmeshv1alpha1.DnsServiceDiscovery{
				HostName: hostname,
			},
		}
	}
	return &vn
}

// newAWSVirtualNode is a helper function to generate an App Mesh API object.  Ports and protocols should be arrays
// of the same length.
func newAWSVirtualNode(ports []int64, protocols []string, backends []string, hostname string) *aws.VirtualNode {
	awsVn := aws.VirtualNode{
		Data: appmesh.VirtualNodeData{
			Spec: &appmesh.VirtualNodeSpec{},
		},
	}

	if len(ports) != len(protocols) {
		panic("ports and protocols are different lengths")
	}

	if len(ports) > 0 {
		listeners := []*appmesh.Listener{}
		for i := range ports {
			listeners = append(listeners, &appmesh.Listener{
				PortMapping: &appmesh.PortMapping{
					Port:     awssdk.Int64(ports[i]),
					Protocol: awssdk.String(protocols[i]),
				},
			})
		}
		awsVn.Data.Spec.SetListeners(listeners)
	}
	if len(backends) > 0 {
		bes := []*appmesh.Backend{}
		for _, b := range backends {
			bes = append(bes, &appmesh.Backend{
				VirtualService: &appmesh.VirtualServiceBackend{
					VirtualServiceName: awssdk.String(b),
				},
			})
		}
		awsVn.Data.Spec.SetBackends(bes)
	}
	if hostname != "" {
		awsVn.Data.Spec.ServiceDiscovery = &appmesh.ServiceDiscovery{}
		awsVn.Data.Spec.ServiceDiscovery.SetDns(&appmesh.DnsServiceDiscovery{
			Hostname: awssdk.String(hostname),
		})
	}
	return &awsVn
}

var (
	// defaults
	port80       int64 = 80
	protocolHTTP       = "http"
	hostname           = "foo.local"
	backend            = "bar.local"

	// extras
	port443       int64 = 443
	protocolHTTPS       = "https"
	backend2            = "baz.local"
	hostname2           = "fizz.local"

	// Spec with default values
	spec1_default = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname)

	// result with the same values as spec1_default
	result1_default = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname)

	// Spec with extra port
	spec1_extraPort = newAPIVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname)

	// Result with extra port
	result1_extraPort = newAWSVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname)

	// Spec with no ports
	spec1_noPort = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, hostname)

	// Result with no ports
	result1_noPort = newAWSVirtualNode([]int64{}, []string{}, []string{backend}, hostname)

	// Spec with extra backend
	spec1_extraBackend = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname)

	// Result with extra backend
	result1_extraBackend = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname)

	// Spec with no backends
	spec1_noBackends = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname)

	// Result with no backends
	result1_noBackends = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname)

	result1_differentHostname = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname2)
)

var vnodetests = []struct {
	name        string
	spec        *appmeshv1alpha1.VirtualNode
	aws         *aws.VirtualNode
	needsUpdate bool
}{
	{"vnodes are the same", spec1_default, result1_default, false},
	{"extra port in spec", spec1_extraPort, result1_default, true},
	{"extra port in result", spec1_default, result1_extraPort, true},
	{"no ports in spec", spec1_noPort, result1_default, true},
	{"no ports in result", spec1_default, result1_noPort, true},
	{"no ports in either", spec1_noPort, result1_noPort, false},
	{"extra backend in spec", spec1_extraBackend, result1_default, true},
	{"extra backend in result", spec1_default, result1_extraBackend, true},
	{"extra backend in both", spec1_extraBackend, result1_extraBackend, false},
	{"no backend in spec", spec1_noBackends, result1_default, true},
	{"no backend in result", spec1_default, result1_noBackends, true},
	{"no backend in both", spec1_noBackends, result1_noBackends, false},
	{"different hostname in result", spec1_default, result1_differentHostname, true},
}

func TestVNodeNeedsUpdate(t *testing.T) {
	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vnodeNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}
