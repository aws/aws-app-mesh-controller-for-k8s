package controller

import (
	"context"
	"os"
	"testing"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	ctrlaws "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	ctrlawsmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/mocks"
	appmeshv1beta1mocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/mocks"
	appmeshv1beta1typedmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks"
	appmeshv1beta1listermocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/listers/appmesh/v1beta1/mocks"
	corev1mocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s.io/client-go/listers/core/v1/mocks"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var stats *metrics.Recorder

func TestMain(m *testing.M) {
	stats = metrics.NewRecorder(true)
	code := m.Run()
	os.Exit(code)
}

func TestSyncInstancesWith(t *testing.T) {
	var (
		meshName             = "test-mesh"
		cloudMapVnodeName    = "cloudmap-vn"
		nonCloudMapVnodeName = "non-cloudmap-vn"
		k8sNamespace         = "test-ns"
		cloudMapVnode        = &appmeshv1beta1.VirtualNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cloudMapVnodeName,
				Namespace: k8sNamespace,
			},
			Spec: appmeshv1beta1.VirtualNodeSpec{
				MeshName: meshName,
				ServiceDiscovery: &appmeshv1beta1.ServiceDiscovery{
					CloudMap: &appmeshv1beta1.CloudMapServiceDiscovery{
						ServiceName:   "foo",
						NamespaceName: "local",
					},
				},
			},
		}
		nonCloudMapVnode = &appmeshv1beta1.VirtualNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nonCloudMapVnodeName,
				Namespace: k8sNamespace,
			},
			Spec: appmeshv1beta1.VirtualNodeSpec{
				MeshName: meshName,
			},
		}

		cloudMapVnodePod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cloudmap-vn-pod",
				Namespace: k8sNamespace,
				Annotations: map[string]string{
					annotationAppMeshMeshName:        meshName,
					annotationAppMeshVirtualNodeName: cloudMapVnodeName,
				},
			},
		}

		nonCloudMapVnodePod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-cloudmap-vn-pod",
				Namespace: k8sNamespace,
				Annotations: map[string]string{
					annotationAppMeshMeshName:        meshName,
					annotationAppMeshVirtualNodeName: nonCloudMapVnodeName,
				},
			},
		}

		nonVnodePod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-vn-pod",
				Namespace: k8sNamespace,
			},
		}

		cloudmapVnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("cloudmap-vnode-instance-id"),
			Attributes: map[string]*string{
				ctrlaws.AttrK8sNamespace: aws.String(k8sNamespace),
				ctrlaws.AttrK8sPod:       aws.String(cloudMapVnodePod.Name),
			},
		}

		nonCloudMapVnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("non-cloudmap-vnode-instance-id"),
			Attributes: map[string]*string{
				ctrlaws.AttrK8sNamespace: aws.String(k8sNamespace),
				ctrlaws.AttrK8sPod:       aws.String(nonCloudMapVnodePod.Name),
			},
		}

		nonVnodeInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("non-vnode-instance-id"),
			Attributes: map[string]*string{
				ctrlaws.AttrK8sNamespace: aws.String(k8sNamespace),
				ctrlaws.AttrK8sPod:       aws.String(nonVnodePod.Name),
			},
		}

		missingPodName     = "missing-pod"
		missingPodInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("missing-pod-instance-id"),
			Attributes: map[string]*string{
				ctrlaws.AttrK8sNamespace: aws.String(k8sNamespace),
				ctrlaws.AttrK8sPod:       aws.String(missingPodName),
			},
		}

		nonPodInstance = &servicediscovery.InstanceSummary{
			Id: awssdk.String("non-pod-instance-id"),
		}

		testData = []struct {
			name             string
			instance         *servicediscovery.InstanceSummary
			expectDeregister bool
		}{
			{"instance associated with cloudmap virtual-node", cloudmapVnodeInstance, false},
			{"instance associated with non cloudmap virtual-node", nonCloudMapVnodeInstance, true},
			{"instance not associated with virtual-node", nonVnodeInstance, true},
			{"instance associated with missing pod", missingPodInstance, true},
			{"instance not associated with pod", nonPodInstance, false},
		}
	)

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCloudAPI := new(ctrlawsmocks.CloudAPI)
			mockMeshClientSet := new(appmeshv1beta1mocks.Interface)
			mockAppmeshv1beta1Client := new(appmeshv1beta1typedmocks.AppmeshV1beta1Interface)
			mockMeshClientSet.On(
				"AppmeshV1beta1",
			).Return(mockAppmeshv1beta1Client)

			mockVirtualNodeLister := new(appmeshv1beta1listermocks.VirtualNodeLister)
			mockVirtualNodeLister.On(
				"List",
				labels.Everything(),
			).Return([]*appmeshv1beta1.VirtualNode{
				cloudMapVnode,
				nonCloudMapVnode,
			}, nil)

			mockCloudAPI.On(
				"ListInstances",
				ctx,
				mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
			).Return(
				[]*servicediscovery.InstanceSummary{
					tt.instance,
				},
				nil,
			)

			if tt.expectDeregister {
				mockCloudAPI.On(
					"DeregisterInstance",
					ctx,
					awssdk.StringValue(tt.instance.Id),
					mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
				).Return(nil)
			} else {
				mockCloudAPI.AssertNotCalled(
					t,
					"DeregisterInstance",
					ctx,
					awssdk.StringValue(tt.instance.Id),
					mock.AnythingOfType("*appmesh.AwsCloudMapServiceDiscovery"),
				)
			}

			mockPodLister := new(corev1mocks.PodLister)
			mockPodNamespaceLister := new(corev1mocks.PodNamespaceLister)
			mockPodLister.On(
				"Pods",
				k8sNamespace,
			).Return(mockPodNamespaceLister)
			mockPodNamespaceLister.
				On(
					"Get",
					cloudMapVnodePod.Name,
				).Return(cloudMapVnodePod, nil).
				On(
					"Get",
					nonCloudMapVnodePod.Name,
				).Return(nonCloudMapVnodePod, nil).
				On(
					"Get",
					nonVnodePod.Name,
				).Return(nonVnodePod, nil).
				On(
					"Get",
					missingPodName,
				).Return(nil, newNotFoundError())

			c := &Controller{
				stats:             stats,
				name:              "test",
				cloud:             mockCloudAPI,
				meshclientset:     mockMeshClientSet,
				virtualNodeLister: mockVirtualNodeLister,
				podsLister:        mockPodLister,
			}
			c.syncInstances(ctx)
		})
	}
}

func newNotFoundError() error {
	return &errors.StatusError{
		ErrStatus: metav1.Status{
			Reason: metav1.StatusReasonNotFound,
		},
	}
}
