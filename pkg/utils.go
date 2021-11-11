package pkg

import (
	"fmt"
	"unsafe"
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

func stringPtr(value string) *string {
	return &value
}

// Taken straight from https://github.com/gin-gonic/gin/blob/ee4de846a894e9049321e809d69f4343f62d2862/internal/bytesconv/bytesconv.go
// StringToBytes converts string to byte slice without a memory allocation.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
