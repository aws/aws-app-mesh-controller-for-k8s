package virtualgateway

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceManager is dedicated to manage AppMesh VirtualGateway resources for k8s VirtualGateway CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh VirtualGateway to match vg.spec, and update vg.status
	Reconcile(ctx context.Context, vg *appmesh.VirtualGateway) error

	// Cleanup will delete AppMesh VirtualGateway created for vg.
	Cleanup(ctx context.Context, vg *appmesh.VirtualGateway) error
}

func NewDefaultResourceManager(
	k8sClient client.Client,
	appMeshSDK services.AppMesh,
	referencesResolver references.Resolver,
	accountID string,
	log logr.Logger) ResourceManager {

	return &defaultResourceManager{
		k8sClient:          k8sClient,
		appMeshSDK:         appMeshSDK,
		referencesResolver: referencesResolver,
		accountID:          accountID,
		log:                log,
	}
}

// defaultResourceManager implements ResourceManager
type defaultResourceManager struct {
	k8sClient          client.Client
	appMeshSDK         services.AppMesh
	referencesResolver references.Resolver
	accountID          string
	log                logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vg *appmesh.VirtualGateway) error {
	ms, err := m.findMeshDependency(ctx, vg)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}

	sdkVG, err := m.findSDKVirtualGateway(ctx, ms, vg)
	if err != nil {
		return err
	}
	if sdkVG == nil {
		sdkVG, err = m.createSDKVirtualGateway(ctx, ms, vg)
		if err != nil {
			return err
		}
	} else {
		sdkVG, err = m.updateSDKVirtualGateway(ctx, sdkVG, ms, vg)
		if err != nil {
			return err
		}
	}

	return m.updateCRDVirtualGateway(ctx, vg, sdkVG)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vg *appmesh.VirtualGateway) error {
	ms, err := m.findMeshDependency(ctx, vg)
	if err != nil {
		return err
	}
	sdkVG, err := m.findSDKVirtualGateway(ctx, ms, vg)
	if err != nil {
		if vg.Status.VirtualGatewayARN == nil {
			return nil
		}
		return err
	}
	if sdkVG == nil {
		return nil
	}

	return m.deleteSDKVirtualGateway(ctx, sdkVG, ms, vg)
}

// findMeshDependency find the Mesh dependency for this virtualGateway.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vg *appmesh.VirtualGateway) (*appmesh.Mesh, error) {
	if vg.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *vg.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
}

// validateMeshDependencies validate the Mesh dependency for this virtualGateway.
func (m *defaultResourceManager) validateMeshDependencies(ctx context.Context, ms *appmesh.Mesh) error {
	if !mesh.IsMeshActive(ms) {
		return runtime.NewRequeueError(errors.New("mesh is not active yet"))
	}
	return nil
}

func (m *defaultResourceManager) findSDKVirtualGateway(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) (*appmeshsdk.VirtualGatewayData, error) {
	resp, err := m.appMeshSDK.DescribeVirtualGatewayWithContext(ctx, &appmeshsdk.DescribeVirtualGatewayInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualGatewayName: vg.Spec.AWSName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}

	return resp.VirtualGateway, nil
}

func (m *defaultResourceManager) createSDKVirtualGateway(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) (*appmeshsdk.VirtualGatewayData, error) {
	sdkVGSpec, err := BuildSDKVirtualGatewaySpec(ctx, vg)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateVirtualGatewayWithContext(ctx, &appmeshsdk.CreateVirtualGatewayInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		Spec:               sdkVGSpec,
		VirtualGatewayName: vg.Spec.AWSName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualGateway, nil
}

func (m *defaultResourceManager) updateSDKVirtualGateway(ctx context.Context, sdkVG *appmeshsdk.VirtualGatewayData, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) (*appmeshsdk.VirtualGatewayData, error) {
	actualSDKVGSpec := sdkVG.Spec
	desiredSDKVGSpec, err := BuildSDKVirtualGatewaySpec(ctx, vg)
	if err != nil {
		return nil, err
	}

	opts := equality.CompareOptionForVirtualGatewaySpec()
	if cmp.Equal(desiredSDKVGSpec, actualSDKVGSpec, opts) {
		return sdkVG, nil
	}
	if !m.isSDKVirtualGatewayControlledByCRDVirtualGateway(ctx, sdkVG, vg) {
		m.log.V(2).Info("skip virtualGateway update since it's not controlled",
			"virtualGateway", k8s.NamespacedName(vg),
			"virtualGatewayARN", aws.StringValue(sdkVG.Metadata.Arn),
		)
		return sdkVG, nil
	}

	diff := cmp.Diff(desiredSDKVGSpec, actualSDKVGSpec, opts)
	m.log.V(2).Info("virtualGatewaySpec changed",
		"virtualGateway", k8s.NamespacedName(vg),
		"actualSDKVGSpec", actualSDKVGSpec,
		"desiredSDKVGSpec", desiredSDKVGSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateVirtualGatewayWithContext(ctx, &appmeshsdk.UpdateVirtualGatewayInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		Spec:               desiredSDKVGSpec,
		VirtualGatewayName: sdkVG.VirtualGatewayName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualGateway, nil
}

func (m *defaultResourceManager) deleteSDKVirtualGateway(ctx context.Context, sdkVG *appmeshsdk.VirtualGatewayData, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) error {
	if !m.isSDKVirtualGatewayOwnedByCRDVirtualGateway(ctx, sdkVG, vg) {
		m.log.V(2).Info("skip mesh virtualGateway since its not owned",
			"virtualGateway", k8s.NamespacedName(vg),
			"virtualGatewayARN", aws.StringValue(sdkVG.Metadata.Arn),
		)
		return nil
	}

	_, err := m.appMeshSDK.DeleteVirtualGatewayWithContext(ctx, &appmeshsdk.DeleteVirtualGatewayInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualGatewayName: sdkVG.VirtualGatewayName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDVirtualGateway(ctx context.Context, vg *appmesh.VirtualGateway, sdkVG *appmeshsdk.VirtualGatewayData) error {
	oldVG := vg.DeepCopy()
	needsUpdate := false
	if aws.StringValue(vg.Status.VirtualGatewayARN) != aws.StringValue(sdkVG.Metadata.Arn) {
		vg.Status.VirtualGatewayARN = sdkVG.Metadata.Arn
		needsUpdate = true
	}

	vgActiveConditionStatus := corev1.ConditionFalse
	if sdkVG.Status != nil && aws.StringValue(sdkVG.Status.Status) == appmeshsdk.VirtualGatewayStatusCodeActive {
		vgActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(vg, appmesh.VirtualGatewayActive, vgActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, vg, client.MergeFrom(oldVG))
}

func (m *defaultResourceManager) buildSDKVirtualGatewayTags(ctx context.Context, vg *appmesh.VirtualGateway) []*appmeshsdk.TagRef {
	// TODO, support tags
	return nil
}

// isSDKVirtualGatewayControlledByCRDVirtualGateway checks whether an AppMesh virtualGateway is controlled by CRD virtualGateway
// if it's controlled, CRD virtualGateway update is responsible for update AppMesh virtualGateway.
func (m *defaultResourceManager) isSDKVirtualGatewayControlledByCRDVirtualGateway(ctx context.Context, sdkVG *appmeshsdk.VirtualGatewayData, vg *appmesh.VirtualGateway) bool {
	if aws.StringValue(sdkVG.Metadata.ResourceOwner) != m.accountID {
		return false
	}
	return true
}

// isSDKVirtualGatewayOwnedByCRDVirtualGateway checks whether an AppMesh virtualGateway is owned by CRD virtualGateway.
// if it's owned, CRD virtualGateway deletion is responsible for delete AppMesh virtualGateway.
func (m *defaultResourceManager) isSDKVirtualGatewayOwnedByCRDVirtualGateway(ctx context.Context, sdkVG *appmeshsdk.VirtualGatewayData, vg *appmesh.VirtualGateway) bool {
	if !m.isSDKVirtualGatewayControlledByCRDVirtualGateway(ctx, sdkVG, vg) {
		return false
	}

	// TODO: Add tagging support.
	// currently, virtualGateway control == ownership, but it doesn't have to be so once we add tagging support.
	return true
}

func BuildSDKVirtualGatewaySpec(ctx context.Context, vg *appmesh.VirtualGateway) (*appmeshsdk.VirtualGatewaySpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualGatewaySpec)(nil), (*appmeshsdk.VirtualGatewaySpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_VirtualGatewaySpec_To_SDK_VirtualGatewaySpec(a.(*appmesh.VirtualGatewaySpec), b.(*appmeshsdk.VirtualGatewaySpec), scope)
	})

	sdkVGSpec := &appmeshsdk.VirtualGatewaySpec{}
	if err := converter.Convert(&vg.Spec, sdkVGSpec, nil); err != nil {
		return nil, err
	}
	return sdkVGSpec, nil
}
