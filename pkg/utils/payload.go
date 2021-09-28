package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

const (
	payloadKeyArrayLength       = "__gtePayloadArrayLength"
	keyArgsHeadersKey           = "__gteHeaders"
	defaultFormMultipartMaxSize = 64 * 1024 * 1024
)

func ExtractArgsFromGinContext(c *gin.Context) (map[string]interface{}, error) {
	args := make(map[string]interface{})

	// Use route params, if any
	for _, param := range c.Params {
		args[param.Key] = param.Value
	}

	// Add headers to args
	{
		headerMap := make(map[string]interface{})
		for k := range c.Request.Header {
			headerMap[strings.ToLower(k)] = c.GetHeader(k)
		}
		args[keyArgsHeadersKey] = headerMap
	}

	if c.Request.ContentLength > 0 {
		contentType := c.ContentType()

		if contentType == gin.MIMEJSON || contentType == gin.MIMEPlain || contentType == "" {

			/*
				There could be an object or an array, so we need to expect both
			*/

			payloadBytes, _ := ioutil.ReadAll(c.Request.Body)
			defer c.Request.Body.Close()

			out, err := ExtractPayloadArgsJSON(payloadBytes)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to extract payload arguments (json)")
			}

			for k, v := range out {
				args[k] = v
			}

		} else if contentType == gin.MIMEMultipartPOSTForm || contentType == gin.MIMEPOSTForm {

			// Brutally ignoring errors here, because this function fails at different steps
			_ = c.Request.ParseMultipartForm(defaultFormMultipartMaxSize)

			for key, values := range c.Request.Form {
				if len(values) == 1 {
					args[key] = values[0]
					continue
				}

				args[key] = values
			}

		} else if contentType == "application/x-yaml" || contentType == "application/yaml" || contentType == "text/yaml" || contentType == "text/x-yaml" {

			/*
				There could be an object or an array, so we need to expect both
			*/

			payloadBytes, _ := ioutil.ReadAll(c.Request.Body)
			defer c.Request.Body.Close()

			out, err := ExtractPayloadArgsYAML(payloadBytes)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to extract payload arguments (yaml)")
			}

			for k, v := range out {
				args[k] = v
			}

		} else {
			return nil, errors.New(fmt.Sprintf("invalid content type provided: %s", contentType))
		}
	}

	// Always bind query
	{
		queryMap := make(map[string][]string)
		if err := c.ShouldBindQuery(&queryMap); err != nil {
			return nil, errors.WithMessage(err, "failed to parse request query")
		}
		for key, vals := range queryMap {
			if len(vals) > 0 {
				args[key] = vals[len(vals)-1]
			} else {
				args[key] = true
			}

			args["_query_"+key] = vals
		}
	}

	return args, nil
}

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

			out[payloadKeyArrayLength] = len(arrayData)
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

			out[payloadKeyArrayLength] = len(arrayData)
		} else {
			return nil, errors.WithMessage(errDecode, "could not bind json body")
		}
	}
	return out, nil
}
