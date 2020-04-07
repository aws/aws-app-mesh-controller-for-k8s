package algorithm

// MergeStringMap will merge multiple map[string]string into single one.
// The merge is executed for maps argument in sequential order, if a key already exists, the value from previous map is kept.
// e.g. MergeStringMap(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "3", "d": "4"}) == map[string]string{"a": "1", "b": "2", "d": "4"}
func MergeStringMap(maps ...map[string]string) map[string]string {
	ret := make(map[string]string)
	for _, _map := range maps {
		for k, v := range _map {
			if _, ok := ret[k]; !ok {
				ret[k] = v
			}
		}
	}
	return ret
}
