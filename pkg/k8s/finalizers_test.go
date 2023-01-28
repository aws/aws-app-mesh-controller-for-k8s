package k8s

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func TestHasFinalizer(t *testing.T) {
	tests := []struct {
		name      string
		obj       metav1.Object
		finalizer string
		want      bool
	}{
		{
			name: "finalizer exists and matches",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizers.appmesh.k8s.aws/aws-resources"},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      true,
		},
		{
			name: "finalizer not exists",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      false,
		},
		{
			name: "finalizer exists but not matches",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizers.appmesh.k8s.aws/mesh-members"},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasFinalizer(tt.obj, tt.finalizer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultFinalizerManager_AddFinalizers(t *testing.T) {
	type args struct {
		obj        *appmesh.Mesh
		finalizers []string
	}
	tests := []struct {
		name    string
		args    args
		wantObj *appmesh.Mesh
		wantErr error
	}{
		{
			name: "add one finalizer",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-mesh",
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-1"},
				},
			},
		},
		{
			name: "add one finalizer + added finalizer already exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-1"},
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-1"},
				},
			},
		},
		{
			name: "add one finalizer + other finalizer already exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-2"},
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-2", "finalizer-1"},
				},
			},
		},
		{
			name: "add two finalizer",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-mesh",
					},
				},
				finalizers: []string{"finalizer-1", "finalizer-2"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-1", "finalizer-2"},
				},
			},
		},
		{
			name: "add two finalizer + one added finalizer already exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-2"},
					},
				},
				finalizers: []string{"finalizer-1", "finalizer-2"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-2", "finalizer-1"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := NewDefaultFinalizerManager(k8sClient, logr.New(&log.NullLogSink{}))

			err := k8sClient.Create(ctx, tt.args.obj.DeepCopy())
			assert.NoError(t, err)

			err = m.AddFinalizers(ctx, tt.args.obj, tt.args.finalizers...)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotObj := &appmesh.Mesh{}
				err = k8sClient.Get(ctx, NamespacedName(tt.args.obj), gotObj)
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.wantObj, gotObj, opts), "diff", cmp.Diff(tt.wantObj, gotObj, opts))
			}
		})
	}
}

func Test_defaultFinalizerManager_RemoveFinalizers(t *testing.T) {
	type args struct {
		obj        *appmesh.Mesh
		finalizers []string
	}
	tests := []struct {
		name    string
		args    args
		wantObj *appmesh.Mesh
		wantErr error
	}{
		{
			name: "remove one finalizer",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-1"},
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: nil,
				},
			},
		},
		{
			name: "remove one finalizer + removed finalizer didn't exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-mesh",
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: nil,
				},
			},
		},
		{
			name: "remove one finalizer + other finalizer already exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-1", "finalizer-2"},
					},
				},
				finalizers: []string{"finalizer-1"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-2"},
				},
			},
		},
		{
			name: "remove two finalizer",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-1", "finalizer-2"},
					},
				},
				finalizers: []string{"finalizer-1", "finalizer-2"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: nil,
				},
			},
		},
		{
			name: "remove two finalizer + one removed finalizer already exists",
			args: args{
				obj: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "my-ns",
						Name:       "my-mesh",
						Finalizers: []string{"finalizer-2", "finalizer-3"},
					},
				},
				finalizers: []string{"finalizer-1", "finalizer-2"},
			},
			wantObj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "my-ns",
					Name:       "my-mesh",
					Finalizers: []string{"finalizer-3"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := NewDefaultFinalizerManager(k8sClient, logr.New(&log.NullLogSink{}))

			err := k8sClient.Create(ctx, tt.args.obj.DeepCopy())
			assert.NoError(t, err)

			err = m.RemoveFinalizers(ctx, tt.args.obj, tt.args.finalizers...)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotObj := &appmesh.Mesh{}
				err = k8sClient.Get(ctx, NamespacedName(tt.args.obj), gotObj)
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.wantObj, gotObj, opts), "diff", cmp.Diff(tt.wantObj, gotObj, opts))
			}
		})
	}
}
