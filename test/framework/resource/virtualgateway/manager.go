package virtualgateway

import (
	"context"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilVirtualGatewayActive(ctx context.Context, vg *appmesh.VirtualGateway) (*appmesh.VirtualGateway, error)
	WaitUntilVirtualGatewayDeleted(ctx context.Context, vg *appmesh.VirtualGateway) error
	CheckVirtualGatewayInAWS(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) error
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

func (m *defaultManager) WaitUntilVirtualGatewayActive(ctx context.Context, vg *appmesh.VirtualGateway) (*appmesh.VirtualGateway, error) {
	observedVG := &appmesh.VirtualGateway{}
	retryCount := 0
	return observedVG, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		err := m.k8sClient.Get(ctx, k8s.NamespacedName(vg), observedVG)
		if err != nil {
			if retryCount >= utils.PollRetries {
				return false, err
			}
			retryCount++
			return false, nil
		}

		for _, condition := range observedVG.Status.Conditions {
			if condition.Type == appmesh.VirtualGatewayActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualGatewayDeleted(ctx context.Context, vg *appmesh.VirtualGateway) error {
	observedVG := &appmesh.VirtualGateway{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vg), observedVG); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
func (m *defaultManager) CheckVirtualGatewayInAWS(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) error {
	// TODO: handle aws throttling
	desiredSDKVGSpec, err := virtualgateway.BuildSDKVirtualGatewaySpec(ctx, vg)
	if err != nil {
		return err
	}

	resp, err := m.appMeshSDK.DescribeVirtualGatewayWithContext(ctx, &appmeshsdk.DescribeVirtualGatewayInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualGatewayName: vg.Spec.AWSName,
	})
	if err != nil {
		return err
	}

	opts := equality.CompareOptionForVirtualGatewaySpec()
	if !cmp.Equal(desiredSDKVGSpec, resp.VirtualGateway.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKVGSpec, resp.VirtualGateway.Spec, opts))
	}
	return nil
}
