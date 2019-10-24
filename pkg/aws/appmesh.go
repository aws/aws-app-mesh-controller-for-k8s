package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	set "github.com/deckarep/golang-set"
	"k8s.io/klog"
)

const (
	DescribeMeshTimeout           = 10
	CreateMeshTimeout             = 10
	DeleteMeshTimeout             = 10
	DescribeVirtualNodeTimeout    = 10
	CreateVirtualNodeTimeout      = 10
	UpdateVirtualNodeTimeout      = 10
	DeleteVirtualNodeTimeout      = 10
	DescribeVirtualServiceTimeout = 10
	CreateVirtualServiceTimeout   = 10
	UpdateVirtualServiceTimeout   = 10
	DeleteVirtualServiceTimeout   = 10
	DescribeVirtualRouterTimeout  = 10
	CreateVirtualRouterTimeout    = 10
	UpdateVirtualRouterTimeout    = 10
	DeleteVirtualRouterTimeout    = 10
	DescribeRouteTimeout          = 10
	CreateRouteTimeout            = 10
	ListRoutesTimeout             = 10
	UpdateRouteTimeout            = 10
	DeleteRouteTimeout            = 10
	DefaultHealthyThreshold       = 10
	DefaultIntervalMillis         = 30000
	DefaultTimeoutMillis          = 5000
	DefaultUnhealthyThreshold     = 2
)

type AppMeshAPI interface {
	GetMesh(context.Context, string) (*Mesh, error)
	CreateMesh(context.Context, *appmeshv1beta1.Mesh) (*Mesh, error)
	DeleteMesh(context.Context, string) (*Mesh, error)
	GetVirtualNode(context.Context, string, string) (*VirtualNode, error)
	CreateVirtualNode(context.Context, *appmeshv1beta1.VirtualNode) (*VirtualNode, error)
	UpdateVirtualNode(context.Context, *appmeshv1beta1.VirtualNode) (*VirtualNode, error)
	DeleteVirtualNode(context.Context, string, string) (*VirtualNode, error)
	GetVirtualService(context.Context, string, string) (*VirtualService, error)
	CreateVirtualService(context.Context, *appmeshv1beta1.VirtualService) (*VirtualService, error)
	UpdateVirtualService(context.Context, *appmeshv1beta1.VirtualService) (*VirtualService, error)
	DeleteVirtualService(context.Context, string, string) (*VirtualService, error)
	GetVirtualRouter(context.Context, string, string) (*VirtualRouter, error)
	CreateVirtualRouter(context.Context, *appmeshv1beta1.VirtualRouter, string) (*VirtualRouter, error)
	UpdateVirtualRouter(context.Context, *appmeshv1beta1.VirtualRouter, string) (*VirtualRouter, error)
	DeleteVirtualRouter(context.Context, string, string) (*VirtualRouter, error)
	GetRoute(context.Context, string, string, string) (*Route, error)
	CreateRoute(context.Context, *appmeshv1beta1.Route, string, string) (*Route, error)
	UpdateRoute(context.Context, *appmeshv1beta1.Route, string, string) (*Route, error)
	GetRoutesForVirtualRouter(context.Context, string, string) (Routes, error)
	DeleteRoute(context.Context, string, string, string) (*Route, error)
}

type Mesh struct {
	Data appmesh.MeshData
}

// Name returns the name or an empty string
func (v *Mesh) Name() string {
	return aws.StringValue(v.Data.MeshName)
}

// GetMesh calls describe mesh.
func (c *Cloud) GetMesh(ctx context.Context, name string) (*Mesh, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DescribeMeshTimeout)
	defer cancel()

	input := &appmesh.DescribeMeshInput{
		MeshName: aws.String(name),
	}

	if output, err := c.appmesh.DescribeMeshWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Mesh == nil {
		return nil, fmt.Errorf("mesh %s not found", name)
	} else {
		return &Mesh{
			Data: *output.Mesh,
		}, nil
	}
}

// CreateMesh converts the desired mesh spec into CreateMeshInput and calls create mesh.
func (c *Cloud) CreateMesh(ctx context.Context, mesh *appmeshv1beta1.Mesh) (*Mesh, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateMeshTimeout)
	defer cancel()

	input := &appmesh.CreateMeshInput{
		MeshName: aws.String(mesh.Name),
	}

	if output, err := c.appmesh.CreateMeshWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Mesh == nil {
		return nil, fmt.Errorf("mesh %s not found", mesh.Name)
	} else {
		return &Mesh{
			Data: *output.Mesh,
		}, nil
	}
}

// DeleteMesh deletes the given mesh
func (c *Cloud) DeleteMesh(ctx context.Context, name string) (*Mesh, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DeleteMeshTimeout)
	defer cancel()

	input := &appmesh.DeleteMeshInput{
		MeshName: aws.String(name),
	}

	if output, err := c.appmesh.DeleteMeshWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Mesh == nil {
		return nil, fmt.Errorf("mesh %s not found", name)
	} else {
		return &Mesh{
			Data: *output.Mesh,
		}, nil
	}
}

type VirtualNode struct {
	Data appmesh.VirtualNodeData
}

// Name returns the name or an empty string
func (v *VirtualNode) Name() string {
	return aws.StringValue(v.Data.VirtualNodeName)
}

// Status returns the status or an empty string
func (v *VirtualNode) Status() string {
	if v.Data.Status != nil &&
		v.Data.Status.Status != nil {
		return aws.StringValue(v.Data.Status.Status)
	}
	return ""
}

// HostName returns the hostname or an empty string
func (v *VirtualNode) HostName() string {
	if v.Data.Spec.ServiceDiscovery != nil &&
		v.Data.Spec.ServiceDiscovery.Dns != nil {
		return aws.StringValue(v.Data.Spec.ServiceDiscovery.Dns.Hostname)
	}
	return ""
}

// Listeners converts into our API type
func (v *VirtualNode) Listeners() []appmeshv1beta1.Listener {
	if v.Data.Spec.Listeners == nil {
		return []appmeshv1beta1.Listener{}
	}

	var listeners = []appmeshv1beta1.Listener{}
	for _, appmeshListener := range v.Data.Spec.Listeners {
		listener := appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     aws.Int64Value(appmeshListener.PortMapping.Port),
				Protocol: aws.StringValue(appmeshListener.PortMapping.Protocol),
			},
		}
		if appmeshListener.HealthCheck != nil {
			healthCheck := &appmeshv1beta1.HealthCheckPolicy{
				HealthyThreshold:   appmeshListener.HealthCheck.HealthyThreshold,
				IntervalMillis:     appmeshListener.HealthCheck.IntervalMillis,
				Path:               appmeshListener.HealthCheck.Path,
				Port:               appmeshListener.HealthCheck.Port,
				Protocol:           appmeshListener.HealthCheck.Protocol,
				TimeoutMillis:      appmeshListener.HealthCheck.TimeoutMillis,
				UnhealthyThreshold: appmeshListener.HealthCheck.UnhealthyThreshold,
			}
			listener.HealthCheck = healthCheck
		}
		listeners = append(listeners, listener)
	}
	return listeners
}

// ListenersSet converts into a Set of Listeners
func (v *VirtualNode) ListenersSet() set.Set {
	listeners := v.Listeners()
	s := set.NewSet()
	for i := range listeners {
		s.Add(listeners[i])
	}
	return s
}

// Backends converts into our API type
func (v *VirtualNode) Backends() []appmeshv1beta1.Backend {
	if v.Data.Spec.Backends == nil {
		return []appmeshv1beta1.Backend{}
	}

	var backends = []appmeshv1beta1.Backend{}
	for _, b := range v.Data.Spec.Backends {
		backends = append(backends, appmeshv1beta1.Backend{
			VirtualService: appmeshv1beta1.VirtualServiceBackend{
				VirtualServiceName: aws.StringValue(b.VirtualService.VirtualServiceName),
			},
		})
	}
	return backends
}

// BackendsSet returns a set of Backends defined for virtual-node
func (v *VirtualNode) BackendsSet() set.Set {
	backends := v.Backends()
	s := set.NewSet()
	for i := range backends {
		s.Add(backends[i])
	}
	return s
}

// GetVirtualNode calls describe virtual node.
func (c *Cloud) GetVirtualNode(ctx context.Context, name string, meshName string) (*VirtualNode, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DescribeVirtualNodeTimeout)
	defer cancel()

	input := &appmesh.DescribeVirtualNodeInput{
		MeshName:        aws.String(meshName),
		VirtualNodeName: aws.String(name),
	}

	if output, err := c.appmesh.DescribeVirtualNodeWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualNode == nil {
		return nil, fmt.Errorf("virtual node %s not found", name)
	} else {
		return &VirtualNode{
			Data: *output.VirtualNode,
		}, nil
	}
}

// CreateVirtualNode converts the desired virtual node spec into CreateVirtualNodeInput and calls create
// virtual node.
func (c *Cloud) CreateVirtualNode(ctx context.Context, vnode *appmeshv1beta1.VirtualNode) (*VirtualNode, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateVirtualNodeTimeout)
	defer cancel()

	input := &appmesh.CreateVirtualNodeInput{
		VirtualNodeName: aws.String(vnode.Name),
		MeshName:        aws.String(vnode.Spec.MeshName),
		Spec:            &appmesh.VirtualNodeSpec{},
	}

	if vnode.Spec.Listeners != nil {
		listeners := []*appmesh.Listener{}
		for _, listener := range vnode.Spec.Listeners {
			appmeshListener := &appmesh.Listener{
				PortMapping: &appmesh.PortMapping{
					Port:     aws.Int64(listener.PortMapping.Port),
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			}
			if listener.HealthCheck != nil {
				appmeshHealthCheck := &appmesh.HealthCheckPolicy{
					HealthyThreshold:   defaultInt64(listener.HealthCheck.HealthyThreshold, DefaultHealthyThreshold),
					IntervalMillis:     defaultInt64(listener.HealthCheck.IntervalMillis, DefaultIntervalMillis),
					Path:               listener.HealthCheck.Path,
					Port:               defaultInt64(listener.HealthCheck.Port, listener.PortMapping.Port),          //using listener's port
					Protocol:           defaultString(listener.HealthCheck.Protocol, listener.PortMapping.Protocol), //using listener's protocol
					TimeoutMillis:      defaultInt64(listener.HealthCheck.TimeoutMillis, DefaultTimeoutMillis),
					UnhealthyThreshold: defaultInt64(listener.HealthCheck.UnhealthyThreshold, DefaultUnhealthyThreshold),
				}
				appmeshListener.HealthCheck = appmeshHealthCheck
			}
			listeners = append(listeners, appmeshListener)
		}
		input.Spec.SetListeners(listeners)
	}

	if vnode.Spec.Backends != nil {
		backends := []*appmesh.Backend{}
		for _, backend := range vnode.Spec.Backends {
			backends = append(backends, &appmesh.Backend{
				VirtualService: &appmesh.VirtualServiceBackend{
					VirtualServiceName: aws.String(backend.VirtualService.VirtualServiceName),
				},
			})
		}
		input.Spec.SetBackends(backends)
	}

	if vnode.Spec.ServiceDiscovery != nil {
		if vnode.Spec.ServiceDiscovery.Dns != nil {
			serviceDiscovery := &appmesh.ServiceDiscovery{
				Dns: &appmesh.DnsServiceDiscovery{
					Hostname: aws.String(vnode.Spec.ServiceDiscovery.Dns.HostName),
				},
			}
			input.Spec.SetServiceDiscovery(serviceDiscovery)
		} else if vnode.Spec.ServiceDiscovery.CloudMap != nil {
			input.Spec.SetServiceDiscovery(c.buildAwsCloudMapServiceDiscovery(vnode))
		} else {
			klog.Warningf("No service discovery set for virtual node %s", vnode.Name)
		}
	}

	if vnode.Spec.Logging != nil &&
		vnode.Spec.Logging.AccessLog != nil &&
		vnode.Spec.Logging.AccessLog.File != nil {
		input.Spec.SetLogging(&appmesh.Logging{
			AccessLog: &appmesh.AccessLog{
				File: &appmesh.FileAccessLog{
					Path: aws.String(vnode.Spec.Logging.AccessLog.File.Path),
				},
			},
		})
	}

	if output, err := c.appmesh.CreateVirtualNodeWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualNode == nil {
		return nil, fmt.Errorf("virtual node %s not found", vnode.Name)
	} else {
		return &VirtualNode{
			Data: *output.VirtualNode,
		}, nil
	}
}

// UpdateVirtualNode converts the desired virtual node spec into UpdateVirtualNodeInput and calls update
// virtual node.
func (c *Cloud) UpdateVirtualNode(ctx context.Context, vnode *appmeshv1beta1.VirtualNode) (*VirtualNode, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*UpdateVirtualNodeTimeout)
	defer cancel()

	input := &appmesh.UpdateVirtualNodeInput{
		VirtualNodeName: aws.String(vnode.Name),
		MeshName:        aws.String(vnode.Spec.MeshName),
		Spec:            &appmesh.VirtualNodeSpec{},
	}

	if vnode.Spec.Listeners != nil {
		listeners := []*appmesh.Listener{}
		for _, listener := range vnode.Spec.Listeners {
			appmeshListener := &appmesh.Listener{
				PortMapping: &appmesh.PortMapping{
					Port:     &listener.PortMapping.Port,
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			}
			if listener.HealthCheck != nil {
				appmeshHealthCheck := &appmesh.HealthCheckPolicy{
					HealthyThreshold:   defaultInt64(listener.HealthCheck.HealthyThreshold, 10),
					IntervalMillis:     defaultInt64(listener.HealthCheck.IntervalMillis, 30000),
					Path:               listener.HealthCheck.Path,
					Port:               listener.HealthCheck.Port,
					Protocol:           defaultString(listener.HealthCheck.Protocol, appmeshv1beta1.PortProtocolHttp),
					TimeoutMillis:      defaultInt64(listener.HealthCheck.TimeoutMillis, 5000),
					UnhealthyThreshold: defaultInt64(listener.HealthCheck.UnhealthyThreshold, 2),
				}
				appmeshListener.HealthCheck = appmeshHealthCheck
			}
			listeners = append(listeners, appmeshListener)
		}
		input.Spec.SetListeners(listeners)
	}

	if vnode.Spec.Backends != nil {
		backends := []*appmesh.Backend{}
		for _, backend := range vnode.Spec.Backends {
			backends = append(backends, &appmesh.Backend{
				VirtualService: &appmesh.VirtualServiceBackend{
					VirtualServiceName: aws.String(backend.VirtualService.VirtualServiceName),
				},
			})
		}
		input.Spec.SetBackends(backends)
	}

	if vnode.Spec.ServiceDiscovery != nil {
		if vnode.Spec.ServiceDiscovery.Dns != nil {
			serviceDiscovery := &appmesh.ServiceDiscovery{
				Dns: &appmesh.DnsServiceDiscovery{
					Hostname: aws.String(vnode.Spec.ServiceDiscovery.Dns.HostName),
				},
			}
			input.Spec.SetServiceDiscovery(serviceDiscovery)
		} else if vnode.Spec.ServiceDiscovery.CloudMap != nil {
			input.Spec.SetServiceDiscovery(c.buildAwsCloudMapServiceDiscovery(vnode))
		} else {
			klog.Warningf("No service discovery set for virtual node %s", vnode.Name)
		}
	}

	if vnode.Spec.Logging != nil &&
		vnode.Spec.Logging.AccessLog != nil &&
		vnode.Spec.Logging.AccessLog.File != nil {
		input.Spec.SetLogging(&appmesh.Logging{
			AccessLog: &appmesh.AccessLog{
				File: &appmesh.FileAccessLog{
					Path: aws.String(vnode.Spec.Logging.AccessLog.File.Path),
				},
			},
		})
	}

	if output, err := c.appmesh.UpdateVirtualNodeWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualNode == nil {
		return nil, fmt.Errorf("virtual node %s not found", vnode.Name)
	} else {
		return &VirtualNode{
			Data: *output.VirtualNode,
		}, nil
	}
}

func (c *Cloud) DeleteVirtualNode(ctx context.Context, name string, meshName string) (*VirtualNode, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DeleteVirtualNodeTimeout)
	defer cancel()

	input := &appmesh.DeleteVirtualNodeInput{
		MeshName:        aws.String(meshName),
		VirtualNodeName: aws.String(name),
	}

	if output, err := c.appmesh.DeleteVirtualNodeWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualNode == nil {
		return nil, fmt.Errorf("virtual node %s not found", name)
	} else {
		return &VirtualNode{
			Data: *output.VirtualNode,
		}, nil
	}
}

type VirtualService struct {
	Data appmesh.VirtualServiceData
}

// Name returns the name or an empty string
func (v *VirtualService) Name() string {
	return aws.StringValue(v.Data.VirtualServiceName)
}

// VirtualRouterName returns the virtual router name or an empty string
func (v *VirtualService) VirtualRouterName() string {
	if v.Data.Spec.Provider != nil &&
		v.Data.Spec.Provider.VirtualRouter != nil &&
		v.Data.Spec.Provider.VirtualRouter.VirtualRouterName != nil {
		return aws.StringValue(v.Data.Spec.Provider.VirtualRouter.VirtualRouterName)
	}
	return ""
}

// Status returns the status or an empty string
func (v *VirtualService) Status() string {
	if v.Data.Status != nil &&
		v.Data.Status.Status != nil {
		return aws.StringValue(v.Data.Status.Status)
	}
	return ""
}

// GetVirtualService calls describe virtual service.
func (c *Cloud) GetVirtualService(ctx context.Context, name string, meshName string) (*VirtualService, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DescribeVirtualServiceTimeout)
	defer cancel()

	input := &appmesh.DescribeVirtualServiceInput{
		MeshName:           aws.String(meshName),
		VirtualServiceName: aws.String(name),
	}

	if output, err := c.appmesh.DescribeVirtualServiceWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualService == nil {
		return nil, fmt.Errorf("virtual service %s not found", name)
	} else {
		return &VirtualService{
			Data: *output.VirtualService,
		}, nil
	}
}

// CreateVirtualService converts the desired virtual service spec into CreateVirtualServiceInput and calls create
// virtual service.
func (c *Cloud) CreateVirtualService(ctx context.Context, vservice *appmeshv1beta1.VirtualService) (*VirtualService, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateVirtualServiceTimeout)
	defer cancel()

	input := &appmesh.CreateVirtualServiceInput{
		MeshName:           aws.String(vservice.Spec.MeshName),
		VirtualServiceName: aws.String(vservice.Name),
		Spec: &appmesh.VirtualServiceSpec{
			Provider: &appmesh.VirtualServiceProvider{
				// We only support virtual router providers for now
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterName: aws.String(vservice.Spec.VirtualRouter.Name),
				},
			},
		},
	}

	if output, err := c.appmesh.CreateVirtualServiceWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualService == nil {
		return nil, fmt.Errorf("virtual service %s not found", vservice.Name)
	} else {
		return &VirtualService{
			Data: *output.VirtualService,
		}, nil
	}
}

func (c *Cloud) UpdateVirtualService(ctx context.Context, vservice *appmeshv1beta1.VirtualService) (*VirtualService, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*UpdateVirtualServiceTimeout)
	defer cancel()

	input := &appmesh.UpdateVirtualServiceInput{
		MeshName:           aws.String(vservice.Spec.MeshName),
		VirtualServiceName: aws.String(vservice.Name),
		Spec: &appmesh.VirtualServiceSpec{
			Provider: &appmesh.VirtualServiceProvider{
				// We only support virtual router providers for now
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterName: aws.String(vservice.Spec.VirtualRouter.Name),
				},
			},
		},
	}

	if output, err := c.appmesh.UpdateVirtualServiceWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualService == nil {
		return nil, fmt.Errorf("virtual service %s not found", vservice.Name)
	} else {
		return &VirtualService{
			Data: *output.VirtualService,
		}, nil
	}
}

func (c *Cloud) DeleteVirtualService(ctx context.Context, name string, meshName string) (*VirtualService, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DeleteVirtualServiceTimeout)
	defer cancel()

	input := &appmesh.DeleteVirtualServiceInput{
		MeshName:           aws.String(meshName),
		VirtualServiceName: aws.String(name),
	}

	if output, err := c.appmesh.DeleteVirtualServiceWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualService == nil {
		return nil, fmt.Errorf("virtual service %s not found", name)
	} else {
		return &VirtualService{
			Data: *output.VirtualService,
		}, nil
	}
}

type VirtualRouter struct {
	Data appmesh.VirtualRouterData
}

// Name returns the name or an empty string
func (v *VirtualRouter) Name() string {
	return aws.StringValue(v.Data.VirtualRouterName)
}

// Status returns the name or an empty string
func (v *VirtualRouter) Status() string {
	if v.Data.Status != nil &&
		v.Data.Status.Status != nil {
		return aws.StringValue(v.Data.Status.Status)
	}
	return ""
}

// GetVirtualRouter calls describe virtual router.
func (c *Cloud) GetVirtualRouter(ctx context.Context, name string, meshName string) (*VirtualRouter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DescribeVirtualRouterTimeout)
	defer cancel()

	input := &appmesh.DescribeVirtualRouterInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(name),
	}

	if output, err := c.appmesh.DescribeVirtualRouterWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualRouter == nil {
		return nil, fmt.Errorf("virtual router %s not found", name)
	} else {
		return &VirtualRouter{
			Data: *output.VirtualRouter,
		}, nil
	}
}

// CreateVirtualRouter converts the desired virtual service spec into CreateVirtualServiceInput and calls create
// virtual router.
func (c *Cloud) CreateVirtualRouter(ctx context.Context, vrouter *appmeshv1beta1.VirtualRouter, meshName string) (*VirtualRouter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateVirtualRouterTimeout)
	defer cancel()

	listeners := []*appmesh.VirtualRouterListener{}
	if vrouter.Listeners != nil {
		for _, listener := range vrouter.Listeners {
			listeners = append(listeners, &appmesh.VirtualRouterListener{
				PortMapping: &appmesh.PortMapping{
					Port:     &listener.PortMapping.Port,
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			})
		}
	}

	klog.Infof("Using %d vrouter listeners to build %d input listeners", len(vrouter.Listeners), len(listeners))
	input := &appmesh.CreateVirtualRouterInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(vrouter.Name),
		Spec: &appmesh.VirtualRouterSpec{
			Listeners: listeners,
		},
	}

	if output, err := c.appmesh.CreateVirtualRouterWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualRouter == nil {
		return nil, fmt.Errorf("virtual router %s not found", vrouter.Name)
	} else {
		return &VirtualRouter{
			Data: *output.VirtualRouter,
		}, nil
	}
}

// UpdateVirtualRouter converts the desired virtual router spec into UpdateVirtualRouter calls
func (c *Cloud) UpdateVirtualRouter(ctx context.Context, vrouter *appmeshv1beta1.VirtualRouter, meshName string) (*VirtualRouter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*UpdateVirtualRouterTimeout)
	defer cancel()

	listeners := []*appmesh.VirtualRouterListener{}
	if vrouter.Listeners != nil {
		for _, listener := range vrouter.Listeners {
			listeners = append(listeners, &appmesh.VirtualRouterListener{
				PortMapping: &appmesh.PortMapping{
					Port:     &listener.PortMapping.Port,
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			})
		}
	}

	klog.Infof("Using %d vrouter listeners to build %d input listeners", len(vrouter.Listeners), len(listeners))
	input := &appmesh.UpdateVirtualRouterInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(vrouter.Name),
		Spec: &appmesh.VirtualRouterSpec{
			Listeners: listeners,
		},
	}

	if output, err := c.appmesh.UpdateVirtualRouterWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualRouter == nil {
		return nil, fmt.Errorf("virtual router %s not found", vrouter.Name)
	} else {
		return &VirtualRouter{
			Data: *output.VirtualRouter,
		}, nil
	}
}

func (c *Cloud) DeleteVirtualRouter(ctx context.Context, name string, meshName string) (*VirtualRouter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DeleteVirtualRouterTimeout)
	defer cancel()

	input := &appmesh.DeleteVirtualRouterInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(name),
	}

	if output, err := c.appmesh.DeleteVirtualRouterWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.VirtualRouter == nil {
		return nil, fmt.Errorf("virtual router %s not found", name)
	} else {
		return &VirtualRouter{
			Data: *output.VirtualRouter,
		}, nil
	}
}

type Route struct {
	Data appmesh.RouteData
}

// Name returns the name or an empty string
func (r *Route) Name() string {
	return aws.StringValue(r.Data.RouteName)
}

// Status returns the name or an empty string
func (r *Route) Status() string {
	if r.Data.Status != nil &&
		r.Data.Status.Status != nil {
		return aws.StringValue(r.Data.Status.Status)
	}
	return ""
}

// Name returns the name or an empty string
func (r *Route) Prefix() string {
	if r.Data.Spec.HttpRoute != nil &&
		r.Data.Spec.HttpRoute.Match != nil {
		return aws.StringValue(r.Data.Spec.HttpRoute.Match.Prefix)
	}
	return ""
}

// WeightedTargets converts into our API type
func (r *Route) WeightedTargets() []appmeshv1beta1.WeightedTarget {
	var targets []appmeshv1beta1.WeightedTarget
	var inputTargets []*appmesh.WeightedTarget

	if r.Data.Spec.HttpRoute != nil {
		inputTargets = r.Data.Spec.HttpRoute.Action.WeightedTargets
	} else if r.Data.Spec.TcpRoute != nil {
		inputTargets = r.Data.Spec.TcpRoute.Action.WeightedTargets
	} else if r.Data.Spec.Http2Route != nil {
		inputTargets = r.Data.Spec.Http2Route.Action.WeightedTargets
	} else if r.Data.Spec.GrpcRoute != nil {
		inputTargets = r.Data.Spec.GrpcRoute.Action.WeightedTargets
	}

	for _, t := range inputTargets {
		targets = append(targets, appmeshv1beta1.WeightedTarget{
			VirtualNodeName: aws.StringValue(t.VirtualNode),
			Weight:          aws.Int64Value(t.Weight),
		})
	}

	return targets
}

// WeightedTargetSet converts into a Set of WeightedTargets
func (r *Route) WeightedTargetSet() set.Set {
	targets := r.WeightedTargets()
	s := set.NewSet()
	for _, target := range targets {
		s.Add(target)
	}
	return s
}

func (r *Route) Http2RouteRetryPolicy() *appmeshv1beta1.HttpRetryPolicy {
	if r.Data.Spec.Http2Route == nil || r.Data.Spec.Http2Route.RetryPolicy == nil {
		return nil
	}

	inputRetryPolicy := r.Data.Spec.Http2Route.RetryPolicy

	return HttpRouteRetryPolicyHelper(inputRetryPolicy)
}

func (r *Route) HttpRouteRetryPolicy() *appmeshv1beta1.HttpRetryPolicy {
	if r.Data.Spec.HttpRoute == nil || r.Data.Spec.HttpRoute.RetryPolicy == nil {
		return nil
	}

	inputRetryPolicy := r.Data.Spec.HttpRoute.RetryPolicy

	return HttpRouteRetryPolicyHelper(inputRetryPolicy)
}

func HttpRouteRetryPolicyHelper(r *appmesh.HttpRetryPolicy) *appmeshv1beta1.HttpRetryPolicy {
	result := &appmeshv1beta1.HttpRetryPolicy{
		PerRetryTimeoutMillis: durationToMillis(r.PerRetryTimeout),
		MaxRetries:            r.MaxRetries,
	}

	for _, inputEvent := range r.HttpRetryEvents {
		resultEvent := appmeshv1beta1.HttpRetryPolicyEvent(aws.StringValue(inputEvent))
		result.HttpRetryPolicyEvents = append(result.HttpRetryPolicyEvents, resultEvent)
	}

	for _, inputEvent := range r.TcpRetryEvents {
		resultEvent := appmeshv1beta1.TcpRetryPolicyEvent(aws.StringValue(inputEvent))
		result.TcpRetryPolicyEvents = append(result.TcpRetryPolicyEvents, resultEvent)
	}

	return result
}

func (r *Route) Http2RouteMatch() *appmeshv1beta1.HttpRouteMatch {
	if r.Data.Spec.Http2Route == nil || r.Data.Spec.Http2Route.Match == nil {
		return nil
	}

	inputMatch := r.Data.Spec.Http2Route.Match

	return HttpRouteMatchHelper(inputMatch)
}

func (r *Route) HttpRouteMatch() *appmeshv1beta1.HttpRouteMatch {
	if r.Data.Spec.HttpRoute == nil || r.Data.Spec.HttpRoute.Match == nil {
		return nil
	}

	inputMatch := r.Data.Spec.HttpRoute.Match

	return HttpRouteMatchHelper(inputMatch)
}

func HttpRouteMatchHelper(m *appmesh.HttpRouteMatch) *appmeshv1beta1.HttpRouteMatch {
	resultMatch := &appmeshv1beta1.HttpRouteMatch{
		Prefix: aws.StringValue(m.Prefix),
		Method: m.Method,
		Scheme: m.Scheme,
	}

	for _, h := range m.Headers {
		resultHeader := appmeshv1beta1.HttpRouteHeader{
			Name:   aws.StringValue(h.Name),
			Invert: h.Invert,
		}
		if h.Match != nil {
			resultHeader.Match = &appmeshv1beta1.HeaderMatchMethod{
				Exact:  h.Match.Exact,
				Prefix: h.Match.Prefix,
				Suffix: h.Match.Suffix,
				Regex:  h.Match.Regex,
			}
			if h.Match.Range != nil {
				resultHeader.Match.Range = &appmeshv1beta1.MatchRange{
					Start: h.Match.Range.Start,
					End:   h.Match.Range.End,
				}
			}
		}
		resultMatch.Headers = append(resultMatch.Headers, resultHeader)
	}

	return resultMatch
}

func (r *Route) GrpcRouteMatch() *appmeshv1beta1.GrpcRouteMatch {
	if r.Data.Spec.GrpcRoute == nil || r.Data.Spec.GrpcRoute.Match == nil {
		return nil
	}

	inputMatch := r.Data.Spec.GrpcRoute.Match
	resultMatch := &appmeshv1beta1.GrpcRouteMatch{
		ServiceName: inputMatch.ServiceName,
		MethodName:  inputMatch.MethodName,
	}

	for _, m := range inputMatch.Metadata {
		resultMetadata := appmeshv1beta1.GrpcRouteMetadata{
			Name:   aws.StringValue(m.Name),
			Invert: m.Invert,
		}
		if m.Match != nil {
			resultMetadata.Match = &appmeshv1beta1.MetadataMatchMethod{
				Exact:  m.Match.Exact,
				Prefix: m.Match.Prefix,
				Suffix: m.Match.Suffix,
				Regex:  m.Match.Regex,
			}
			if m.Match.Range != nil {
				resultMetadata.Match.Range = &appmeshv1beta1.MatchRange{
					Start: m.Match.Range.Start,
					End:   m.Match.Range.End,
				}
			}
		}
		resultMatch.Metadata = append(resultMatch.Metadata, resultMetadata)
	}

	return resultMatch
}

func (r *Route) GrpcRouteRetryPolicy() *appmeshv1beta1.GrpcRetryPolicy {
	if r.Data.Spec.GrpcRoute == nil || r.Data.Spec.GrpcRoute.RetryPolicy == nil {
		return nil
	}

	input := r.Data.Spec.GrpcRoute.RetryPolicy
	result := &appmeshv1beta1.GrpcRetryPolicy{
		PerRetryTimeoutMillis: durationToMillis(input.PerRetryTimeout),
		MaxRetries:            input.MaxRetries,
	}

	for _, inputEvent := range input.HttpRetryEvents {
		resultEvent := appmeshv1beta1.HttpRetryPolicyEvent(aws.StringValue(inputEvent))
		result.HttpRetryPolicyEvents = append(result.HttpRetryPolicyEvents, resultEvent)
	}

	for _, inputEvent := range input.TcpRetryEvents {
		resultEvent := appmeshv1beta1.TcpRetryPolicyEvent(aws.StringValue(inputEvent))
		result.TcpRetryPolicyEvents = append(result.TcpRetryPolicyEvents, resultEvent)
	}

	for _, inputEvent := range input.GrpcRetryEvents {
		resultEvent := appmeshv1beta1.GrpcRetryPolicyEvent(aws.StringValue(inputEvent))
		result.GrpcRetryPolicyEvents = append(result.GrpcRetryPolicyEvents, resultEvent)
	}

	return result
}

type Routes []Route

func (r Routes) RouteNamesSet() set.Set {
	s := set.NewSet()
	for _, route := range r {
		s.Add(route.Name())
	}
	return s
}

func (r Routes) RouteByName(name string) Route {
	for _, route := range r {
		if route.Name() == name {
			return route
		}
	}
	return Route{
		Data: appmesh.RouteData{},
	}
}

// GetRoute calls describe route.
func (c *Cloud) GetRoute(ctx context.Context, name string, routerName string, meshName string) (*Route, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DescribeRouteTimeout)
	defer cancel()

	input := &appmesh.DescribeRouteInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(routerName),
		RouteName:         aws.String(name),
	}

	if output, err := c.appmesh.DescribeRouteWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Route == nil {
		return nil, fmt.Errorf("route %s not found", name)
	} else {
		return &Route{
			Data: *output.Route,
		}, nil
	}
}

// CreateRoute converts the desired virtual service spec into CreateVirtualServiceInput and calls create route.
func (c *Cloud) CreateRoute(ctx context.Context, route *appmeshv1beta1.Route, routerName string, meshName string) (*Route, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateRouteTimeout)
	defer cancel()

	input := &appmesh.CreateRouteInput{
		MeshName:          aws.String(meshName),
		RouteName:         aws.String(route.Name),
		VirtualRouterName: aws.String(routerName),
		Spec:              c.buildRouteSpec(route),
	}

	if output, err := c.appmesh.CreateRouteWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Route == nil {
		return nil, fmt.Errorf("route %s not found", route.Name)
	} else {
		return &Route{
			Data: *output.Route,
		}, nil
	}
}

func (c *Cloud) GetRoutesForVirtualRouter(ctx context.Context, routerName string, meshName string) (Routes, error) {
	listctx, cancel := context.WithTimeout(ctx, time.Second*ListRoutesTimeout)
	defer cancel()

	input := &appmesh.ListRoutesInput{
		MeshName:          aws.String(meshName),
		VirtualRouterName: aws.String(routerName),
	}

	if output, err := c.appmesh.ListRoutesWithContext(listctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Routes == nil {
		return nil, fmt.Errorf("routes not found")
	} else {
		routes := Routes{}
		for _, ref := range output.Routes {
			route, err := c.GetRoute(ctx, aws.StringValue(ref.RouteName), aws.StringValue(ref.VirtualRouterName), aws.StringValue(ref.MeshName))
			if err != nil {
				if !IsAWSErrNotFound(err) {
					klog.Errorf("error describing route: %s", err)
				}
				continue
			}
			routes = append(routes, Route{
				Data: route.Data,
			})
		}
		return routes, nil
	}

}

// UpdateRoute converts the desired virtual service spec into UpdateRouteInput and calls update route.
func (c *Cloud) UpdateRoute(ctx context.Context, route *appmeshv1beta1.Route, routerName string, meshName string) (*Route, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*UpdateRouteTimeout)
	defer cancel()

	input := &appmesh.UpdateRouteInput{
		MeshName:          aws.String(meshName),
		RouteName:         aws.String(route.Name),
		VirtualRouterName: aws.String(routerName),
		Spec:              c.buildRouteSpec(route),
	}

	if output, err := c.appmesh.UpdateRouteWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Route == nil {
		return nil, fmt.Errorf("route %s not found", route.Name)
	} else {
		return &Route{
			Data: *output.Route,
		}, nil
	}
}

func (c *Cloud) DeleteRoute(ctx context.Context, name string, routerName string, meshName string) (*Route, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*DeleteRouteTimeout)
	defer cancel()

	input := &appmesh.DeleteRouteInput{
		RouteName:         aws.String(name),
		VirtualRouterName: aws.String(routerName),
		MeshName:          aws.String(meshName),
	}

	if output, err := c.appmesh.DeleteRouteWithContext(ctx, input); err != nil {
		return nil, err
	} else if output == nil || output.Route == nil {
		return nil, fmt.Errorf("route %s not found", name)
	} else {
		return &Route{
			Data: *output.Route,
		}, nil
	}
}

func (c *Cloud) buildAwsCloudMapServiceDiscovery(vnode *appmeshv1beta1.VirtualNode) *appmesh.ServiceDiscovery {
	attr := []*appmesh.AwsCloudMapInstanceAttribute{}

	//adding attributes defined by customer
	for k, v := range vnode.Spec.ServiceDiscovery.CloudMap.Attributes {
		attr = append(attr, &appmesh.AwsCloudMapInstanceAttribute{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	return &appmesh.ServiceDiscovery{
		AwsCloudMap: &appmesh.AwsCloudMapServiceDiscovery{
			NamespaceName: aws.String(vnode.Spec.ServiceDiscovery.CloudMap.NamespaceName),
			ServiceName:   aws.String(vnode.Spec.ServiceDiscovery.CloudMap.ServiceName),
			Attributes:    attr,
		},
	}
}

func IsAWSErrNotFound(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == appmesh.ErrCodeNotFoundException {
				return true
			}
		}
	}
	return false
}

func IsAWSErrResourceInUse(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == appmesh.ErrCodeResourceInUseException {
				return true
			}
		}
	}
	return false
}

func (c *Cloud) buildRouteSpec(route *appmeshv1beta1.Route) *appmesh.RouteSpec {
	if route == nil {
		return nil
	}

	if route.Http != nil {
		return &appmesh.RouteSpec{
			Priority: route.Priority,
			HttpRoute: &appmesh.HttpRoute{
				Match: c.buildHttpRouteMatch(route.Http.Match),
				Action: &appmesh.HttpRouteAction{
					WeightedTargets: c.buildWeightedTargets(route.Http.Action.WeightedTargets),
				},
				RetryPolicy: c.buildHttpRetryPolicy(route.Http.RetryPolicy),
			},
		}
	}

	if route.Tcp != nil {
		return &appmesh.RouteSpec{
			Priority: route.Priority,
			TcpRoute: &appmesh.TcpRoute{
				Action: &appmesh.TcpRouteAction{
					WeightedTargets: c.buildWeightedTargets(route.Tcp.Action.WeightedTargets),
				},
			},
		}
	}

	if route.Http2 != nil {
		return &appmesh.RouteSpec{
			Priority: route.Priority,
			Http2Route: &appmesh.HttpRoute{
				Match: c.buildHttpRouteMatch(route.Http2.Match),
				Action: &appmesh.HttpRouteAction{
					WeightedTargets: c.buildWeightedTargets(route.Http2.Action.WeightedTargets),
				},
				RetryPolicy: c.buildHttpRetryPolicy(route.Http2.RetryPolicy),
			},
		}
	}

	if route.Grpc != nil {
		return &appmesh.RouteSpec{
			Priority: route.Priority,
			GrpcRoute: &appmesh.GrpcRoute{
				Match: c.buildGrpcRouteMatch(route.Grpc.Match),
				Action: &appmesh.GrpcRouteAction{
					WeightedTargets: c.buildWeightedTargets(route.Grpc.Action.WeightedTargets),
				},
				RetryPolicy: c.buildGrpcRetryPolicy(route.Grpc.RetryPolicy),
			},
		}
	}

	return nil
}

func (c *Cloud) buildWeightedTargets(input []appmeshv1beta1.WeightedTarget) []*appmesh.WeightedTarget {
	targets := []*appmesh.WeightedTarget{}
	for _, target := range input {
		weight := target.Weight
		targets = append(targets, &appmesh.WeightedTarget{
			VirtualNode: aws.String(target.VirtualNodeName),
			Weight:      aws.Int64(weight),
		})
	}
	return targets
}

func (c *Cloud) buildHttpRouteMatch(input appmeshv1beta1.HttpRouteMatch) *appmesh.HttpRouteMatch {
	appmeshRouteMatch := &appmesh.HttpRouteMatch{
		Prefix: aws.String(input.Prefix),
		Method: input.Method,
		Scheme: input.Scheme,
	}

	if len(input.Headers) > 0 {
		appmeshRouteMatch.Headers = []*appmesh.HttpRouteHeader{}
		for _, h := range input.Headers {
			appmeshRouteMatch.Headers = append(appmeshRouteMatch.Headers, c.buildHttpRouteHeader(h))
		}
	}

	return appmeshRouteMatch
}

func (c *Cloud) buildHttpRouteHeader(input appmeshv1beta1.HttpRouteHeader) *appmesh.HttpRouteHeader {
	appmeshHeader := &appmesh.HttpRouteHeader{
		Name:   aws.String(input.Name),
		Invert: input.Invert,
	}
	if input.Match != nil {
		appmeshHeader.Match = &appmesh.HeaderMatchMethod{
			Exact:  input.Match.Exact,
			Prefix: input.Match.Prefix,
			Regex:  input.Match.Regex,
			Suffix: input.Match.Suffix,
		}
		if input.Match.Range != nil {
			appmeshHeader.Match.Range = &appmesh.MatchRange{
				Start: input.Match.Range.Start,
				End:   input.Match.Range.End,
			}
			klog.Infof("Range = %+v", appmeshHeader.Match.Range)
		}
	}

	return appmeshHeader
}

func (c *Cloud) buildHttpRetryPolicy(input *appmeshv1beta1.HttpRetryPolicy) *appmesh.HttpRetryPolicy {
	if input == nil {
		return nil
	}

	appmeshRetryPolicy := &appmesh.HttpRetryPolicy{
		MaxRetries: input.MaxRetries,
	}

	if input.PerRetryTimeoutMillis != nil {
		appmeshRetryPolicy.PerRetryTimeout = &appmesh.Duration{
			Unit:  aws.String(appmesh.DurationUnitMs),
			Value: input.PerRetryTimeoutMillis,
		}
	}

	for _, inputEvent := range input.HttpRetryPolicyEvents {
		appmeshRetryPolicy.HttpRetryEvents = append(appmeshRetryPolicy.HttpRetryEvents, aws.String(string(inputEvent)))
	}

	for _, inputEvent := range input.TcpRetryPolicyEvents {
		appmeshRetryPolicy.TcpRetryEvents = append(appmeshRetryPolicy.TcpRetryEvents, aws.String(string(inputEvent)))
	}

	return appmeshRetryPolicy
}

func (c *Cloud) buildGrpcRetryPolicy(input *appmeshv1beta1.GrpcRetryPolicy) *appmesh.GrpcRetryPolicy {
	if input == nil {
		return nil
	}

	appmeshRetryPolicy := &appmesh.GrpcRetryPolicy{
		MaxRetries: input.MaxRetries,
	}

	if input.PerRetryTimeoutMillis != nil {
		appmeshRetryPolicy.PerRetryTimeout = &appmesh.Duration{
			Unit:  aws.String(appmesh.DurationUnitMs),
			Value: input.PerRetryTimeoutMillis,
		}
	}

	for _, inputEvent := range input.HttpRetryPolicyEvents {
		appmeshRetryPolicy.HttpRetryEvents = append(appmeshRetryPolicy.HttpRetryEvents, aws.String(string(inputEvent)))
	}

	for _, inputEvent := range input.TcpRetryPolicyEvents {
		appmeshRetryPolicy.TcpRetryEvents = append(appmeshRetryPolicy.TcpRetryEvents, aws.String(string(inputEvent)))
	}

	for _, inputEvent := range input.GrpcRetryPolicyEvents {
		appmeshRetryPolicy.GrpcRetryEvents = append(appmeshRetryPolicy.GrpcRetryEvents, aws.String(string(inputEvent)))
	}

	return appmeshRetryPolicy
}

func (c *Cloud) buildGrpcRouteMatch(input appmeshv1beta1.GrpcRouteMatch) *appmesh.GrpcRouteMatch {
	appmeshRouteMatch := &appmesh.GrpcRouteMatch{
		ServiceName: input.ServiceName,
		MethodName:  input.MethodName,
	}

	if len(input.Metadata) > 0 {
		appmeshRouteMatch.Metadata = []*appmesh.GrpcRouteMetadata{}
		for _, m := range input.Metadata {
			appmeshRouteMatch.Metadata = append(appmeshRouteMatch.Metadata, c.buildGrpcRouteMetadata(m))
		}
	}

	return appmeshRouteMatch
}

func (c *Cloud) buildGrpcRouteMetadata(input appmeshv1beta1.GrpcRouteMetadata) *appmesh.GrpcRouteMetadata {
	appmeshMetadata := &appmesh.GrpcRouteMetadata{
		Name:   aws.String(input.Name),
		Invert: input.Invert,
	}
	if input.Match != nil {
		appmeshMetadata.Match = &appmesh.GrpcRouteMetadataMatchMethod{
			Exact:  input.Match.Exact,
			Prefix: input.Match.Prefix,
			Regex:  input.Match.Regex,
			Suffix: input.Match.Suffix,
		}
		if input.Match.Range != nil {
			appmeshMetadata.Match.Range = &appmesh.MatchRange{
				Start: input.Match.Range.Start,
				End:   input.Match.Range.End,
			}
			klog.Infof("Range = %+v", appmeshMetadata.Match.Range)
		}
	}

	return appmeshMetadata
}

func defaultInt64(v *int64, defaultVal int64) *int64 {
	if v != nil {
		return v
	}
	return aws.Int64(defaultVal)
}

func defaultString(v *string, defaultVal string) *string {
	if v != nil {
		return v
	}
	return aws.String(defaultVal)
}

func durationToMillis(d *appmesh.Duration) *int64 {
	if d == nil {
		return nil
	}

	switch aws.StringValue(d.Unit) {
	case appmesh.DurationUnitMs:
		return d.Value
	case appmesh.DurationUnitS:
		return aws.Int64(aws.Int64Value(d.Value) * 1000)
	}

	return nil
}
