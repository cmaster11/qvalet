package pkg

import (
	"fmt"
)

func SanitizeInterfaceToMapString(intf interface{}) interface{} {
	if arr, ok := intf.([]interface{}); ok {
		var outArray []interface{}
		for _, el := range arr {
			outArray = append(outArray, SanitizeInterfaceToMapString(el))
		}
		return outArray
	}

	if m, ok := intf.(map[interface{}]interface{}); ok {
		res := map[string]interface{}{}
		for k, v := range m {
			switch v2 := v.(type) {
			case []interface{}:
				res[fmt.Sprint(k)] = SanitizeInterfaceToMapString(v2)
			case map[interface{}]interface{}:
				res[fmt.Sprint(k)] = SanitizeInterfaceToMapString(v2)
			default:
				res[fmt.Sprint(k)] = v
			}
		}
		return res
	}

	return intf
}

func boolVal(value *bool) bool {
	if value != nil {
		return *value
	}
	return false
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func firstBoolPtr(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}

	return nil
}
