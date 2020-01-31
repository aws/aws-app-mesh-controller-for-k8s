package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/stretchr/testify/mock"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	ctrlawsmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/mocks"
	appmeshv1beta1mocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/mocks"
	appmeshv1beta1typedmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks"
)

// newAWSVirtualNode is a helper function to generate an Kubernetes Custom Resource API object.
// Ports and protocols should be arrays of the same length.
func newAPIVirtualNode(ports []int64, protocols []string, backends []string, hostname string, fileAccessLogPath *string) *appmeshv1beta1.VirtualNode {
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

	if fileAccessLogPath != nil {
		vn.Spec.Logging = &appmeshv1beta1.Logging{
			AccessLog: &appmeshv1beta1.AccessLog{
				File: &appmeshv1beta1.FileAccessLog{
					Path: awssdk.StringValue(fileAccessLogPath),
				},
			},
		}
	}

	return &vn
}

func newAPIVirtualNodeWithCloudMap(ports []int64, protocols []string, backends []string, serviceDiscovery *appmeshv1beta1.ServiceDiscovery, fileAccessLogPath *string) *appmeshv1beta1.VirtualNode {
	virtualNode := newAPIVirtualNode(ports, protocols, backends, "", fileAccessLogPath)
	virtualNode.Spec.ServiceDiscovery = serviceDiscovery
	return virtualNode
}

// newAWSVirtualNode is a helper function to generate an App Mesh API object.  Ports and protocols should be arrays
// of the same length.
func newAWSVirtualNode(ports []int64, protocols []string, backends []string, hostname string, fileAccessLogPath *string) *aws.VirtualNode {
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
	if fileAccessLogPath != nil {
		awsVn.Data.Spec.Logging = &appmesh.Logging{
			AccessLog: &appmesh.AccessLog{
				File: &appmesh.FileAccessLog{
					Path: fileAccessLogPath,
				},
			},
		}
	}
	return &awsVn
}

func newAWSVirtualNodeWithCloudMap(ports []int64, protocols []string, backends []string, serviceDiscovery *appmesh.ServiceDiscovery, fileAccessLogPath *string) *aws.VirtualNode {
	virtualNode := newAWSVirtualNode(ports, protocols, backends, "", fileAccessLogPath)
	virtualNode.Data.Spec.ServiceDiscovery = serviceDiscovery
	return virtualNode
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

		// Spec with default values
		defaultNodeSpec = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)

		// result with the same values as defaultNodeSpec
		defaultNodeResult  = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		extraPortSpec      = newAPIVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname, nil)
		extraPortResult    = newAWSVirtualNode([]int64{port80, port443}, []string{protocolHTTP, protocolHTTPS}, []string{backend}, hostname, nil)
		noPortSpec         = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, hostname, nil)
		noPortResult       = newAWSVirtualNode([]int64{}, []string{}, []string{backend}, hostname, nil)
		extraBackendSpec   = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname, nil)
		extraBackendResult = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend, backend2}, hostname, nil)
		noBackendsSpec     = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname, nil)
		noBackendsResult   = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{}, hostname, nil)
	)

	var vnodetests = []struct {
		name        string
		spec        *appmeshv1beta1.VirtualNode
		aws         *aws.VirtualNode
		needsUpdate bool
	}{
		{"vnodes are the same", defaultNodeSpec, defaultNodeResult, false},
		//listener
		{"extra port in spec", extraPortSpec, defaultNodeResult, true},
		{"extra port in result", defaultNodeSpec, extraPortResult, true},
		{"no ports in spec", noPortSpec, defaultNodeResult, true},
		{"no ports in result", defaultNodeSpec, noPortResult, true},
		{"no ports in either", noPortSpec, noPortResult, false},
		//backend
		{"extra backend in spec", extraBackendSpec, defaultNodeResult, true},
		{"extra backend in result", defaultNodeSpec, extraBackendResult, true},
		{"extra backend in both", extraBackendSpec, extraBackendResult, false},
		{"no backend in spec", noBackendsSpec, defaultNodeResult, true},
		{"no backend in result", defaultNodeSpec, noBackendsResult, true},
		{"no backend in both", noBackendsSpec, noBackendsResult, false},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vnodeNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestVNodeServiceDiscoveryNeedsUpdate(t *testing.T) {

	var (
		// defaults
		port80       int64 = 80
		protocolHTTP       = "http"
		hostname           = "foo.local"
		backend            = "bar.local"

		// extras
		hostname2 = "fizz.local"

		// Spec with default values
		defaultNodeSpec = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)

		// result with the same values as defaultNodeSpec
		defaultNodeResult       = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		differentHostnameResult = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname2, nil)
		//cloudmap testdata
		cloudMapNodeSpec = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
					Attributes: map[string]string{
						"version": "v1",
						"stage":   "canary",
					},
				},
			},
			nil)
		cloudMapNodeWithNoAttributesSpec = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
				},
			},
			nil)
		noServiceDiscoverySpec    = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, "", nil)
		noServiceDiscoveryResult  = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, "", nil)
		dnsServiceDiscoveryResult = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, "foo.dns.local", nil)
		cloudmapResult            = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("foo"),
					NamespaceName: awssdk.String("local"),
					Attributes: []*appmesh.AwsCloudMapInstanceAttribute{
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("version"),
							Value: awssdk.String("v1"),
						},
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("stage"),
							Value: awssdk.String("canary"),
						},
					},
				},
			},
			nil,
		)
		cloudmapWithNoAttributesResult = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("foo"),
					NamespaceName: awssdk.String("local"),
					Attributes:    []*appmesh.AwsCloudMapInstanceAttribute{},
				},
			},
			nil,
		)
		cloudmapWithDifferentNamespaceResult = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("foo"),
					NamespaceName: awssdk.String("other"),
					Attributes: []*appmesh.AwsCloudMapInstanceAttribute{
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("version"),
							Value: awssdk.String("v1"),
						},
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("stage"),
							Value: awssdk.String("canary"),
						},
					},
				},
			},
			nil,
		)
		cloudmapWithDifferentServiceNameResult = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("bar"),
					NamespaceName: awssdk.String("local"),
					Attributes: []*appmesh.AwsCloudMapInstanceAttribute{
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("version"),
							Value: awssdk.String("v1"),
						},
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("stage"),
							Value: awssdk.String("canary"),
						},
					},
				},
			},
			nil,
		)
		cloudmapWithDifferentAttributeKeyResult = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("foo"),
					NamespaceName: awssdk.String("local"),
					Attributes: []*appmesh.AwsCloudMapInstanceAttribute{
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("version"),
							Value: awssdk.String("v1"),
						},
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("type"),
							Value: awssdk.String("canary"),
						},
					},
				},
			},
			nil,
		)
		cloudmapWithDifferentAttributeValueResult = newAWSVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{backend},
			&appmesh.ServiceDiscovery{
				AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
					ServiceName:   awssdk.String("foo"),
					NamespaceName: awssdk.String("local"),
					Attributes: []*appmesh.AwsCloudMapInstanceAttribute{
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("version"),
							Value: awssdk.String("v2"),
						},
						&appmesh.AwsCloudMapInstanceAttribute{
							Key:   awssdk.String("stage"),
							Value: awssdk.String("canary"),
						},
					},
				},
			},
			nil,
		)
	)

	var vnodetests = []struct {
		name        string
		spec        *appmeshv1beta1.VirtualNode
		aws         *aws.VirtualNode
		needsUpdate bool
	}{
		{"vnodes are the same", defaultNodeSpec, defaultNodeResult, false},
		{"different hostname in result", defaultNodeSpec, differentHostnameResult, true},
		{"different servicediscovery in spec", cloudMapNodeSpec, dnsServiceDiscoveryResult, true},
		{"no servicediscovery in result", cloudMapNodeSpec, noServiceDiscoveryResult, true},
		{"no servicediscovery in both", noServiceDiscoverySpec, noServiceDiscoveryResult, false},
		{"same servicediscovery in both", cloudMapNodeSpec, cloudmapResult, false},
		{"no cloudmap attributes in result", cloudMapNodeSpec, cloudmapWithNoAttributesResult, true},
		{"no cloudmap attributes in both", cloudMapNodeWithNoAttributesSpec, cloudmapWithNoAttributesResult, false},
		{"different cloudmap namespace", cloudMapNodeSpec, cloudmapWithDifferentNamespaceResult, true},
		{"different cloudmap serviceName", cloudMapNodeSpec, cloudmapWithDifferentServiceNameResult, true},
		{"different cloudmap attribute keys", cloudMapNodeSpec, cloudmapWithDifferentAttributeKeyResult, true},
		{"different cloudmap attribute values", cloudMapNodeSpec, cloudmapWithDifferentAttributeValueResult, true},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vnodeNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestVNodeLoggingNeedsUpdate(t *testing.T) {
	var (
		// defaults
		port80            int64 = 80
		protocolHTTP            = "http"
		hostname                = "foo.local"
		backend                 = "bar.local"
		fileAccessLogPath       = awssdk.String("/dev/stdout")

		// Spec with default values
		defaultNodeSpec = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, fileAccessLogPath)

		// result with the same values as defaultNodeSpec
		defaultNodeResult      = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, fileAccessLogPath)
		noLoggingSpec          = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		noLoggingResult        = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		differentLoggingResult = newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, awssdk.String("/dev/stdout2"))
	)

	var vnodetests = []struct {
		name        string
		spec        *appmeshv1beta1.VirtualNode
		aws         *aws.VirtualNode
		needsUpdate bool
	}{
		{"vnodes are the same", defaultNodeSpec, defaultNodeResult, false},
		{"no logging in either", noLoggingSpec, noLoggingResult, false},
		{"no logging in result", defaultNodeSpec, noLoggingResult, true},
		{"different logging in result", defaultNodeSpec, differentLoggingResult, true},
		{"no logging in spec", noLoggingSpec, defaultNodeResult, true},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vnodeNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestHandleCloudMapServiceDiscovery(t *testing.T) {

	var (
		// defaults
		port80       int64 = 80
		protocolHTTP       = "http"
		hostname           = "foo.local"
		backend            = "bar.local"

		// Spec with no service-discovery (client virtual-node)
		noServiceDiscoverySpec = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, "", nil)

		// Spec with default values
		dnsServiceDiscvery = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)

		//cloudmap testdata
		cloudMapServiceDiscovery = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
				},
			},
			nil)

		cloudMapWithNoServiceName = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					NamespaceName: "local",
				},
			},
			nil)

		cloudMapWithNoNamespaceName = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName: "foo",
				},
			},
			nil)
	)

	var vnodetests = []struct {
		name         string
		spec         *appmeshv1beta1.VirtualNode
		errExpected  bool
		callCloudMap bool
		errCloudMap  error
	}{
		{"no service-discovery", noServiceDiscoverySpec, false, false, nil},
		{"dns service-discovery", dnsServiceDiscvery, false, false, nil},
		{"cloudmap service-discovery with no namespaceName", cloudMapWithNoNamespaceName, true, false, nil},
		{"cloudmap service-discovery with no serviceName", cloudMapWithNoServiceName, true, false, nil},
		{"valid cloudmap service-discovery", cloudMapServiceDiscovery, false, true, nil},
		{"valid cloudmap service-discovery but error from cloudmap", cloudMapServiceDiscovery, true, true, errors.New("cloudmap error")},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCloudAPI := new(ctrlawsmocks.CloudAPI)
			mockMeshClientSet := new(appmeshv1beta1mocks.Interface)
			mockAppmeshv1beta1Client := new(appmeshv1beta1typedmocks.AppmeshV1beta1Interface)
			mockMeshClientSet.On(
				"AppmeshV1beta1",
			).Return(mockAppmeshv1beta1Client)
			c := &Controller{
				name:          "test",
				cloud:         mockCloudAPI,
				meshclientset: mockMeshClientSet,
			}
			if tt.callCloudMap {
				mockCloudmapService := &servicediscovery.Service{
					Id: awssdk.String("id"),
				}
				mockCloudAPI.On(
					"CloudMapCreateService",
					ctx,
					mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
					c.name,
				).Return(
					&aws.CloudMapServiceSummary{
						NamespaceID: "nsId",
						ServiceID:   awssdk.StringValue(mockCloudmapService.Id),
					},
					tt.errCloudMap,
				)

				mockVirtualNodeInterface := new(appmeshv1beta1typedmocks.VirtualNodeInterface)
				mockAppmeshv1beta1Client.On(
					"VirtualNodes",
					mock.Anything,
				).Return(mockVirtualNodeInterface)
				mockVirtualNodeInterface.On(
					"UpdateStatus",
					mock.AnythingOfType("*v1beta1.VirtualNode"),
				).Return(nil, nil)
			}
			err := c.handleServiceDiscovery(ctx, tt.spec, tt.spec.DeepCopy())
			if err != nil && !tt.errExpected {
				t.Errorf("unexpected error, %v", err)
			} else if err == nil && tt.errExpected {
				t.Errorf("expected error but no error was thrown")
			}
		})
	}
}

func TestVirtualNode_deregisterInstancesForVirtualNode(t *testing.T) {
	var (
		// defaults
		port80       int64 = 80
		protocolHTTP       = "http"
		hostname           = "foo.local"
		backend            = "bar.local"
		k8sNamespace       = "test-ns"
		meshName           = "test-mesh"
		vnodeName          = "test-vn"

		noServiceDiscoverySpec   = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, "", nil)
		dnsServiceDiscvery       = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		cloudMapServiceDiscovery = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
				},
			},
			nil)

		//vnodeInstance should be deregistered
		vnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("i-1"),
			Attributes: map[string]*string{
				attributeKeyAppMeshMeshName:        awssdk.String(meshName),
				attributeKeyAppMeshVirtualNodeName: awssdk.String(namespacedResourceName(vnodeName, k8sNamespace)),
			},
		}

		//diffVnodeInstance should be deregistered
		diffVnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("i-4"),
			Attributes: map[string]*string{
				attributeKeyAppMeshMeshName:        awssdk.String(meshName),
				attributeKeyAppMeshVirtualNodeName: awssdk.String(namespacedResourceName(vnodeName+"other", k8sNamespace)),
			},
		}

		//nonVnodeInstance should not be deregistered
		nonVnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("i-2"),
			Attributes: map[string]*string{
				attributeKeyAppMeshMeshName: awssdk.String(meshName),
			},
		}

		//nonMeshInstance should not be deregistered
		nonMeshInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("i-3"),
		}
	)

	var vnodetests = []struct {
		name       string
		spec       *appmeshv1beta1.VirtualNode
		isCloudmap bool
	}{
		{"no service-discovery", noServiceDiscoverySpec, false},
		{"dns service-discovery", dnsServiceDiscvery, false},
		{"cloudmap service-discovery", cloudMapServiceDiscovery, true},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tt.spec.Namespace = "test-ns"
			tt.spec.Name = vnodeName
			tt.spec.Spec.MeshName = meshName
			mockCloudAPI := new(ctrlawsmocks.CloudAPI)
			mockCloudAPI.On(
				"ListInstances",
				ctx,
				mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
			).Return(
				[]*servicediscovery.InstanceSummary{
					vnodeInstance, nonVnodeInstance, nonMeshInstance, diffVnodeInstance,
				},
				nil,
			)
			mockCloudAPI.On(
				"DeregisterInstance",
				ctx,
				awssdk.StringValue(vnodeInstance.Id),
				mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
			).Return(nil)
			for _, id := range []*string{nonVnodeInstance.Id, nonMeshInstance.Id, diffVnodeInstance.Id} {
				mockCloudAPI.On(
					"DeregisterInstance",
					ctx,
					awssdk.StringValue(id),
					mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
				).Return(fmt.Errorf("Unexpected deregisterInstance call id=%s", awssdk.StringValue(id)))
			}

			c := &Controller{
				cloud: mockCloudAPI,
			}
			err := c.deregisterInstancesForVirtualNode(ctx, tt.spec)
			if err != nil {
				t.Errorf("Unexpected error %+v", err)
			}
		})
	}
}

func TestMutateVirtualNodeForProcessing(t *testing.T) {
	var (
		// defaults
		port80       int64 = 80
		protocolHTTP       = "http"
		hostname           = "foo.local"
		backend            = "bar.local"

		noServiceDiscoverySpec                   = newAPIVirtualNode([]int64{}, []string{}, []string{backend}, "", nil)
		dnsServiceDiscvery                       = newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, nil)
		cloudMapServiceDiscoveryWithNoAttributes = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
				},
			},
			nil)

		cloudMapServiceDiscoveryWithAttributes = newAPIVirtualNodeWithCloudMap([]int64{port80},
			[]string{protocolHTTP},
			[]string{},
			&appmeshv1beta1.ServiceDiscovery{
				CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
					ServiceName:   "foo",
					NamespaceName: "local",
					Attributes:    map[string]string{"key1": "value1"},
				},
			},
			nil)
	)

	var vnodetests = []struct {
		name       string
		spec       *appmeshv1beta1.VirtualNode
		isCloudMap bool
	}{
		{"no service-discovery", noServiceDiscoverySpec, false},
		{"dns service-discovery", dnsServiceDiscvery, false},
		{"cloudmap service-discovery with no attributes", cloudMapServiceDiscoveryWithNoAttributes, true},
		{"cloudmap service-discovery with attributes", cloudMapServiceDiscoveryWithAttributes, true},
	}

	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{}
			tt.spec.Namespace = "test-ns"
			tt.spec.Spec.MeshName = "test-mesh"
			c.mutateVirtualNodeForProcessing(tt.spec)
			if !strings.HasSuffix(tt.spec.Name, tt.spec.Namespace) {
				t.Errorf("Virtual Node name is not namespaced correctly")
			}

			if tt.isCloudMap {
				meshAttrValue, ok := tt.spec.Spec.ServiceDiscovery.CloudMap.Attributes[attributeKeyAppMeshMeshName]
				if !ok || meshAttrValue != tt.spec.Spec.MeshName {
					t.Errorf("CloudMap service discover attribute %s is not set properly", attributeKeyAppMeshMeshName)
				}

				virtualNodeAttrValue, ok := tt.spec.Spec.ServiceDiscovery.CloudMap.Attributes[attributeKeyAppMeshVirtualNodeName]
				if !ok || virtualNodeAttrValue != tt.spec.Name {
					t.Errorf("CloudMap service discover attribute %s is not set properly", attributeKeyAppMeshVirtualNodeName)
				}
			}
		})
	}

}

func TestVNodeListenerHealthCheckNeedsUpdate(t *testing.T) {
	var (
		// defaults
		port80            int64 = 80
		protocolHTTP            = "http"
		hostname                = "foo.local"
		backend                 = "bar.local"
		fileAccessLogPath       = awssdk.String("/dev/stdout")

		specHealthCheck = &appmeshv1beta1.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(1),
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/"),
			Port:               awssdk.Int64(9080),
			Protocol:           awssdk.String("HTTP"),
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}

		resultHealthCheck = &appmesh.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(1),
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/"),
			Port:               awssdk.Int64(9080),
			Protocol:           awssdk.String("HTTP"),
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}

		differentHealthyThresholdHealthCheck = &appmesh.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(2), //diff
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/"),
			Port:               awssdk.Int64(9080),
			Protocol:           awssdk.String("HTTP"),
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}

		differentPortHealthCheck = &appmesh.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(1),
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/"),
			Port:               awssdk.Int64(8080), //diff
			Protocol:           awssdk.String("HTTP"),
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}

		differentPathHealthCheck = &appmesh.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(1),
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/diff"), //diff
			Port:               awssdk.Int64(9080),
			Protocol:           awssdk.String("HTTP"),
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}

		differentProtocolHealthCheck = &appmesh.HealthCheckPolicy{
			HealthyThreshold:   awssdk.Int64(1),
			IntervalMillis:     awssdk.Int64(1000),
			Path:               awssdk.String("/"),
			Port:               awssdk.Int64(9080),
			Protocol:           awssdk.String("TCP"), //diff
			TimeoutMillis:      awssdk.Int64(1000),
			UnhealthyThreshold: awssdk.Int64(1),
		}
	)

	var vnodetests = []struct {
		name        string
		desired     *appmeshv1beta1.HealthCheckPolicy
		target      *appmesh.HealthCheckPolicy
		needsUpdate bool
	}{
		{"no health-check defined", nil, nil, false},
		{"health-check are in sync", specHealthCheck, resultHealthCheck, false},
		{"Path: target health-check is different", specHealthCheck, differentPathHealthCheck, true},
		{"Protocol: target health-check is different", specHealthCheck, differentProtocolHealthCheck, true},
		{"Port: target health-check is different", specHealthCheck, differentPortHealthCheck, true},
		{"HealthyThreshold: target health-check is different", specHealthCheck, differentHealthyThresholdHealthCheck, true},
	}
	for _, tt := range vnodetests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, fileAccessLogPath)
			spec.Spec.Listeners[0].HealthCheck = tt.desired
			result := newAWSVirtualNode([]int64{port80}, []string{protocolHTTP}, []string{backend}, hostname, fileAccessLogPath)
			result.Data.Spec.Listeners[0].HealthCheck = tt.target
			if res := vnodeNeedsUpdate(spec, result); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}
