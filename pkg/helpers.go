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
