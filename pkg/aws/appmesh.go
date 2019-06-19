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
	DescribeMeshTimeout           = 5
	CreateMeshTimeout             = 5
	DeleteMeshTimeout             = 5
	DescribeVirtualNodeTimeout    = 5
	CreateVirtualNodeTimeout      = 5
	UpdateVirtualNodeTimeout      = 5
	DeleteVirtualNodeTimeout      = 5
	DescribeVirtualServiceTimeout = 5
	CreateVirtualServiceTimeout   = 5
	UpdateVirtualServiceTimeout   = 5
	DeleteVirtualServiceTimeout   = 5
	DescribeVirtualRouterTimeout  = 5
	CreateVirtualRouterTimeout    = 5
	UpdateVirtualRouterTimeout    = 5
	DeleteVirtualRouterTimeout    = 5
	DescribeRouteTimeout          = 5
	CreateRouteTimeout            = 5
	ListRoutesTimeout             = 10
	UpdateRouteTimeout            = 5
	DeleteRouteTimeout            = 5
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
	for _, l := range v.Data.Spec.Listeners {
		listeners = append(listeners, appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     aws.Int64Value(l.PortMapping.Port),
				Protocol: aws.StringValue(l.PortMapping.Protocol),
			},
		})
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
			listeners = append(listeners, &appmesh.Listener{
				PortMapping: &appmesh.PortMapping{
					Port:     &listener.PortMapping.Port,
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			})
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
			// TODO(nic) add CloudMap Service Discovery when SDK supports it
		} else {
			klog.Warningf("No service discovery set for virtual node %s", vnode.Name)
		}
	}

	if vnode.Spec.Logging != nil {
		if vnode.Spec.Logging.AccessLog != nil {
			if vnode.Spec.Logging.AccessLog.File != nil {
				input.Spec.SetLogging(&appmesh.Logging{
					AccessLog: &appmesh.AccessLog{
						File: &appmesh.FileAccessLog{
							Path: aws.String(vnode.Spec.Logging.AccessLog.File.Path),
						},
					},
				})
			}
		}
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
			listeners = append(listeners, &appmesh.Listener{
				PortMapping: &appmesh.PortMapping{
					Port:     &listener.PortMapping.Port,
					Protocol: aws.String(listener.PortMapping.Protocol),
				},
			})
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
			// TODO(nic) add CloudMap Service Discovery when SDK supports it
		} else {
			klog.Warningf("No service discovery set for virtual node %s", vnode.Name)
		}
	}

	if vnode.Spec.Logging != nil {
		if vnode.Spec.Logging.AccessLog != nil {
			if vnode.Spec.Logging.AccessLog.File != nil {
				input.Spec.SetLogging(&appmesh.Logging{
					AccessLog: &appmesh.AccessLog{
						File: &appmesh.FileAccessLog{
							Path: aws.String(vnode.Spec.Logging.AccessLog.File.Path),
						},
					},
				})
			}
		}
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
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{},
			},
		},
	}

	if vservice.Spec.VirtualRouter != nil {
		input.Spec.Provider.VirtualRouter.VirtualRouterName = aws.String(vservice.Spec.VirtualRouter.Name)
	} else {
		// We default to a virtual router with the same name as the virtual service
		input.Spec.Provider.VirtualRouter.VirtualRouterName = aws.String(vservice.Name)
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
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{},
			},
		},
	}

	if vservice.Spec.VirtualRouter != nil {
		input.Spec.Provider.VirtualRouter.VirtualRouterName = aws.String(vservice.Spec.VirtualRouter.Name)
	} else {
		// We default to a virtual router with the same name as the virtual service
		input.Spec.Provider.VirtualRouter.VirtualRouterName = aws.String(vservice.Name)
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
	if r.Data.Spec.HttpRoute.Action.WeightedTargets == nil {
		return []appmeshv1beta1.WeightedTarget{}
	}

	var targets = []appmeshv1beta1.WeightedTarget{}
	for _, t := range r.Data.Spec.HttpRoute.Action.WeightedTargets {
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
		Spec: &appmesh.RouteSpec{
			HttpRoute: &appmesh.HttpRoute{
				Match: &appmesh.HttpRouteMatch{
					Prefix: aws.String(route.Http.Match.Prefix),
				},
				Action: &appmesh.HttpRouteAction{},
			},
		},
	}

	targets := []*appmesh.WeightedTarget{}
	for _, target := range route.Http.Action.WeightedTargets {
		weight := target.Weight
		targets = append(targets, &appmesh.WeightedTarget{
			VirtualNode: aws.String(target.VirtualNodeName),
			Weight:      aws.Int64(weight),
		})
	}

	input.Spec.HttpRoute.Action.SetWeightedTargets(targets)

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
		Spec: &appmesh.RouteSpec{
			HttpRoute: &appmesh.HttpRoute{
				Match: &appmesh.HttpRouteMatch{
					Prefix: aws.String(route.Http.Match.Prefix),
				},
				Action: &appmesh.HttpRouteAction{},
			},
		},
	}

	targets := []*appmesh.WeightedTarget{}
	for _, target := range route.Http.Action.WeightedTargets {
		weight := target.Weight
		targets = append(targets, &appmesh.WeightedTarget{
			VirtualNode: aws.String(target.VirtualNodeName),
			Weight:      aws.Int64(weight),
		})
	}

	input.Spec.HttpRoute.Action.SetWeightedTargets(targets)

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
