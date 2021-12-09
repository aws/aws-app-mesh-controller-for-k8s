package k8s

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	fakeDiscovery "k8s.io/client-go/discovery/fake"
	kubeTesting "k8s.io/client-go/testing"
	"testing"
)

func TestNamespacedName(t *testing.T) {
	tests := []struct {
		name string
		obj  metav1.Object
		want types.NamespacedName
	}{
		{
			name: "cluster-scoped object",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "global",
				},
			},
			want: types.NamespacedName{
				Namespace: "",
				Name:      "global",
			},
		},
		{
			name: "namespace-scoped object",
			obj: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
					Name:      "my-node",
				},
			},
			want: types.NamespacedName{
				Namespace: "namespace",
				Name:      "my-node",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NamespacedName(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServerVersion(t *testing.T) {
	discovery := fakeDiscovery.FakeDiscovery{Fake: &kubeTesting.Fake{}}
	expectedVersion := "v1.0.0"
	discovery.FakedServerVersion = &version.Info{
		GitVersion: expectedVersion,
	}
	actualVersion := ServerVersion(&discovery)
	assert.Equal(t, expectedVersion, actualVersion)
}
