package gatewayroute

import (
	"context"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/gatewayroute"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
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
	WaitUntilGatewayRouteActive(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.GatewayRoute, error)
	WaitUntilGatewayRouteDeleted(ctx context.Context, gr *appmesh.GatewayRoute) error
	CheckGatewayRouteInAWS(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute) error
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

func (m *defaultManager) WaitUntilGatewayRouteActive(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.GatewayRoute, error) {
	observedGR := &appmesh.GatewayRoute{}
	retryCount := 0
	return observedGR, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		err := m.k8sClient.Get(ctx, k8s.NamespacedName(gr), observedGR)
		if err != nil {
			if retryCount >= utils.PollRetries {
				return false, err
			}
			retryCount++
			return false, nil
		}

		for _, condition := range observedGR.Status.Conditions {
			if condition.Type == appmesh.GatewayRouteActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilGatewayRouteDeleted(ctx context.Context, gr *appmesh.GatewayRoute) error {
	observedGR := &appmesh.GatewayRoute{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(gr), observedGR); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
func (m *defaultManager) CheckGatewayRouteInAWS(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute) error {
	// TODO: handle aws throttling
	vsByKey := make(map[types.NamespacedName]*appmesh.VirtualService)
	vsRefs := gatewayroute.ExtractVirtualServiceReferences(gr)
	for _, vsRef := range vsRefs {
		vs := &appmesh.VirtualService{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualServiceReference(gr, vsRef), vs); err != nil {
			return err
		}
		vsByKey[k8s.NamespacedName(vs)] = vs
	}

	desiredSDKGRSpec, err := gatewayroute.BuildSDKGatewayRouteSpec(ctx, gr, vsByKey)
	if err != nil {
		return err
	}

	resp, err := m.appMeshSDK.DescribeGatewayRouteWithContext(ctx, &appmeshsdk.DescribeGatewayRouteInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		GatewayRouteName:   gr.Spec.AWSName,
		VirtualGatewayName: vg.Spec.AWSName,
	})
	if err != nil {
		return err
	}

	opts := cmpopts.EquateEmpty()
	if !cmp.Equal(desiredSDKGRSpec, resp.GatewayRoute.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKGRSpec, resp.GatewayRoute.Spec, opts))
	}
	return nil
}
