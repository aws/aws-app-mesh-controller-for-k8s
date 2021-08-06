package references

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectReferenceIndexer is responsible for build indexes based on object's reference,
// and fetch objects based on reference using index.
type ObjectReferenceIndexer interface {
	Setup(obj client.Object, indexFuncByKind map[string]ObjectReferenceIndexFunc) error
	Fetch(ctx context.Context, objList client.ObjectList, referentKind string, referentKey types.NamespacedName, opts ...client.ListOption) error
}

type ObjectReferenceIndexFunc func(obj client.Object) []types.NamespacedName

func NewDefaultObjectReferenceIndexer(k8sCache cache.Cache, k8sFieldIndexer client.FieldIndexer) *defaultObjectReferenceIndexer {
	return &defaultObjectReferenceIndexer{
		k8sCache:        k8sCache,
		k8sFieldIndexer: k8sFieldIndexer,
	}
}

var _ ObjectReferenceIndexer = &defaultObjectReferenceIndexer{}

type defaultObjectReferenceIndexer struct {
	k8sCache        cache.Cache
	k8sFieldIndexer client.FieldIndexer
}

func (i *defaultObjectReferenceIndexer) Setup(obj client.Object, indexFuncByKind map[string]ObjectReferenceIndexFunc) error {
	for kind := range indexFuncByKind {
		indexFunc := indexFuncByKind[kind]
		ctrlIndexFunc := func(obj client.Object) []string {
			var indexValues []string
			for _, referent := range indexFunc(obj) {
				indexValues = append(indexValues, buildIndexValue(referent))
			}
			return indexValues
		}
		if err := i.k8sFieldIndexer.IndexField(context.Background(), obj, buildIndexKey(kind), ctrlIndexFunc); err != nil {
			return err
		}
	}
	return nil
}

func (i *defaultObjectReferenceIndexer) Fetch(ctx context.Context, objList client.ObjectList, referentKind string, referentKey types.NamespacedName, opts ...client.ListOption) error {
	indexKey := buildIndexKey(referentKind)
	indexValue := buildIndexValue(referentKey)
	opts = append(opts, client.MatchingFields{indexKey: indexValue})
	return i.k8sCache.List(ctx, objList, opts...)
}

func buildIndexKey(referentKind string) string {
	return "objectRefIndex:" + referentKind
}

func buildIndexValue(referentKey types.NamespacedName) string {
	return fmt.Sprintf("%s/%s", referentKey.Namespace, referentKey.Name)
}
