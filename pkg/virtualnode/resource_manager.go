package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/conversion"
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
	meshRefResolver mesh.ReferenceResolver,
	virtualServiceRefResolver virtualservice.ReferenceResolver,
	log logr.Logger) ResourceManager {

	return &defaultResourceManager{
		k8sClient:                 k8sClient,
		appMeshSDK:                appMeshSDK,
		meshRefResolver:           meshRefResolver,
		virtualServiceRefResolver: virtualServiceRefResolver,
		log:                       log,
	}
}

// defaultResourceManager implements ResourceManager
type defaultResourceManager struct {
	k8sClient                 client.Client
	appMeshSDK                services.AppMesh
	meshRefResolver           mesh.ReferenceResolver
	virtualServiceRefResolver virtualservice.ReferenceResolver

	log logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error {
	ms, err := m.findMeshDependency(ctx, vn)
	if err != nil {
		return err
	}
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}
	vsByRef, err := m.findVirtualServiceDependencies(ctx, vn)
	if err != nil {
		return err
	}
	if err := m.validateVirtualServiceDependencies(ctx, ms, vsByRef); err != nil {
		return err
	}

	sdkVN, err := m.findSDKVirtualNode(ctx, ms, vn)
	if err != nil {
		return err
	}
	if sdkVN == nil {
		sdkVN, err = m.createSDKVirtualNode(ctx, ms, vn, vsByRef)
		if err != nil {
			return err
		}
	} else {
		sdkVN, err = m.updateSDKVirtualNode(ctx, sdkVN, ms, vn, vsByRef)
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
	if err := m.validateMeshDependencies(ctx, ms); err != nil {
		return err
	}

	sdkVN, err := m.findSDKVirtualNode(ctx, ms, vn)
	if sdkVN == nil {
		return nil
	}

	return m.deleteSDKVirtualNode(ctx, ms, vn)
}

// findMeshDependency find the Mesh dependency for this virtualNode.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.Mesh, error) {
	if vn.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.meshRefResolver.Resolve(ctx, *vn.Spec.MeshRef)
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
func (m *defaultResourceManager) findVirtualServiceDependencies(ctx context.Context, vn *appmesh.VirtualNode) (map[appmesh.VirtualServiceReference]*appmesh.VirtualService, error) {
	vsByRef := make(map[appmesh.VirtualServiceReference]*appmesh.VirtualService, len(vn.Spec.Backends))
	for _, backend := range vn.Spec.Backends {
		vsRef := backend.VirtualService.VirtualServiceRef
		vs, err := m.virtualServiceRefResolver.Resolve(ctx, vn, vsRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve virtualServiceRef")
		}
		vsByRef[vsRef] = vs
	}
	return vsByRef, nil
}

// validateVirtualServiceDependencies validates the VirtualService dependencies for this virtualNode.
// AppMesh API allows to create a virtualNode with virtualService backend without the virtualService presents first.
// we will not validate whether virtualService is active or not, to allows circular dependency between virtualNode and virtualService.
func (m *defaultResourceManager) validateVirtualServiceDependencies(ctx context.Context, ms *appmesh.Mesh, vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService) error {
	for _, vs := range vsByRef {
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

func (m *defaultResourceManager) createSDKVirtualNode(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeData, error) {
	sdkVNSpec, err := m.buildSDKVirtualNodeSpec(ctx, vn, vsByRef)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateVirtualNodeWithContext(ctx, &appmeshsdk.CreateVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		Spec:            sdkVNSpec,
		Tags:            nil,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualNode, nil
}

func (m *defaultResourceManager) updateSDKVirtualNode(ctx context.Context, sdkVN *appmeshsdk.VirtualNodeData, ms *appmesh.Mesh, vn *appmesh.VirtualNode, vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeData, error) {
	actualSDKVNSpec := sdkVN.Spec
	desiredSDKVNSpec, err := m.buildSDKVirtualNodeSpec(ctx, vn, vsByRef)
	if err != nil {
		return nil, err
	}

	if cmp.Equal(desiredSDKVNSpec, actualSDKVNSpec, equality.IgnoreLeftHandUnset()) {
		return sdkVN, nil
	}
	diff := cmp.Diff(desiredSDKVNSpec, actualSDKVNSpec, equality.IgnoreLeftHandUnset())
	m.log.V(2).Info("virtualNodeSpec changed",
		"virtualNode", k8s.NamespacedName(vn),
		"actualSDKVNSpec", actualSDKVNSpec,
		"desiredSDKVNSpec", desiredSDKVNSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateVirtualNodeWithContext(ctx, &appmeshsdk.UpdateVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		Spec:            desiredSDKVNSpec,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		return nil, err
	}
	return resp.VirtualNode, nil
}

func (m *defaultResourceManager) deleteSDKVirtualNode(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error {
	_, err := m.appMeshSDK.DeleteVirtualNodeWithContext(ctx, &appmeshsdk.DeleteVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		VirtualNodeName: vn.Spec.AWSName,
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
		vn.Status.MeshARN = sdkVN.Metadata.Arn
		needsUpdate = true
	}
	if needsUpdate {
		return nil
	}
	return m.k8sClient.Patch(ctx, vn, client.MergeFrom(oldVN))
}

func (m *defaultResourceManager) buildSDKVirtualNodeSpec(ctx context.Context, vn *appmesh.VirtualNode, vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService) (*appmeshsdk.VirtualNodeSpec, error) {
	sdkVSRefConvertFunc := m.buildSDKVirtualServiceReferenceConvertFunc(ctx, vsByRef)
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualNodeSpec)(nil), (*appmeshsdk.VirtualNodeSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_VirtualNodeSpec_To_SDK_VirtualNodeSpec(a.(*appmesh.VirtualNodeSpec), b.(*appmeshsdk.VirtualNodeSpec), scope)
	})
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualServiceReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVSRefConvertFunc(a.(*appmesh.VirtualServiceReference), b.(*string), scope)
	})
	sdkVNSpec := &appmeshsdk.VirtualNodeSpec{}
	if err := converter.Convert(&vn.Spec, sdkVNSpec, conversion.DestFromSource, nil); err != nil {
		return nil, err
	}
	return sdkVNSpec, nil
}

func (m *defaultResourceManager) buildSDKVirtualNodeTags(ctx context.Context, vn *appmesh.VirtualNode) []*appmeshsdk.TagRef {
	// TODO, support tags
	return nil
}

type sdkVirtualServiceReferenceConvertFunc func(vsRef *appmesh.VirtualServiceReference, vsAWSName *string, scope conversion.Scope) error

func (m *defaultResourceManager) buildSDKVirtualServiceReferenceConvertFunc(ctx context.Context, vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService) sdkVirtualServiceReferenceConvertFunc {
	return func(vsRef *appmesh.VirtualServiceReference, vsAWSName *string, scope conversion.Scope) error {
		vs, ok := vsByRef[*vsRef]
		if !ok {
			return errors.Errorf("unexpected VirtualServiceReference: %v", *vsRef)
		}
		*vsAWSName = aws.StringValue(vs.Spec.AWSName)
		return nil
	}
}
