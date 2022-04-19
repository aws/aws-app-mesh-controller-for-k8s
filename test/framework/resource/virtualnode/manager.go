package virtualnode

import (
	"context"
	"fmt"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilVirtualNodeActive(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.VirtualNode, error)
	WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmesh.VirtualNode) error
	CheckVirtualNodeInAWS(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error
	CheckVirtualNodeInCloudMap(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, ipFamily string) error
}

func NewManager(k8sClient client.Client, appMeshSDK services.AppMesh, cloudMapSDK services.CloudMap, ipFamily string) Manager {
	return &defaultManager{
		k8sClient:   k8sClient,
		appMeshSDK:  appMeshSDK,
		cloudMapSDK: cloudMapSDK,
		ipFamily:    ipFamily,
	}
}

type defaultManager struct {
	k8sClient   client.Client
	appMeshSDK  services.AppMesh
	cloudMapSDK services.CloudMap
	ipFamily    string
}

func (m *defaultManager) WaitUntilVirtualNodeActive(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.VirtualNode, error) {
	observedVN := &appmesh.VirtualNode{}
	return observedVN, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		// sometimes there's a delay in the resource showing up
		for i := 0; i < 5; i++ {
			if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vn), observedVN); err != nil {
				if i >= 5 {
					return false, err
				}
			}
			time.Sleep(100 * time.Millisecond)
		}

		for _, condition := range observedVN.Status.Conditions {
			if condition.Type == appmesh.VirtualNodeActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmesh.VirtualNode) error {
	observedVN := &appmesh.VirtualNode{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vn), observedVN); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) CheckVirtualNodeInAWS(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error {
	// TODO: handle aws throttling
	vsByKey := make(map[types.NamespacedName]*appmesh.VirtualService)
	vsRefs := virtualnode.ExtractVirtualServiceReferences(vn)
	for _, vsRef := range vsRefs {
		vs := &appmesh.VirtualService{}
		if err := m.k8sClient.Get(ctx, references.ObjectKeyForVirtualServiceReference(vn, vsRef), vs); err != nil {
			return err
		}
		vsByKey[k8s.NamespacedName(vs)] = vs
	}
	desiredSDKVNSpec, err := virtualnode.BuildSDKVirtualNodeSpec(vn, vsByKey)
	if err != nil {
		return err
	}
	resp, err := m.appMeshSDK.DescribeVirtualNodeWithContext(ctx, &appmeshsdk.DescribeVirtualNodeInput{
		MeshName:        ms.Spec.AWSName,
		MeshOwner:       ms.Spec.MeshOwner,
		VirtualNodeName: vn.Spec.AWSName,
	})
	if err != nil {
		return err
	}
	opts := equality.CompareOptionForVirtualNodeSpec()
	if !cmp.Equal(desiredSDKVNSpec, resp.VirtualNode.Spec, opts) {
		return errors.New(cmp.Diff(desiredSDKVNSpec, resp.VirtualNode.Spec, opts))
	}

	if vn.Spec.ServiceDiscovery.AWSCloudMap != nil {
		if err := m.CheckVirtualNodeInCloudMap(ctx, ms, vn, m.ipFamily); err != nil {
			return err
		}
	}

	return nil
}

func (m *defaultManager) CheckVirtualNodeInCloudMap(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, ipFamily string) error {
	//Get Pods that the VirtualNode selects on
	var podsList corev1.PodList
	var listOptions client.ListOptions
	listOptions.LabelSelector, _ = metav1.LabelSelectorAsSelector(vn.Spec.PodSelector)
	listOptions.Namespace = vn.Namespace

	if err := m.k8sClient.List(ctx, &podsList, &listOptions); err != nil {
		return err
	}

	instanceCount := len(podsList.Items)
	localInstanceInfoMap := make(map[string]map[string]string, instanceCount)
	for i := range podsList.Items {
		pod := &podsList.Items[i]
		instanceAttributeMap := make(map[string]string)
		if ipFamily == utils.IPv6 {
			instanceAttributeMap[cloudmap.AttrAWSInstanceIPV6] = pod.Status.PodIP
		} else {
			instanceAttributeMap[cloudmap.AttrAWSInstanceIPV4] = pod.Status.PodIP
		}
		instanceAttributeMap[cloudmap.AttrK8sPod] = pod.Name
		instanceAttributeMap[cloudmap.AttrK8sNamespace] = pod.Namespace
		instanceAttributeMap[cloudmap.AttrAppMeshMesh] = awssdk.StringValue(ms.Spec.AWSName)
		instanceAttributeMap[cloudmap.AttrAppMeshVirtualNode] = awssdk.StringValue(vn.Spec.AWSName)
		instanceAttributeMap[cloudmap.AttrAWSInstancePort] = strconv.Itoa(int(vn.Spec.Listeners[0].PortMapping.Port))
		localInstanceInfoMap[pod.Status.PodIP] = instanceAttributeMap
	}
	cloudMapConfig := vn.Spec.ServiceDiscovery.AWSCloudMap

	//Get CloudMap Namespace Info
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var nsSummary *servicediscovery.NamespaceSummary
	if err := m.cloudMapSDK.ListNamespacesPagesWithContext(ctx, listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == cloudMapConfig.NamespaceName {
					nsSummary = ns
					return false
				}
			}
			return true
		},
	); err != nil {
		return err
	}

	//Get ServiceInfo from CloudMap
	listServicesInput := &servicediscovery.ListServicesInput{
		Filters: []*servicediscovery.ServiceFilter{
			{
				Name:   awssdk.String(servicediscovery.ServiceFilterNameNamespaceId),
				Values: []*string{nsSummary.Id},
			},
		},
	}
	var sdkSVCSummary *servicediscovery.ServiceSummary
	if err := m.cloudMapSDK.ListServicesPagesWithContext(ctx, listServicesInput,
		func(listServicesOutput *servicediscovery.ListServicesOutput, lastPage bool) bool {
			for _, svc := range listServicesOutput.Services {
				if awssdk.StringValue(svc.Name) == cloudMapConfig.ServiceName {
					sdkSVCSummary = svc
					return false
				}
			}
			return true
		},
	); err != nil {
		return err
	}

	listInstancesInput := &servicediscovery.ListInstancesInput{
		ServiceId: sdkSVCSummary.Id,
	}

	//Get Instance info
	cloudMapInstanceInfoMap := make(map[string]map[string]string, instanceCount)
	if err := m.cloudMapSDK.ListInstancesPagesWithContext(ctx, listInstancesInput,
		func(listInstancesOutput *servicediscovery.ListInstancesOutput, lastPage bool) bool {
			for _, instance := range listInstancesOutput.Instances {
				cloudMapInstanceAttributes := make(map[string]string)
				if ipFamily == utils.IPv6 {
					cloudMapInstanceAttributes[cloudmap.AttrAWSInstanceIPV6] = *instance.Attributes[cloudmap.AttrAWSInstanceIPV6]
				} else {
					cloudMapInstanceAttributes[cloudmap.AttrAWSInstanceIPV4] = *instance.Attributes[cloudmap.AttrAWSInstanceIPV4]
				}
				cloudMapInstanceAttributes[cloudmap.AttrK8sPod] = *instance.Attributes[cloudmap.AttrK8sPod]
				cloudMapInstanceAttributes[cloudmap.AttrK8sNamespace] = *instance.Attributes[cloudmap.AttrK8sNamespace]
				cloudMapInstanceAttributes[cloudmap.AttrAppMeshMesh] = *instance.Attributes[cloudmap.AttrAppMeshMesh]
				cloudMapInstanceAttributes[cloudmap.AttrAppMeshVirtualNode] = *instance.Attributes[cloudmap.AttrAppMeshVirtualNode]
				cloudMapInstanceAttributes[cloudmap.AttrAWSInstancePort] = *instance.Attributes[cloudmap.AttrAWSInstancePort]

				cloudMapInstanceInfoMap[*instance.Id] = cloudMapInstanceAttributes
			}
			return true
		},
	); err != nil {
		return err
	}

	if len(cloudMapInstanceInfoMap) != len(localInstanceInfoMap) {
		return fmt.Errorf("instance count mismatch")
	}
	if err := compareInstances(cloudMapInstanceInfoMap, localInstanceInfoMap); err != nil {
		return fmt.Errorf("instance info mismatch")
	}
	return nil
}

func compareInstances(cloudMapInstanceInfo map[string]map[string]string, localInstanceInfo map[string]map[string]string) error {
	for cloudMapInstanceId, cloudMapInstanceAttr := range cloudMapInstanceInfo {
		localInstanceAttributes := localInstanceInfo[cloudMapInstanceId]
		if cloudMapInstanceAttr[cloudmap.AttrAWSInstanceIPV4] != localInstanceAttributes[cloudmap.AttrAWSInstanceIPV4] ||
			cloudMapInstanceAttr[cloudmap.AttrAWSInstanceIPV6] != localInstanceAttributes[cloudmap.AttrAWSInstanceIPV6] ||
			cloudMapInstanceAttr[cloudmap.AttrK8sPod] != localInstanceAttributes[cloudmap.AttrK8sPod] ||
			cloudMapInstanceAttr[cloudmap.AttrK8sNamespace] != localInstanceAttributes[cloudmap.AttrK8sNamespace] ||
			cloudMapInstanceAttr[cloudmap.AttrAppMeshMesh] != localInstanceAttributes[cloudmap.AttrAppMeshMesh] ||
			cloudMapInstanceAttr[cloudmap.AttrAppMeshVirtualNode] != localInstanceAttributes[cloudmap.AttrAppMeshVirtualNode] ||
			cloudMapInstanceAttr[cloudmap.AttrAWSInstancePort] != localInstanceAttributes[cloudmap.AttrAWSInstancePort] {
			return fmt.Errorf("instance info mismatch")
		}
	}
	return nil
}
