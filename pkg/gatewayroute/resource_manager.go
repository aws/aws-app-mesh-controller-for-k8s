package gatewayroute

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceManager is dedicated to manage AppMesh GatewayRoute resources for k8s GatewayRoute CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh GatewayRoute to match gr.spec, and update gr.status
	Reconcile(ctx context.Context, gr *appmesh.GatewayRoute) error

	// Cleanup will delete AppMesh GatewayRoute created for gr.
	Cleanup(ctx context.Context, gr *appmesh.GatewayRoute) error
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

func (m *defaultResourceManager) Reconcile(ctx context.Context, gr *appmesh.GatewayRoute) error {
	ms, err := m.findMeshDependency(ctx, gr)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependency(ctx, ms); err != nil {
		return err
	}
	vg, err := m.findVirtualGatewayDependency(ctx, gr)
	if err != nil {
		return err
	}
	if err := m.validateVirtualGatewayDependency(ctx, ms, vg); err != nil {
		return err
	}
	vsByKey, err := m.findVirtualServiceDependencies(ctx, gr)
	if err != nil {
		return err
	}
	if err := m.validateVirtualServiceDependencies(ctx, ms, vsByKey); err != nil {
		return err
	}

	sdkGR, err := m.findSDKGatewayRoute(ctx, ms, vg, gr)
	if err != nil {
		return err
	}
	if sdkGR == nil {
		sdkGR, err = m.createSDKGatewayRoute(ctx, ms, vg, gr, vsByKey)
		if err != nil {
			return err
		}
	} else {
		sdkGR, err = m.updateSDKGatewayRoute(ctx, sdkGR, ms, vg, gr, vsByKey)
		if err != nil {
			return err
		}
	}

	return m.updateCRDGatewayRoute(ctx, gr, sdkGR)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, gr *appmesh.GatewayRoute) error {
	ms, err := m.findMeshDependency(ctx, gr)
	if err != nil {
		return err
	}
	vg, err := m.findVirtualGatewayDependency(ctx, gr)
	if err != nil {
		return err
	}
	sdkGR, err := m.findSDKGatewayRoute(ctx, ms, vg, gr)
	if err != nil {
		if gr.Status.GatewayRouteARN == nil {
			return nil
		}
		return err
	}
	if sdkGR == nil {
		return nil
	}

	return m.deleteSDKGatewayRoute(ctx, sdkGR, ms, vg, gr)
}

// findMeshDependency find the Mesh dependency for this gatewayRoute.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.Mesh, error) {
	if gr.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *gr.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
}

// validateMeshDependency validate the Mesh dependency for this gatewayRoute.
func (m *defaultResourceManager) validateMeshDependency(ctx context.Context, ms *appmesh.Mesh) error {
	if !mesh.IsMeshActive(ms) {
		return runtime.NewRequeueError(errors.New("mesh is not active yet"))
	}
	return nil
}

// findVirtualGatewayDependency find the VirtualGateway dependency for this gatewayRoute.
func (m *defaultResourceManager) findVirtualGatewayDependency(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.VirtualGateway, error) {
	if gr.Spec.VirtualGatewayRef == nil {
		return nil, errors.Errorf("virtualGatewayRef shouldn't be nil, please check webhook setup")
	}
	vg, err := m.referencesResolver.ResolveVirtualGatewayReference(ctx, gr, *gr.Spec.VirtualGatewayRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve virtualGatewayRef")
	}
	return vg, nil
}

// validateVirtualGatewayDependency validates the VirtualGateway dependencies for this gatewayRoute.
func (m *defaultResourceManager) validateVirtualGatewayDependency(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) error {
	if vg.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vg.Spec.MeshRef) {
		return errors.Errorf("virtualGateway %v didn't belong to mesh %v", k8s.NamespacedName(vg), k8s.NamespacedName(ms))
	}
	if !virtualgateway.IsVirtualGatewayActive(vg) {
		return runtime.NewRequeueError(errors.New("virtualGateway is not active yet"))
	}
	return nil
}

// findVirtualServiceDependencies find the VirtualService dependency for this gatewayRoute.
func (m *defaultResourceManager) findVirtualServiceDependencies(ctx context.Context, gr *appmesh.GatewayRoute) (map[types.NamespacedName]*appmesh.VirtualService, error) {
	vsRefs := ExtractVirtualServiceReferences(gr)
	vsByKey := make(map[types.NamespacedName]*appmesh.VirtualService)
	for _, vsRef := range vsRefs {
		vsKey := references.ObjectKeyForVirtualServiceReference(gr, vsRef)
		if _, ok := vsByKey[vsKey]; ok {
			continue
		}
		vs, err := m.referencesResolver.ResolveVirtualServiceReference(ctx, gr, vsRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve virtualServiceRef")
		}
		vsByKey[vsKey] = vs
	}
	return vsByKey, nil
}

// validateVirtualServiceDependencies validates the VirtualService dependencies for this gatewayRoute.
func (m *defaultResourceManager) validateVirtualServiceDependencies(ctx context.Context, ms *appmesh.Mesh, vsByKey map[types.NamespacedName]*appmesh.VirtualService) error {
	for _, vs := range vsByKey {
		if vs.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vs.Spec.MeshRef) {
			return errors.Errorf("virtualService %v didn't belong to mesh %v", k8s.NamespacedName(vs), k8s.NamespacedName(ms))
		}
		if !virtualservice.IsVirtualServiceActive(vs) {
			return runtime.NewRequeueError(errors.New("virtualService is not active yet"))
		}
	}
	return nil
}

func (m *defaultResourceManager) findSDKGatewayRoute(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute) (*appmeshsdk.GatewayRouteData, error) {
	resp, err := m.appMeshSDK.DescribeGatewayRouteWithContext(ctx, &appmeshsdk.DescribeGatewayRouteInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualGatewayName: vg.Spec.AWSName,
		GatewayRouteName:   gr.Spec.AWSName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}

	return resp.GatewayRoute, nil
}

func (m *defaultResourceManager) createSDKGatewayRoute(ctx context.Context, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.GatewayRouteData, error) {
	sdkGRSpec, err := BuildSDKGatewayRouteSpec(ctx, gr, vsByKey)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateGatewayRouteWithContext(ctx, &appmeshsdk.CreateGatewayRouteInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		Spec:               sdkGRSpec,
		VirtualGatewayName: vg.Spec.AWSName,
		GatewayRouteName:   gr.Spec.AWSName,
	})
	if err != nil {
		return nil, err
	}
	return resp.GatewayRoute, nil
}

func (m *defaultResourceManager) updateSDKGatewayRoute(ctx context.Context, sdkGR *appmeshsdk.GatewayRouteData, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.GatewayRouteData, error) {
	actualSDKGRSpec := sdkGR.Spec
	desiredSDKGRSpec, err := BuildSDKGatewayRouteSpec(ctx, gr, vsByKey)
	if err != nil {
		return nil, err
	}

	opts := cmpopts.EquateEmpty()
	if cmp.Equal(desiredSDKGRSpec, actualSDKGRSpec, opts) {
		return sdkGR, nil
	}
	if !m.isSDKGatewayRouteControlledByCRDGatewayRoute(ctx, sdkGR, gr) {
		m.log.V(2).Info("skip gatewayRoute update since it's not controlled",
			"gatewayRoute", k8s.NamespacedName(gr),
			"gatewayRouteARN", aws.StringValue(sdkGR.Metadata.Arn),
		)
		return sdkGR, nil
	}

	diff := cmp.Diff(desiredSDKGRSpec, actualSDKGRSpec, opts)
	m.log.V(2).Info("gatewayRouteSpec changed",
		"gatewayRoute", k8s.NamespacedName(gr),
		"actualSDKGRSpec", actualSDKGRSpec,
		"desiredSDKGRSpec", desiredSDKGRSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateGatewayRouteWithContext(ctx, &appmeshsdk.UpdateGatewayRouteInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		Spec:               desiredSDKGRSpec,
		VirtualGatewayName: vg.Spec.AWSName,
		GatewayRouteName:   sdkGR.GatewayRouteName,
	})
	if err != nil {
		return nil, err
	}
	return resp.GatewayRoute, nil
}

func (m *defaultResourceManager) deleteSDKGatewayRoute(ctx context.Context, sdkGR *appmeshsdk.GatewayRouteData, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute) error {
	if !m.isSDKGatewayRouteOwnedByCRDGatewayRoute(ctx, sdkGR, gr) {
		m.log.V(2).Info("skip mesh gatewayRoute since its not owned",
			"gatewayRoute", k8s.NamespacedName(gr),
			"gatewayRouteARN", aws.StringValue(sdkGR.Metadata.Arn),
		)
		return nil
	}

	_, err := m.appMeshSDK.DeleteGatewayRouteWithContext(ctx, &appmeshsdk.DeleteGatewayRouteInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualGatewayName: vg.Spec.AWSName,
		GatewayRouteName:   sdkGR.GatewayRouteName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDGatewayRoute(ctx context.Context, gr *appmesh.GatewayRoute, sdkGR *appmeshsdk.GatewayRouteData) error {
	oldGR := gr.DeepCopy()
	needsUpdate := false
	if aws.StringValue(gr.Status.GatewayRouteARN) != aws.StringValue(sdkGR.Metadata.Arn) {
		gr.Status.GatewayRouteARN = sdkGR.Metadata.Arn
		needsUpdate = true
	}

	if aws.Int64Value(gr.Status.ObservedGeneration) != gr.Generation {
		gr.Status.ObservedGeneration = aws.Int64(gr.Generation)
		needsUpdate = true
	}

	grActiveConditionStatus := corev1.ConditionFalse
	if sdkGR.Status != nil && aws.StringValue(sdkGR.Status.Status) == appmeshsdk.GatewayRouteStatusCodeActive {
		grActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(gr, appmesh.GatewayRouteActive, grActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, gr, client.MergeFrom(oldGR))
}

func (m *defaultResourceManager) buildSDKGatewayRouteTags(ctx context.Context, gr *appmesh.GatewayRoute) []*appmeshsdk.TagRef {
	// TODO, support tags
	return nil
}

// isSDKGatewayRouteControlledByCRDGatewayRoute checks whether an AppMesh gatewayRoute is controlled by CRD gatewayRoute
// if it's controlled, CRD gatewayRoute update is responsible for update AppMesh gatewayRoute.
func (m *defaultResourceManager) isSDKGatewayRouteControlledByCRDGatewayRoute(ctx context.Context, sdkGR *appmeshsdk.GatewayRouteData, gr *appmesh.GatewayRoute) bool {
	if aws.StringValue(sdkGR.Metadata.ResourceOwner) != m.accountID {
		return false
	}
	return true
}

// isSDKGatewayRouteOwnedByCRDGatewayRoute checks whether an AppMesh gatewayRoute is owned by CRD gatewayRoute.
// if it's owned, CRD gatewayRoute deletion is responsible for delete AppMesh gatewayRoute.
func (m *defaultResourceManager) isSDKGatewayRouteOwnedByCRDGatewayRoute(ctx context.Context, sdkGR *appmeshsdk.GatewayRouteData, gr *appmesh.GatewayRoute) bool {
	if !m.isSDKGatewayRouteControlledByCRDGatewayRoute(ctx, sdkGR, gr) {
		return false
	}

	// TODO: Add tagging support.
	// currently, gatewayRoute control == ownership, but it doesn't have to be so once we add tagging support.
	return true
}

func BuildSDKGatewayRouteSpec(ctx context.Context, gr *appmesh.GatewayRoute, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.GatewayRouteSpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.GatewayRouteSpec)(nil), (*appmeshsdk.GatewayRouteSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_GatewayRouteSpec_To_SDK_GatewayRouteSpec(a.(*appmesh.GatewayRouteSpec), b.(*appmeshsdk.GatewayRouteSpec), scope)
	})
	sdkVSRefConvertFunc := references.BuildSDKVirtualServiceReferenceConvertFunc(gr, vsByKey)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualServiceReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVSRefConvertFunc(a.(*appmesh.VirtualServiceReference), b.(*string), scope)
	})
	sdkGRSpec := &appmeshsdk.GatewayRouteSpec{}
	if err := converter.Convert(&gr.Spec, sdkGRSpec, nil); err != nil {
		return nil, err
	}
	return sdkGRSpec, nil
}
