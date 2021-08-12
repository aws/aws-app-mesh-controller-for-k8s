package virtualservice

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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualrouter"
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

// ResourceManager is dedicated to manage AppMesh VirtualService resources for k8s VirtualService CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh VirtualService to match vs.spec, and update vs.status
	Reconcile(ctx context.Context, vs *appmesh.VirtualService) error

	// Cleanup will delete AppMesh VirtualService created for vs.
	Cleanup(ctx context.Context, vs *appmesh.VirtualService) error
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

type defaultResourceManager struct {
	k8sClient          client.Client
	appMeshSDK         services.AppMesh
	referencesResolver references.Resolver
	accountID          string
	log                logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vs *appmesh.VirtualService) error {
	ms, err := m.findMeshDependency(ctx, vs)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}
	vnByKey, err := m.findVirtualNodeDependencies(ctx, vs)
	if err != nil {
		return err
	}
	if err := m.validateVirtualNodeDependencies(ctx, ms, vnByKey); err != nil {
		return err
	}
	vrByKey, err := m.findVirtualRouterDependencies(ctx, vs)
	if err != nil {
		return err
	}
	if err := m.validateVirtualRouterDependencies(ctx, ms, vrByKey); err != nil {
		return err
	}

	sdkVS, err := m.findSDKVirtualService(ctx, ms, vs)
	if err != nil {
		return err
	}
	if sdkVS == nil {
		sdkVS, err = m.createSDKVirtualService(ctx, ms, vs, vnByKey, vrByKey)
		if err != nil {
			return err
		}
	} else {
		sdkVS, err = m.updateSDKVirtualService(ctx, sdkVS, vs, vnByKey, vrByKey)
		if err != nil {
			return err
		}
	}
	return m.updateCRDVirtualService(ctx, vs, sdkVS)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vs *appmesh.VirtualService) error {
	ms, err := m.findMeshDependency(ctx, vs)
	if err != nil {
		return err
	}
	sdkVS, err := m.findSDKVirtualService(ctx, ms, vs)
	if err != nil {
		if vs.Status.VirtualServiceARN == nil {
			return nil
		}
		return err
	}
	if sdkVS == nil {
		return nil
	}
	return m.deleteSDKVirtualService(ctx, sdkVS, vs)
}

// findMeshDependency find the Mesh dependency for this VirtualService.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vs *appmesh.VirtualService) (*appmesh.Mesh, error) {
	if vs.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *vs.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
}

// validateMeshDependencies validate the Mesh dependency for this VirtualService.
func (m *defaultResourceManager) validateMeshDependencies(ctx context.Context, ms *appmesh.Mesh) error {
	if !mesh.IsMeshActive(ms) {
		return runtime.NewRequeueError(errors.New("mesh is not active yet"))
	}
	return nil
}

func (m *defaultResourceManager) findVirtualNodeDependencies(ctx context.Context, vs *appmesh.VirtualService) (map[types.NamespacedName]*appmesh.VirtualNode, error) {
	vnRefs := ExtractVirtualNodeReferences(vs)
	vnByKey := make(map[types.NamespacedName]*appmesh.VirtualNode)
	for _, vnRef := range vnRefs {
		vnKey := references.ObjectKeyForVirtualNodeReference(vs, vnRef)
		if _, ok := vnByKey[vnKey]; ok {
			continue
		}
		vn, err := m.referencesResolver.ResolveVirtualNodeReference(ctx, vs, vnRef)
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

func (m *defaultResourceManager) findVirtualRouterDependencies(ctx context.Context, vs *appmesh.VirtualService) (map[types.NamespacedName]*appmesh.VirtualRouter, error) {
	vrRefs := ExtractVirtualRouterReferences(vs)
	vrByKey := make(map[types.NamespacedName]*appmesh.VirtualRouter)
	for _, vrRef := range vrRefs {
		vrKey := references.ObjectKeyForVirtualRouterReference(vs, vrRef)
		if _, ok := vrByKey[vrKey]; ok {
			continue
		}
		vr, err := m.referencesResolver.ResolveVirtualRouterReference(ctx, vs, vrRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve virtualRouterRef")
		}
		vrByKey[vrKey] = vr
	}
	return vrByKey, nil
}

func (m *defaultResourceManager) validateVirtualRouterDependencies(ctx context.Context, ms *appmesh.Mesh, vrByKey map[types.NamespacedName]*appmesh.VirtualRouter) error {
	for _, vr := range vrByKey {
		if vr.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vr.Spec.MeshRef) {
			return errors.Errorf("virtualRouter %v didn't belong to mesh %v", k8s.NamespacedName(vr), k8s.NamespacedName(ms))
		}
		if !virtualrouter.IsVirtualRouterActive(vr) {
			return runtime.NewRequeueError(errors.New("virtualRouter is not active yet"))
		}
	}
	return nil
}

func (m *defaultResourceManager) findSDKVirtualService(ctx context.Context, ms *appmesh.Mesh, vs *appmesh.VirtualService) (*appmeshsdk.VirtualServiceData, error) {
	resp, err := m.appMeshSDK.DescribeVirtualServiceWithContext(ctx, &appmeshsdk.DescribeVirtualServiceInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualServiceName: vs.Spec.AWSName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}
	return resp.VirtualService, nil
}

func (m *defaultResourceManager) createSDKVirtualService(ctx context.Context, ms *appmesh.Mesh, vs *appmesh.VirtualService,
	vnByKey map[types.NamespacedName]*appmesh.VirtualNode, vrByKey map[types.NamespacedName]*appmesh.VirtualRouter) (*appmeshsdk.VirtualServiceData, error) {

	sdkVSSpec, err := BuildSDKVirtualServiceSpec(vs, vnByKey, vrByKey)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateVirtualServiceWithContext(ctx, &appmeshsdk.CreateVirtualServiceInput{
		MeshName:           ms.Spec.AWSName,
		MeshOwner:          ms.Spec.MeshOwner,
		VirtualServiceName: vs.Spec.AWSName,
		Spec:               sdkVSSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualService, nil
}

func (m *defaultResourceManager) updateSDKVirtualService(ctx context.Context, sdkVS *appmeshsdk.VirtualServiceData, vs *appmesh.VirtualService,
	vnByKey map[types.NamespacedName]*appmesh.VirtualNode, vrByKey map[types.NamespacedName]*appmesh.VirtualRouter) (*appmeshsdk.VirtualServiceData, error) {
	actualSDKVSSpec := sdkVS.Spec
	desiredSDKVSSpec, err := BuildSDKVirtualServiceSpec(vs, vnByKey, vrByKey)
	if err != nil {
		return nil, err
	}

	opts := cmpopts.EquateEmpty()
	if cmp.Equal(desiredSDKVSSpec, actualSDKVSSpec, opts) {
		return sdkVS, nil
	}
	if !m.isSDKVirtualServiceControlledByCRDVirtualService(ctx, sdkVS, vs) {
		m.log.V(1).Info("skip virtualService update since it's not controlled",
			"virtualService", k8s.NamespacedName(vs),
			"virtualServiceARN", aws.StringValue(sdkVS.Metadata.Arn),
		)
		return sdkVS, nil
	}

	diff := cmp.Diff(desiredSDKVSSpec, actualSDKVSSpec, opts)
	m.log.V(1).Info("virtualServiceSpec changed",
		"virtualService", k8s.NamespacedName(vs),
		"actualSDKVRSpec", actualSDKVSSpec,
		"desiredSDKVRSpec", desiredSDKVSSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateVirtualServiceWithContext(ctx, &appmeshsdk.UpdateVirtualServiceInput{
		MeshName:           sdkVS.MeshName,
		MeshOwner:          sdkVS.Metadata.MeshOwner,
		VirtualServiceName: sdkVS.VirtualServiceName,
		Spec:               desiredSDKVSSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualService, nil
}

func (m *defaultResourceManager) deleteSDKVirtualService(ctx context.Context, sdkVS *appmeshsdk.VirtualServiceData, vs *appmesh.VirtualService) error {
	if !m.isSDKVirtualServiceOwnedByCRDVirtualService(ctx, sdkVS, vs) {
		m.log.V(1).Info("skip virtualService deletion since its not owned",
			"virtualService", k8s.NamespacedName(vs),
			"virtualServiceARN", aws.StringValue(sdkVS.Metadata.Arn),
		)
		return nil
	}
	_, err := m.appMeshSDK.DeleteVirtualServiceWithContext(ctx, &appmeshsdk.DeleteVirtualServiceInput{
		MeshName:           sdkVS.MeshName,
		MeshOwner:          sdkVS.Metadata.MeshOwner,
		VirtualServiceName: sdkVS.VirtualServiceName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDVirtualService(ctx context.Context, vs *appmesh.VirtualService, sdkVS *appmeshsdk.VirtualServiceData) error {
	oldVS := vs.DeepCopy()
	needsUpdate := false
	if aws.StringValue(vs.Status.VirtualServiceARN) != aws.StringValue(sdkVS.Metadata.Arn) {
		vs.Status.VirtualServiceARN = sdkVS.Metadata.Arn
		needsUpdate = true
	}
	if aws.Int64Value(vs.Status.ObservedGeneration) != vs.Generation {
		vs.Status.ObservedGeneration = aws.Int64(vs.Generation)
		needsUpdate = true
	}

	vsActiveConditionStatus := corev1.ConditionFalse
	if sdkVS.Status != nil && aws.StringValue(sdkVS.Status.Status) == appmeshsdk.VirtualServiceStatusCodeActive {
		vsActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(vs, appmesh.VirtualServiceActive, vsActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, vs, client.MergeFrom(oldVS))
}

// isSDKVirtualServiceControlledByCRDVirtualService checks whether an AppMesh VirtualService is controlled by CRD VirtualService.
// if it's controlled, CRD VirtualService update is responsible for updating the AppMesh VirtualService.
func (m *defaultResourceManager) isSDKVirtualServiceControlledByCRDVirtualService(ctx context.Context, sdkVS *appmeshsdk.VirtualServiceData, vs *appmesh.VirtualService) bool {
	return aws.StringValue(sdkVS.Metadata.ResourceOwner) == m.accountID
}

// isSDKVirtualServiceOwnedByCRDVirtualService checks whether an AppMesh VirtualService is owned by CRD VirtualService.
// if it's owned, CRD VirtualService deletion is responsible for deleting the AppMesh VirtualService.
func (m *defaultResourceManager) isSDKVirtualServiceOwnedByCRDVirtualService(ctx context.Context, sdkVS *appmeshsdk.VirtualServiceData, vs *appmesh.VirtualService) bool {
	if !m.isSDKVirtualServiceControlledByCRDVirtualService(ctx, sdkVS, vs) {
		return false
	}

	// TODO: Adding tagging support, so a existing virtualService in owner account but not ownership can be support.
	// currently, virtualService controllership == ownership, but it don't have to be so once we add tagging support.
	return true
}

func BuildSDKVirtualServiceSpec(vs *appmesh.VirtualService, vnByKey map[types.NamespacedName]*appmesh.VirtualNode, vrByKey map[types.NamespacedName]*appmesh.VirtualRouter) (*appmeshsdk.VirtualServiceSpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualServiceSpec)(nil), (*appmeshsdk.VirtualServiceSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_VirtualServiceSpec_To_SDK_VirtualServiceSpec(a.(*appmesh.VirtualServiceSpec), b.(*appmeshsdk.VirtualServiceSpec), scope)
	})
	sdkVNRefConvertFunc := references.BuildSDKVirtualNodeReferenceConvertFunc(vs, vnByKey)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualNodeReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVNRefConvertFunc(a.(*appmesh.VirtualNodeReference), b.(*string), scope)
	})
	sdkVRRefConvertFunc := references.BuildSDKVirtualRouterReferenceConvertFunc(vs, vrByKey)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualRouterReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVRRefConvertFunc(a.(*appmesh.VirtualRouterReference), b.(*string), scope)
	})

	sdkVSSpec := &appmeshsdk.VirtualServiceSpec{}
	if err := converter.Convert(&vs.Spec, sdkVSSpec, nil); err != nil {
		return nil, err
	}
	return sdkVSSpec, nil
}
