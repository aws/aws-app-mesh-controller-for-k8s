package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceManager is dedicated to manage AppMesh Mesh resources for k8s Mesh CRs.
type ResourceManager interface {
	// Reconcile will create/update AppMesh Mesh to match ms.spec, and update ms.status
	Reconcile(ctx context.Context, ms *appmesh.Mesh) error

	// Cleanup will delete AppMesh Mesh created for ms.
	Cleanup(ctx context.Context, ms *appmesh.Mesh) error
}

func NewDefaultResourceManager(
	k8sClient client.Client,
	appMeshSDK services.AppMesh,
	accountID string,
	log logr.Logger) ResourceManager {

	return &defaultResourceManager{
		k8sClient:  k8sClient,
		appMeshSDK: appMeshSDK,
		accountID:  accountID,
		log:        log,
	}
}

// defaultResourceManager implements ResourceManager
type defaultResourceManager struct {
	k8sClient  client.Client
	appMeshSDK services.AppMesh
	// current iam identity's aws accountID, used to differentiate mesh ownership.
	accountID string
	log       logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, ms *appmesh.Mesh) error {
	sdkMS, err := m.findSDKMesh(ctx, ms)
	if err != nil {
		return err
	}
	if sdkMS == nil {
		sdkMS, err = m.createSDKMesh(ctx, ms)
		if err != nil {
			return err
		}
	} else {
		sdkMS, err = m.updateSDKMesh(ctx, sdkMS, ms)
		if err != nil {
			return err
		}
	}
	return m.updateCRDMesh(ctx, ms, sdkMS)
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, ms *appmesh.Mesh) error {
	sdkMS, err := m.findSDKMesh(ctx, ms)
	if err != nil {
		if ms.Status.MeshARN == nil {
			return nil
		}
		return err
	}
	if sdkMS == nil {
		return nil
	}

	return m.deleteSDKMesh(ctx, sdkMS, ms)
}

func (m *defaultResourceManager) findSDKMesh(ctx context.Context, ms *appmesh.Mesh) (*appmeshsdk.MeshData, error) {
	resp, err := m.appMeshSDK.DescribeMeshWithContext(ctx, &appmeshsdk.DescribeMeshInput{
		MeshName:  ms.Spec.AWSName,
		MeshOwner: ms.Spec.MeshOwner,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}

	return resp.Mesh, nil
}

func (m *defaultResourceManager) createSDKMesh(ctx context.Context, ms *appmesh.Mesh) (*appmeshsdk.MeshData, error) {
	sdkMSSpec, err := BuildSDKMeshSpec(ctx, ms)
	if err != nil {
		return nil, err
	}
	resp, err := m.appMeshSDK.CreateMeshWithContext(ctx, &appmeshsdk.CreateMeshInput{
		MeshName: ms.Spec.AWSName,
		Spec:     sdkMSSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.Mesh, nil
}

func (m *defaultResourceManager) updateSDKMesh(ctx context.Context, sdkMS *appmeshsdk.MeshData, ms *appmesh.Mesh) (*appmeshsdk.MeshData, error) {
	actualSDKMSSpec := sdkMS.Spec
	desiredSDKMSSpec, err := BuildSDKMeshSpec(ctx, ms)
	if err != nil {
		return nil, err
	}
	opts := cmpopts.EquateEmpty()
	if cmp.Equal(desiredSDKMSSpec, actualSDKMSSpec, opts) {
		return sdkMS, nil
	}
	if !m.isSDKMeshControlledByCRDMesh(ctx, sdkMS, ms) {
		m.log.V(1).Info("skip mesh update since it's not controlled",
			"mesh", k8s.NamespacedName(ms),
			"meshARN", aws.StringValue(sdkMS.Metadata.Arn),
		)
		return sdkMS, nil
	}

	diff := cmp.Diff(desiredSDKMSSpec, actualSDKMSSpec, opts)
	m.log.V(1).Info("meshSpec changed",
		"mesh", k8s.NamespacedName(ms),
		"actualSDKMSSpec", actualSDKMSSpec,
		"desiredSDKMSSpec", desiredSDKMSSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateMeshWithContext(ctx, &appmeshsdk.UpdateMeshInput{
		MeshName: sdkMS.MeshName,
		Spec:     desiredSDKMSSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.Mesh, nil
}

func (m *defaultResourceManager) deleteSDKMesh(ctx context.Context, sdkMS *appmeshsdk.MeshData, ms *appmesh.Mesh) error {
	if !m.isSDKMeshOwnedByCRDMesh(ctx, sdkMS, ms) {
		m.log.V(1).Info("skip mesh deletion since its not owned",
			"mesh", k8s.NamespacedName(ms),
			"meshARN", aws.StringValue(sdkMS.Metadata.Arn),
		)
		return nil
	}

	_, err := m.appMeshSDK.DeleteMeshWithContext(ctx, &appmeshsdk.DeleteMeshInput{
		MeshName: sdkMS.MeshName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *defaultResourceManager) updateCRDMesh(ctx context.Context, ms *appmesh.Mesh, sdkMS *appmeshsdk.MeshData) error {
	oldMS := ms.DeepCopy()
	needsUpdate := false

	if aws.StringValue(ms.Status.MeshARN) != aws.StringValue(sdkMS.Metadata.Arn) {
		ms.Status.MeshARN = sdkMS.Metadata.Arn
		needsUpdate = true
	}
	if aws.Int64Value(ms.Status.ObservedGeneration) != ms.Generation {
		ms.Status.ObservedGeneration = aws.Int64(ms.Generation)
		needsUpdate = true
	}

	msActiveConditionStatus := corev1.ConditionFalse
	if sdkMS.Status != nil && aws.StringValue(sdkMS.Status.Status) == appmeshsdk.MeshStatusCodeActive {
		msActiveConditionStatus = corev1.ConditionTrue
	}
	if updateCondition(ms, appmesh.MeshActive, msActiveConditionStatus, nil, nil) {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Status().Patch(ctx, ms, client.MergeFrom(oldMS))
}

// isSDKMeshControlledByCRDMesh checks whether an AppMesh mesh is controlled by CRDMesh
// if it's controlled, CRDMesh update is responsible for update AppMesh mesh.
func (m *defaultResourceManager) isSDKMeshControlledByCRDMesh(ctx context.Context, sdkMS *appmeshsdk.MeshData, ms *appmesh.Mesh) bool {
	if aws.StringValue(sdkMS.Metadata.ResourceOwner) != m.accountID {
		return false
	}
	return true
}

// isSDKMeshOwnedByCRDMesh checks whether an AppMesh mesh is owned by CRDMesh.
// if it's owned, CRDMesh deletion is responsible for delete AppMesh mesh.
func (m *defaultResourceManager) isSDKMeshOwnedByCRDMesh(ctx context.Context, sdkMS *appmeshsdk.MeshData, ms *appmesh.Mesh) bool {
	if !m.isSDKMeshControlledByCRDMesh(ctx, sdkMS, ms) {
		return false
	}

	// TODO: Adding tagging support, so a existing mesh in owner account but not ownership can be support.
	// currently, mesh controllership == ownership, but it don't have to be so once we add tagging support.
	return true
}

func BuildSDKMeshSpec(ctx context.Context, ms *appmesh.Mesh) (*appmeshsdk.MeshSpec, error) {
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.MeshSpec)(nil), (*appmeshsdk.MeshSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_MeshSpec_To_SDK_MeshSpec(a.(*appmesh.MeshSpec), b.(*appmeshsdk.MeshSpec), scope)
	})
	sdkMSSpec := &appmeshsdk.MeshSpec{}
	if err := converter.Convert(&ms.Spec, sdkMSSpec, nil); err != nil {
		return nil, err
	}
	return sdkMSSpec, nil
}
