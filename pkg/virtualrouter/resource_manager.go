package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
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

// ResourceManager is dedicated to manage AppMesh VirtualRouter resources for k8s VirtualRouter CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh VirtualRouter to match vr.spec, and update vr.status
	Reconcile(ctx context.Context, vr *appmesh.VirtualRouter) error

	// Cleanup will delete AppMesh VirtualRouter created for vr.
	Cleanup(ctx context.Context, vr *appmesh.VirtualRouter) error
}

func NewDefaultResourceManager(k8sClient client.Client, appMeshSDK services.AppMesh, referencesResolver references.Resolver,
	accountID string, log logr.Logger) ResourceManager {
	routesManager := newDefaultRoutesManager(appMeshSDK, log)
	return &defaultResourceManager{
		k8sClient:          k8sClient,
		appMeshSDK:         appMeshSDK,
		referencesResolver: referencesResolver,
		routesManager:      routesManager,
		accountID:          accountID,
		log:                log,
	}
}

type defaultResourceManager struct {
	k8sClient          client.Client
	appMeshSDK         services.AppMesh
	referencesResolver references.Resolver
	routesManager      routesManager
	accountID          string
	log                logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vr *appmesh.VirtualRouter) error {
	ms, err := m.findMeshDependency(ctx, vr)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}
	vnByKey, err := m.findVirtualNodeDependencies(ctx, vr)
	if err != nil {
		return err
	}
	if err := m.validateVirtualNodeDependencies(ctx, ms, vnByKey); err != nil {
		return err
	}

	sdkVR, err := m.findSDKVirtualRouter(ctx, ms, vr)
	if err != nil {
		return err
	}
	var sdkRouteByName map[string]*appmeshsdk.RouteData
	if sdkVR == nil {
		sdkVR, err = m.createSDKVirtualRouter(ctx, ms, vr)
		if err != nil {
			return err
		}
		sdkRouteByName, err = m.routesManager.create(ctx, ms, vr, vnByKey)
		if err != nil {
			return err
		}
	} else {
		sdkVR, err = m.updateSDKVirtualRouter(ctx, sdkVR, vr)
		if err != nil {
			return err
		}
		sdkRouteByName, err = m.routesManager.update(ctx, ms, vr, vnByKey)
		if err != nil {
			return err
		}
	}

	return m.updateCRDVirtualRouter(ctx, vr, sdkVR, sdkRouteByName)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vr *appmesh.VirtualRouter) error {
	ms, err := m.findMeshDependency(ctx, vr)
	if err != nil {
		return err
	}
	sdkVR, err := m.findSDKVirtualRouter(ctx, ms, vr)
	if err != nil {
		if vr.Status.VirtualRouterARN == nil {
			return nil
		}
		return err
	}
	if sdkVR == nil {
		return nil
	}
	if err := m.routesManager.cleanup(ctx, ms, vr); err != nil {
		return err
	}
	return m.deleteSDKVirtualRouter(ctx, sdkVR, vr)
}

// findMeshDependency find the Mesh dependency for this VirtualRouter.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vr *appmesh.VirtualRouter) (*appmesh.Mesh, error) {
	if vr.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *vr.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
}

// validateMeshDependencies validate the Mesh dependency for this VirtualRouter.
func (m *defaultResourceManager) validateMeshDependencies(ctx context.Context, ms *appmesh.Mesh) error {
	if !mesh.IsMeshActive(ms) {
		return runtime.NewRequeueError(errors.New("mesh is not active yet"))
	}
	return nil
}

func (m *defaultResourceManager) findVirtualNodeDependencies(ctx context.Context, vr *appmesh.VirtualRouter) (map[types.NamespacedName]*appmesh.VirtualNode, error) {
	vnRefs := ExtractVirtualNodeReferences(vr)
	vnByKey := make(map[types.NamespacedName]*appmesh.VirtualNode)
	for _, vnRef := range vnRefs {
		vnKey := references.ObjectKeyForVirtualNodeReference(vr, vnRef)
		if _, ok := vnByKey[vnKey]; ok {
			continue
		}
		vn, err := m.referencesResolver.ResolveVirtualNodeReference(ctx, vr, vnRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve virtualNodeRef")
		}
		vnByKey[vnKey] = vn
	}
	return vnByKey, nil
}

func (m *defaultResourceManager) validateVirtualNodeDependencies(ctx context.Context, ms *appmesh.Mesh, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) error {
	for _, vn := range vnByKey {
		if vn.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vn.Spec.MeshRef) {
			return errors.Errorf("virtualNode %v didn't belong to mesh %v", k8s.NamespacedName(vn), k8s.NamespacedName(ms))
		}
		if !virtualnode.IsVirtualNodeActive(vn) {
			return runtime.NewRequeueError(errors.New("virtualNode is not active yet"))
		}
	}
	return nil
}

func (m *defaultResourceManager) findSDKVirtualRouter(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) (*appmeshsdk.VirtualRouterData, error) {
	resp, err := m.appMeshSDK.DescribeVirtualRouterWithContext(ctx, &appmeshsdk.DescribeVirtualRouterInput{
		MeshName:          ms.Spec.AWSName,
		MeshOwner:         ms.Spec.MeshOwner,
		VirtualRouterName: vr.Spec.AWSName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}

	return resp.VirtualRouter, nil
}

func (m *defaultResourceManager) createSDKVirtualRouter(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) (*appmeshsdk.VirtualRouterData, error) {
	sdkVRSpec, err := BuildSDKVirtualRouterSpec(vr)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateVirtualRouterWithContext(ctx, &appmeshsdk.CreateVirtualRouterInput{
		MeshName:          ms.Spec.AWSName,
		MeshOwner:         ms.Spec.MeshOwner,
		VirtualRouterName: vr.Spec.AWSName,
		Spec:              sdkVRSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualRouter, nil
}

func (m *defaultResourceManager) updateSDKVirtualRouter(ctx context.Context, sdkVR *appmeshsdk.VirtualRouterData, vr *appmesh.VirtualRouter) (*appmeshsdk.VirtualRouterData, error) {
	actualSDKVRSpec := sdkVR.Spec
	desiredSDKVRSpec, err := BuildSDKVirtualRouterSpec(vr)
	if err != nil {
		return nil, err
	}

	opts := cmpopts.EquateEmpty()
	if cmp.Equal(desiredSDKVRSpec, actualSDKVRSpec, opts) {
		return sdkVR, nil
	}
	if !m.isSDKVirtualRouterControlledByCRDVirtualRouter(ctx, sdkVR, vr) {
		m.log.V(1).Info("skip virtualRouter update since it's not controlled",
			"virtualRouter", k8s.NamespacedName(vr),
			"virtualRouterARN", aws.StringValue(sdkVR.Metadata.Arn),
		)
		return sdkVR, nil
	}

	diff := cmp.Diff(desiredSDKVRSpec, actualSDKVRSpec, opts)
	m.log.V(1).Info("virtualRouterSpec changed",
		"virtualRouter", k8s.NamespacedName(vr),
		"actualSDKVRSpec", actualSDKVRSpec,
		"desiredSDKVRSpec", desiredSDKVRSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateVirtualRouterWithContext(ctx, &appmeshsdk.UpdateVirtualRouterInput{
		MeshName:          sdkVR.MeshName,
		MeshOwner:         sdkVR.Metadata.MeshOwner,
		VirtualRouterName: sdkVR.VirtualRouterName,
		Spec:              desiredSDKVRSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualRouter, nil
}

func (m *defaultResourceManager) deleteSDKVirtualRouter(ctx context.Context, sdkVR *appmeshsdk.VirtualRouterData, vr *appmesh.VirtualRouter) error {
	if !m.isSDKVirtualRouterOwnedByCRDVirtualRouter(ctx, sdkVR, vr) {
		m.log.V(1).Info("skip virtualRouter deletion since its not owned",
			"virtualRouter", k8s.NamespacedName(vr),
			"virtualRouterARN", aws.StringValue(sdkVR.Metadata.Arn),
		)
		return nil
	}
	_, err := m.appMeshSDK.DeleteVirtualRouterWithContext(ctx, &appmeshsdk.DeleteVirtualRouterInput{
		MeshName:          sdkVR.MeshName,
		MeshOwner:         sdkVR.Metadata.MeshOwner,
		VirtualRouterName: sdkVR.VirtualRouterName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDVirtualRouter(ctx context.Context, vr *appmesh.VirtualRouter, sdkVR *appmeshsdk.VirtualRouterData, sdkRouteByName map[string]*appmeshsdk.RouteData) error {
	oldVR := vr.DeepCopy()

	needsUpdate := false
	if aws.StringValue(vr.Status.VirtualRouterARN) != aws.StringValue(sdkVR.Metadata.Arn) {
		vr.Status.VirtualRouterARN = sdkVR.Metadata.Arn
		needsUpdate = true
	}
	if aws.Int64Value(vr.Status.ObservedGeneration) != vr.Generation {
		vr.Status.ObservedGeneration = aws.Int64(vr.Generation)
		needsUpdate = true
	}

	routeARNByName := make(map[string]string)
	for name, sdkRoute := range sdkRouteByName {
		routeARNByName[name] = aws.StringValue(sdkRoute.Metadata.Arn)
	}
	if !cmp.Equal(vr.Status.RouteARNs, routeARNByName) {
		vr.Status.RouteARNs = routeARNByName
		needsUpdate = true
	}

	vrActiveConditionStatus := corev1.ConditionFalse
	if sdkVR.Status != nil && aws.StringValue(sdkVR.Status.Status) == appmeshsdk.VirtualRouterStatusCodeActive {
		vrActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(vr, appmesh.VirtualRouterActive, vrActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, vr, client.MergeFrom(oldVR))
}

// isSDKVirtualRouterControlledByCRDVirtualRouter checks whether an AppMesh virtualRouter is controlled by CRD VirtualRouter.
// if it's controlled, CRD VirtualRouter update is responsible for updating the AppMesh virtualRouter.
func (m *defaultResourceManager) isSDKVirtualRouterControlledByCRDVirtualRouter(ctx context.Context, sdkVR *appmeshsdk.VirtualRouterData, vr *appmesh.VirtualRouter) bool {
	return aws.StringValue(sdkVR.Metadata.ResourceOwner) == m.accountID
}

// isSDKVirtualRouterOwnedByCRDVirtualRouter checks whether an AppMesh virtualRouter is owned by CRD VirtualRouter.
// if it's owned, CRD VirtualRouter deletion is responsible for deleting the AppMesh virtualRouter.
func (m *defaultResourceManager) isSDKVirtualRouterOwnedByCRDVirtualRouter(ctx context.Context, sdkVR *appmeshsdk.VirtualRouterData, vr *appmesh.VirtualRouter) bool {
	if !m.isSDKVirtualRouterControlledByCRDVirtualRouter(ctx, sdkVR, vr) {
		return false
	}

	// TODO: Adding tagging support, so a existing virtualRouter in owner account but not ownership can be support.
	// currently, virtualRouter controllership == ownership, but it don't have to be so once we add tagging support.
	return true
}

func BuildSDKVirtualRouterSpec(vr *appmesh.VirtualRouter) (*appmeshsdk.VirtualRouterSpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualRouterSpec)(nil), (*appmeshsdk.VirtualRouterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_VirtualRouterSpec_To_SDK_VirtualRouterSpec(a.(*appmesh.VirtualRouterSpec), b.(*appmeshsdk.VirtualRouterSpec), scope)
	})
	sdkVRSpec := &appmeshsdk.VirtualRouterSpec{}
	if err := converter.Convert(&vr.Spec, sdkVRSpec, nil); err != nil {
		return nil, err
	}
	return sdkVRSpec, nil
}
