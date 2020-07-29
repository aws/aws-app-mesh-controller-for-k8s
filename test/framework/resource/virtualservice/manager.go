package virtualservice

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilVirtualServiceActive(ctx context.Context, vs *appmesh.VirtualService) (*appmesh.VirtualService, error)
	WaitUntilVirtualServiceDeleted(ctx context.Context, vs *appmesh.VirtualService) error
	CheckVirtualServiceInAWS(ctx context.Context, ms *appmesh.Mesh, vs *appmesh.VirtualService) error
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

func (m *defaultManager) WaitUntilVirtualServiceActive(ctx context.Context, vs *appmesh.VirtualService) (*appmesh.VirtualService, error) {
	observedVS := &appmesh.VirtualService{}
	return observedVS, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vs), observedVS); err != nil {
			return false, err
		}

		for _, condition := range observedVS.Status.Conditions {
			if condition.Type == appmesh.VirtualServiceActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualServiceDeleted(ctx context.Context, vs *appmesh.VirtualService) error {
	observedVS := &appmesh.VirtualService{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vs), observedVS); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
func (m *defaultManager) CheckVirtualServiceInAWS(ctx context.Context, ms *appmesh.Mesh, vs *appmesh.VirtualService) error {
	// TODO: handle aws throttling
	vnByKey := make(map[types.NamespacedName]*appmesh.VirtualNode)
	vnRefs := virtualservice.ExtractVirtualNodeReferences(vs)
	for _, vnRef := range vnRefs {
		vn := &appmesh.VirtualNode{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualNodeReference(vs, vnRef), vn); err != nil {
			return err
		}
		vnByKey[k8s.NamespacedName(vn)] = vn
	}
	vrByKey := make(map[types.NamespacedName]*appmesh.VirtualRouter)
	vrRefs := virtualservice.ExtractVirtualRouterReferences(vs)
	for _, vrRef := range vrRefs {
		vr := &appmesh.VirtualRouter{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualRouterReference(vs, vrRef), vr); err != nil {
			return err
		}
		vrByKey[k8s.NamespacedName(vr)] = vr
	}
	desiredSDKVSSpec, err := virtualservice.BuildSDKVirtualServiceSpec(vs, vnByKey, vrByKey)
	if err != nil {
		return err
	}
	resp, err := m.appMeshSDK.DescribeVirtualServiceWithContext(ctx, &appmeshsdk.DescribeVirtualServiceInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualServiceName: vs.Spec.AWSName,
	})
	if err != nil {
		return err
	}
	opts := cmpopts.EquateEmpty()
	if !cmp.Equal(desiredSDKVSSpec, resp.VirtualService.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKVSSpec, resp.VirtualService.Spec, opts))
	}
	return nil
}
