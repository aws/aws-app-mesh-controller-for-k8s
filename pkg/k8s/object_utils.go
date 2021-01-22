package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

const (
	//NamespaceIndexKey to index objects in datastore
	NamespaceIndexKey = "namespace"
)

// NamespaceKeyIndexerFunc returns indexers to index objects in datastore using namespace
func NamespaceKeyIndexerFunc() cache.Indexers {
	indexer := map[string]cache.IndexFunc{}
	indexer[NamespaceIndexKey] = func(obj interface{}) (strings []string, err error) {
		meta, err := meta.Accessor(obj)
		if err != nil {
			return nil, fmt.Errorf("object has no meta: %v", err)
		}
		return []string{meta.GetNamespace()}, nil
	}
	return indexer
}
