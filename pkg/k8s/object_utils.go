package k8s

import (
	corev1 "k8s.io/api/core/v1"
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
		return []string{obj.(*corev1.Pod).Namespace}, nil
	}
	return indexer
}
