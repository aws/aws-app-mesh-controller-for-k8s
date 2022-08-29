package mesh

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Manager interface {
	WaitUntilMeshActive(ctx context.Context, mesh *appmesh.Mesh) (*appmesh.Mesh, error)
	WaitUntilMeshDeleted(ctx context.Context, mesh *appmesh.Mesh) error
	CheckMeshInAWS(ctx context.Context, ms *appmesh.Mesh) error
}

func NewManager(k8sClient client.Client, appMeshSDK services.AppMesh) Manager {
	return &defaultManager{k8sClient: k8sClient,
		appMeshSDK: appMeshSDK}
}

type defaultManager struct {
	k8sClient  client.Client
	appMeshSDK services.AppMesh
}

func (m *defaultManager) WaitUntilMeshActive(ctx context.Context, mesh *appmesh.Mesh) (*appmesh.Mesh, error) {
	observedMesh := &appmesh.Mesh{}
	return observedMesh, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		// sometimes there's a delay in the resource showing up
		for i := 0; i < 5; i++ {
			fmt.Printf("Waiting for mesh %d\n", i)
			if err := m.k8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
				if i >= 5 {
					return false, err
				}
			}
			time.Sleep(100 * time.Millisecond)
		}

		for _, condition := range observedMesh.Status.Conditions {
			if condition.Type == appmesh.MeshActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilMeshDeleted(ctx context.Context, mesh *appmesh.Mesh) error {
	observedMesh := &appmesh.Mesh{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) CheckMeshInAWS(ctx context.Context, ms *appmesh.Mesh) error {
	// TODO: handle aws throttling
	desiredSDKMeshSpec, err := mesh.BuildSDKMeshSpec(ctx, ms)
	if err != nil {
		return err
	}
	resp, err := m.appMeshSDK.DescribeMeshWithContext(ctx, &appmeshsdk.DescribeMeshInput{
		MeshName: ms.Spec.AWSName,
	})
	if err != nil {
		return err
	}
	opts := cmpopts.EquateEmpty()
	if !cmp.Equal(desiredSDKMeshSpec, resp.Mesh.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKMeshSpec, resp.Mesh.Spec, opts))
	}
	return nil
}
