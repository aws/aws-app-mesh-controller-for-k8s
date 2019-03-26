package controller

import (
	"testing"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
)

// newAWSVirtualNode is a helper function to generate an Kubernetes Custom Resource API object.
// Ports and protocols should be arrays of the same length.
func newAPIVirtualNode(ports []int64, protocols []string, backends []string, hostname string) *appmeshv1beta1.VirtualNode {
	vn := appmeshv1beta1.VirtualNode{
		Spec: appmeshv1beta1.VirtualNodeSpec{},
	}

	if len(ports) != len(protocols) {
		panic("ports and protocols are different lengths")
	}

	if len(ports) > 0 {
		listeners := []appmeshv1beta1.Listener{}
		for i := range ports {
			listeners = append(listeners, appmeshv1beta1.Listener{
				PortMapping: appmeshv1beta1.PortMapping{
					Port:     ports[i],
					Protocol: protocols[i],
				},
			})
		}
		vn.Spec.Listeners = listeners
	}

	if len(backends) > 0 {
		bes := []appmeshv1beta1.Backend{}
		for i := range backends {
			bes = append(bes, appmeshv1beta1.Backend{
				VirtualService: appmeshv1beta1.VirtualServiceBackend{
					VirtualServiceName: backends[i],
				},
			})
		}
		vn.Spec.Backends = bes
	}

	if hostname != "" {
		vn.Spec.ServiceDiscovery = &appmeshv1beta1.ServiceDiscovery{
			Dns: &appmeshv1beta1.DnsServiceDiscovery{
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

func TestVNodeNeedsUpdate(t *testing.T) {

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
		defaultNodeSpec = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname)
		// result with the same values as defaultNodeSpec
		defaultNodeResult       = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname)
		extraPortSpec           = newAPIVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname)
		extraPortResult         = newAWSVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname)
		noPortSpec              = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, hostname)
		noPortResult            = newAWSVirtualNode([]int64{}, []string{}, []string{backend}, hostname)
		extraBackendSpec        = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname)
		extraBackendResult      = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname)
		noBackendsSpec          = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname)
		noBackendsResult        = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname)
		differentHostnameResult = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname2)
	)

	var vnodetests = []struct {
		name        string
		spec        *appmeshv1beta1.VirtualNode
		aws         *aws.VirtualNode
		needsUpdate bool
	}{
		{"vnodes are the same", defaultNodeSpec, defaultNodeResult, false},
		{"extra port in spec", extraPortSpec, defaultNodeResult, true},
		{"extra port in result", defaultNodeSpec, extraPortResult, true},
		{"no ports in spec", noPortSpec, defaultNodeResult, true},
		{"no ports in result", defaultNodeSpec, noPortResult, true},
		{"no ports in either", noPortSpec, noPortResult, false},
		{"extra backend in spec", extraBackendSpec, defaultNodeResult, true},
		{"extra backend in result", defaultNodeSpec, extraBackendResult, true},
		{"extra backend in both", extraBackendSpec, extraBackendResult, false},
		{"no backend in spec", noBackendsSpec, defaultNodeResult, true},
		{"no backend in result", defaultNodeSpec, noBackendsResult, true},
		{"no backend in both", noBackendsSpec, noBackendsResult, false},
		{"different hostname in result", defaultNodeSpec, differentHostnameResult, true},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vnodeNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}
