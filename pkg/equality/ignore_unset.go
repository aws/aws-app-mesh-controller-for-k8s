package equality

import (
	"github.com/google/go-cmp/cmp"
	"reflect"
)

// IgnoreLeftHandUnset is an option that ignores fields that are unset on the
// left hand side of a comparison.
// Note:
//	1. for map and slices, only nil value is considered to be unset, non-nil but empty is not considered as unset.
//  2. for struct pointers, nil value is considered to be unset
func IgnoreLeftHandUnset() cmp.Option {
	return cmp.FilterPath(func(path cmp.Path) bool {
		v1, _ := path.Last().Values()
		switch v1.Kind() {
		case reflect.Slice, reflect.Map, reflect.Ptr:
			return v1.IsNil()
		}
		return false
	}, cmp.Ignore())
}
