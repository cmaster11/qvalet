package utils

import (
	"encoding/json"
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

const PayloadKeyArrayLength = "__gtePayloadLength"

func ExtractPayloadArgsYAML(payload []byte) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	errDecode := yaml.Unmarshal(payload, &out)
	if errDecode != nil {

		// There is a chance this is an array!
		var arrayData []interface{}

		errDecodeArray := yaml.Unmarshal(payload, &arrayData)
		if errDecodeArray == nil {
			// It was an array!
			// Store indexes as strings, to keep consistency with testing, sounds bad but works
			for idx, el := range arrayData {
				out[strconv.FormatInt(int64(idx), 10)] = el
			}

			out[PayloadKeyArrayLength] = len(arrayData)
		} else {
			return nil, errors.WithMessage(errDecode, "could not bind yaml body")
		}
	}
	return out, nil
}

func ExtractPayloadArgsJSON(payload []byte) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	errDecode := json.Unmarshal(payload, &out)
	if errDecode != nil {

		// There is a chance this is an array!
		var arrayData []interface{}

		errDecodeArray := json.Unmarshal(payload, &arrayData)
		if errDecodeArray == nil {
			// It was an array!
			// Store indexes as strings, to keep consistency with testing, sounds bad but works
			for idx, el := range arrayData {
				out[strconv.FormatInt(int64(idx), 10)] = el
			}

			out[PayloadKeyArrayLength] = len(arrayData)
		} else {
			return nil, errors.WithMessage(errDecode, "could not bind json body")
		}
	}
	return out, nil
}
