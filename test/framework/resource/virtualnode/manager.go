package virtualnode

import (
	"context"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilVirtualNodeActive(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.VirtualNode, error)
	WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmesh.VirtualNode) error
	CheckVirtualNodeInAWS(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error
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

func (m *defaultManager) WaitUntilVirtualNodeActive(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.VirtualNode, error) {
	observedVN := &appmesh.VirtualNode{}
	return observedVN, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		// sometimes there's a delay in the resource showing up
		for i := 0; i < 5; i++ {
			if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vn), observedVN); err != nil {
				if i >= 5 {
					return false, err
				}
			}
			time.Sleep(100 * time.Millisecond)
		}

		for _, condition := range observedVN.Status.Conditions {
			if condition.Type == appmesh.VirtualNodeActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmesh.VirtualNode) error {
	observedVN := &appmesh.VirtualNode{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vn), observedVN); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) CheckVirtualNodeInAWS(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error {
	// TODO: handle aws throttling
	vsByKey := make(map[types.NamespacedName]*appmesh.VirtualService)
	vsRefs := virtualnode.ExtractVirtualServiceReferences(vn)
	for _, vsRef := range vsRefs {
		vs := &appmesh.VirtualService{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualServiceReference(vn, vsRef), vs); err != nil {
			return err
		}
		vsByKey[k8s.NamespacedName(vs)] = vs
	}
	desiredSDKVNSpec, err := virtualnode.BuildSDKVirtualNodeSpec(vn, vsByKey)
	if err != nil {
		return err
	}
	resp, err := m.appMeshSDK.DescribeVirtualNodeWithContext(ctx, &appmeshsdk.DescribeVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		return err
	}
	opts := equality.CompareOptionForVirtualNodeSpec()
	if !cmp.Equal(desiredSDKVNSpec, resp.VirtualNode.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKVNSpec, resp.VirtualNode.Spec, opts))
	}
	return nil
}
