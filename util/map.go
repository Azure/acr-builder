package util

// IsMap determines whether the provided interface is a map.
func IsMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}
