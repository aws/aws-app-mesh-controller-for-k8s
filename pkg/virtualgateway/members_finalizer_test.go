package virtualgateway

import (
	"context"
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_pendingMembersFinalizer_buildPendingMembersEventMessage(t *testing.T) {
	type args struct {
		grMembers []*appmesh.GatewayRoute
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "two gatewayRoutes pending",
			args: args{
				grMembers: []*appmesh.GatewayRoute{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "gr-1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "gr-2",
						},
					},
				},
			},
			want: "objects belonging to this virtualGateway exist, please delete them to proceed. gatewayRoute: 2",
		},
		{
			name: "1 gatewayRoute pending",
			args: args{
				grMembers: []*appmesh.GatewayRoute{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "gr-1",
						},
					},
				},
			},
			want: "objects belonging to this virtualGateway exist, please delete them to proceed. gatewayRoute: 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:     k8sClient,
				eventRecorder: eventRecorder,
				log:           logr.New(&log.NullLogSink{}),
			}
			got := m.buildPendingMembersEventMessage(ctx, tt.args.grMembers)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_pendingMembersFinalizer_findGatewayRouteMembers(t *testing.T) {
	vg := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vg-1",
			Namespace: "vg-ns",
			UID:       "uid-1",
		},
	}
	grInVg_1 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "gr-1",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "vg-1",
				Namespace: aws.String("vg-ns"),
				UID:       "uid-1",
			},
		},
	}
	grInVg_2 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "gr-2",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "vg-1",
				Namespace: aws.String("vg-ns"),
				UID:       "uid-1",
			},
		},
	}
	grNotInVg_1 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-3",
			Name:      "gr-3",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "vg-2",
				Namespace: aws.String("vg-ns"),
				UID:       "uid-2",
			},
		},
	}
	grNotInVg_2 := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-4",
			Name:      "gr-4",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "vg-1",
				Namespace: aws.String("vg-ns"),
				UID:       "uid-2",
			},
		},
	}

	type env struct {
		gatewayRoutes []*appmesh.GatewayRoute
	}
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    []*appmesh.GatewayRoute
		wantErr error
	}{
		{
			name: "found no gatewayRoute",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{},
			},
			args: args{
				vg: vg,
			},
			want: []*appmesh.GatewayRoute{},
		},
		{
			name: "found gatewayRoutes that matche",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{
					grInVg_1, grInVg_2, grNotInVg_1, grNotInVg_2,
				},
			},
			args: args{
				vg: vg,
			},
			want: []*appmesh.GatewayRoute{grInVg_1, grInVg_2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              logr.New(&log.NullLogSink{}),
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}

			for _, gr := range tt.env.gatewayRoutes {
				err := k8sClient.Create(ctx, gr.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.findGatewayRouteMembers(ctx, tt.args.vg)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.SortSlices(compareGatewayRoute),
				}
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func Test_pendingMembersFinalizer_Finalize(t *testing.T) {
	vg := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-vg",
			UID:       "uid-1",
			Namespace: "vg-ns",
		},
	}
	gr := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "gr-1",
		},
		Spec: appmesh.GatewayRouteSpec{
			VirtualGatewayRef: &appmesh.VirtualGatewayReference{
				Name:      "my-vg",
				UID:       "uid-1",
				Namespace: aws.String("vg-ns"),
			},
		},
	}

	type env struct {
		gatewayRoutes []*appmesh.GatewayRoute
	}
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "when pending gatewayRoute deletion",
			env: env{
				gatewayRoutes: []*appmesh.GatewayRoute{gr},
			},
			args:    args{vg: vg},
			wantErr: errors.New("pending members deletion"),
		},
		{
			name:    "when pending no member deletion",
			env:     env{},
			args:    args{vg: vg},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			eventRecorder := record.NewFakeRecorder(1)
			m := &pendingMembersFinalizer{
				k8sClient:        k8sClient,
				eventRecorder:    eventRecorder,
				log:              logr.New(&log.NullLogSink{}),
				evaluateInterval: pendingMembersFinalizerEvaluateInterval,
			}
			for _, gr := range tt.env.gatewayRoutes {
				err := k8sClient.Create(ctx, gr)
				assert.NoError(t, err)
			}

			err := m.Finalize(ctx, tt.args.vg)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func compareGatewayRoute(a *appmesh.GatewayRoute, b *appmesh.GatewayRoute) bool {
	return k8s.NamespacedName(a).String() < k8s.NamespacedName(b).String()
}
