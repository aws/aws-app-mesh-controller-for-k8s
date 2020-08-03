package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Manager interface {
	WaitUntilVirtualRouterActive(ctx context.Context, vr *appmesh.VirtualRouter) (*appmesh.VirtualRouter, error)
	WaitUntilVirtualRouterDeleted(ctx context.Context, vr *appmesh.VirtualRouter) error
	CheckVirtualRouterInAWS(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) error
}

func NewManager(k8sClient client.Client, appMeshSDK services.AppMesh) Manager {
	return &defaultManager{
		k8sClient:  k8sClient,
		appMeshSDK: appMeshSDK,
	}
}

type defaultManager struct {
	k8sClient  client.Client
	appMeshSDK services.AppMesh
}

func (m *defaultManager) WaitUntilVirtualRouterActive(ctx context.Context, vr *appmesh.VirtualRouter) (*appmesh.VirtualRouter, error) {
	observedVR := &appmesh.VirtualRouter{}
	return observedVR, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		// sometimes there's a delay in the resource showing up
		for i := 0; i < 5; i++ {
			if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vr), observedVR); err != nil {
				if i >= 5 {
					return false, err
				}
			}
			time.Sleep(100 * time.Millisecond)
		}

		for _, condition := range observedVR.Status.Conditions {
			if condition.Type == appmesh.VirtualRouterActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualRouterDeleted(ctx context.Context, vr *appmesh.VirtualRouter) error {
	observedVR := &appmesh.VirtualRouter{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vr), observedVR); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) CheckVirtualRouterInAWS(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) error {
	// TODO: handle aws throttling
	desiredSDKVRSpec, err := virtualrouter.BuildSDKVirtualRouterSpec(vr)
	if err != nil {
		return err
	}
	resp, err := m.appMeshSDK.DescribeVirtualRouterWithContext(ctx, &appmeshsdk.DescribeVirtualRouterInput{
		MeshName:          ms.Spec.AWSName,
		MeshOwner:         ms.Spec.MeshOwner,
		VirtualRouterName: vr.Spec.AWSName,
	})
	if err != nil {
		return err
	}
	opts := cmpopts.EquateEmpty()
	if !cmp.Equal(desiredSDKVRSpec, resp.VirtualRouter.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKVRSpec, resp.VirtualRouter.Spec, opts))
	}

	vnByKey := make(map[types.NamespacedName]*appmesh.VirtualNode)
	vnRefs := virtualrouter.ExtractVirtualNodeReferences(vr)
	for _, vnRef := range vnRefs {
		vn := &appmesh.VirtualNode{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualNodeReference(vr, vnRef), vn); err != nil {
			return err
		}
		vnByKey[k8s.NamespacedName(vn)] = vn
	}

	for _, route := range vr.Spec.Routes {
		desiredRouteSpec, err := virtualrouter.BuildSDKRouteSpec(vr, route, vnByKey)
		if err != nil {
			return err
		}
		resp, err := m.appMeshSDK.DescribeRouteWithContext(ctx, &appmeshsdk.DescribeRouteInput{
			MeshName:          ms.Spec.AWSName,
			MeshOwner:         ms.Spec.MeshOwner,
			VirtualRouterName: vr.Spec.AWSName,
			RouteName:         aws.String(route.Name),
		})
		if err != nil {
			return err
		}
		if !cmp.Equal(desiredRouteSpec, resp.Route.Spec, opts) {
			return errors.New(cmp.Diff(desiredRouteSpec, resp.Route.Spec, opts))
		}
	}
	return nil
}
