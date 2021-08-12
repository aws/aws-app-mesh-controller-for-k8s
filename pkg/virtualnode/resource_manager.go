package virtualnode

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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceManager is dedicated to manage AppMesh VirtualNode resources for k8s VirtualNode CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh VirtualNode to match vn.spec, and update vn.status
	Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error

	// Cleanup will delete AppMesh VirtualNode created for vn.
	Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error
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

func (m *defaultResourceManager) Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error {
	ms, err := m.findMeshDependency(ctx, vn)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}
	vsByKey, err := m.findVirtualServiceDependencies(ctx, vn)
	if err != nil {
		return err
	}
	if err := m.validateVirtualServiceDependencies(ctx, ms, vsByKey); err != nil {
		return err
	}

	sdkVN, err := m.findSDKVirtualNode(ctx, ms, vn)
	if err != nil {
		return err
	}
	if sdkVN == nil {
		sdkVN, err = m.createSDKVirtualNode(ctx, ms, vn, vsByKey)
		if err != nil {
			return err
		}
	} else {
		sdkVN, err = m.updateSDKVirtualNode(ctx, sdkVN, ms, vn, vsByKey)
		if err != nil {
			return err
		}
	}

	return m.updateCRDVirtualNode(ctx, vn, sdkVN)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error {
	ms, err := m.findMeshDependency(ctx, vn)
	if err != nil {
		return err
	}
	sdkVN, err := m.findSDKVirtualNode(ctx, ms, vn)
	if err != nil {
		if vn.Status.VirtualNodeARN == nil {
			return nil
		}
		return err
	}
	if sdkVN == nil {
		return nil
	}

	return m.deleteSDKVirtualNode(ctx, sdkVN, ms, vn)
}

// findMeshDependency find the Mesh dependency for this virtualNode.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.Mesh, error) {
	if vn.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *vn.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
}

// validateMeshDependencies validate the Mesh dependency for this virtualNode.
func (m *defaultResourceManager) validateMeshDependencies(ctx context.Context, ms *appmesh.Mesh) error {
	if !mesh.IsMeshActive(ms) {
		return runtime.NewRequeueError(errors.New("mesh is not active yet"))
	}
	return nil
}

// findVirtualServiceDependencies find the VirtualService dependencies for this virtualNode.
func (m *defaultResourceManager) findVirtualServiceDependencies(ctx context.Context, vn *appmesh.VirtualNode) (map[types.NamespacedName]*appmesh.VirtualService, error) {
	vsByKey := make(map[types.NamespacedName]*appmesh.VirtualService, len(vn.Spec.Backends))
	vsRefs := ExtractVirtualServiceReferences(vn)
	for _, vsRef := range vsRefs {
		vsKey := references.ObjectKeyForVirtualServiceReference(vn, vsRef)
		if _, ok := vsByKey[vsKey]; ok {
			continue
		}
		vs, err := m.referencesResolver.ResolveVirtualServiceReference(ctx, vn, vsRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve virtualServiceRef")
		}
		vsByKey[vsKey] = vs
	}
	return vsByKey, nil
}

// validateVirtualServiceDependencies validates the VirtualService dependencies for this virtualNode.
// AppMesh API allows to create a virtualNode with virtualService backend without the virtualService presents first.
// we will not validate whether virtualService is active or not, to allows circular dependency between virtualNode and virtualService.
func (m *defaultResourceManager) validateVirtualServiceDependencies(ctx context.Context, ms *appmesh.Mesh, vsByKey map[types.NamespacedName]*appmesh.VirtualService) error {
	for _, vs := range vsByKey {
		if vs.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vs.Spec.MeshRef) {
			return errors.Errorf("virtualService %v didn't belong to mesh %v", k8s.NamespacedName(vs), k8s.NamespacedName(ms))
		}
	}
	return nil
}

func (m *defaultResourceManager) findSDKVirtualNode(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) (*appmeshsdk.VirtualNodeData, error) {
	resp, err := m.appMeshSDK.DescribeVirtualNodeWithContext(ctx, &appmeshsdk.DescribeVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}

	return resp.VirtualNode, nil
}

func (m *defaultResourceManager) createSDKVirtualNode(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeData, error) {
	sdkVNSpec, err := BuildSDKVirtualNodeSpec(vn, vsByKey)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateVirtualNodeWithContext(ctx, &appmeshsdk.CreateVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		Spec:            sdkVNSpec,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualNode, nil
}

func (m *defaultResourceManager) updateSDKVirtualNode(ctx context.Context, sdkVN *appmeshsdk.VirtualNodeData, ms *appmesh.Mesh, vn *appmesh.VirtualNode, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeData, error) {
	actualSDKVNSpec := sdkVN.Spec
	desiredSDKVNSpec, err := BuildSDKVirtualNodeSpec(vn, vsByKey)
	if err != nil {
		return nil, err
	}

	opts := equality.CompareOptionForVirtualNodeSpec()
	if cmp.Equal(desiredSDKVNSpec, actualSDKVNSpec, opts) {
		return sdkVN, nil
	}
	if !m.isSDKVirtualNodeControlledByCRDVirtualNode(ctx, sdkVN, vn) {
		m.log.V(1).Info("skip virtualNode update since it's not controlled",
			"virtualNode", k8s.NamespacedName(vn),
			"virtualNodeARN", aws.StringValue(sdkVN.Metadata.Arn),
		)
		return sdkVN, nil
	}

	diff := cmp.Diff(desiredSDKVNSpec, actualSDKVNSpec, opts)
	m.log.V(1).Info("virtualNodeSpec changed",
		"virtualNode", k8s.NamespacedName(vn),
		"actualSDKVNSpec", actualSDKVNSpec,
		"desiredSDKVNSpec", desiredSDKVNSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateVirtualNodeWithContext(ctx, &appmeshsdk.UpdateVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		Spec:            desiredSDKVNSpec,
		VirtualNodeName: sdkVN.VirtualNodeName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualNode, nil
}

func (m *defaultResourceManager) deleteSDKVirtualNode(ctx context.Context, sdkVN *appmeshsdk.VirtualNodeData, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error {
	if !m.isSDKVirtualNodeOwnedByCRDVirtualNode(ctx, sdkVN, vn) {
		m.log.V(1).Info("skip mesh virtualNode since its not owned",
			"virtualNode", k8s.NamespacedName(vn),
			"virtualNodeARN", aws.StringValue(sdkVN.Metadata.Arn),
		)
		return nil
	}

	_, err := m.appMeshSDK.DeleteVirtualNodeWithContext(ctx, &appmeshsdk.DeleteVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		VirtualNodeName: sdkVN.VirtualNodeName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDVirtualNode(ctx context.Context, vn *appmesh.VirtualNode, sdkVN *appmeshsdk.VirtualNodeData) error {
	oldVN := vn.DeepCopy()
	needsUpdate := false
	if aws.StringValue(vn.Status.VirtualNodeARN) != aws.StringValue(sdkVN.Metadata.Arn) {
		vn.Status.VirtualNodeARN = sdkVN.Metadata.Arn
		needsUpdate = true
	}
	if aws.Int64Value(vn.Status.ObservedGeneration) != vn.Generation {
		vn.Status.ObservedGeneration = aws.Int64(vn.Generation)
		needsUpdate = true
	}

	vnActiveConditionStatus := corev1.ConditionFalse
	if sdkVN.Status != nil && aws.StringValue(sdkVN.Status.Status) == appmeshsdk.VirtualNodeStatusCodeActive {
		vnActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(vn, appmesh.VirtualNodeActive, vnActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, vn, client.MergeFrom(oldVN))
}

func (m *defaultResourceManager) buildSDKVirtualNodeTags(ctx context.Context, vn *appmesh.VirtualNode) []*appmeshsdk.TagRef {
	// TODO, support tags
	return nil
}

// isSDKVirtualNodeControlledByCRDVirtualNode checks whether an AppMesh virtualNode is controlled by CRD virtualNode
// if it's controlled, CRD virtualNode update is responsible for updating the AppMesh virtualNode.
func (m *defaultResourceManager) isSDKVirtualNodeControlledByCRDVirtualNode(ctx context.Context, sdkVN *appmeshsdk.VirtualNodeData, vn *appmesh.VirtualNode) bool {
	return aws.StringValue(sdkVN.Metadata.ResourceOwner) == m.accountID
}

// isSDKVirtualNodeOwnedByCRDVirtualNode checks whether an AppMesh virtualNode is owned by CRD virtualNode.
// if it's owned, CRD virtualNode deletion is responsible for deleting the AppMesh virtualNode.
func (m *defaultResourceManager) isSDKVirtualNodeOwnedByCRDVirtualNode(ctx context.Context, sdkVN *appmeshsdk.VirtualNodeData, vn *appmesh.VirtualNode) bool {
	if !m.isSDKVirtualNodeControlledByCRDVirtualNode(ctx, sdkVN, vn) {
		return false
	}

	// TODO: Adding tagging support, so a existing virtualNode in owner account but not ownership can be support.
	// currently, virtualNode controllership == ownership, but it don't have to be so once we add tagging support.
	return true
}

func BuildSDKVirtualNodeSpec(vn *appmesh.VirtualNode, vsByKey map[types.NamespacedName]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeSpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualNodeSpec)(nil), (*appmeshsdk.VirtualNodeSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_VirtualNodeSpec_To_SDK_VirtualNodeSpec(a.(*appmesh.VirtualNodeSpec), b.(*appmeshsdk.VirtualNodeSpec), scope)
	})
	sdkVSRefConvertFunc := references.BuildSDKVirtualServiceReferenceConvertFunc(vn, vsByKey)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualServiceReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVSRefConvertFunc(a.(*appmesh.VirtualServiceReference), b.(*string), scope)
	})
	sdkVNSpec := &appmeshsdk.VirtualNodeSpec{}
	if err := converter.Convert(&vn.Spec, sdkVNSpec, nil); err != nil {
		return nil, err
	}
	return sdkVNSpec, nil
}
