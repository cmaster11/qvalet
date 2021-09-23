package utils

// https://stackoverflow.com/questions/2809543/pointer-to-a-map
func MergeMap(acc map[string]interface{}, in map[string]interface{}) map[string]interface{} {
	for k, v := range in {
		acc[k] = v
	}
	return acc
}
