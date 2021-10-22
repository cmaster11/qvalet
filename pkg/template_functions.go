package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

func GetTPLFuncsMap() template.FuncMap {
	tplFuncs := make(template.FuncMap)

	// Add all Sprig functions
	for key, fn := range sprig.TxtFuncMap() {
		tplFuncs[key] = fn
	}

	// Own functions
	tplFuncs["dump"] = tplDump
	tplFuncs["fileReadToString"] = tplFileReadToString
	tplFuncs["yamlDecode"] = tplYAMLDecode
	tplFuncs["yamlToJson"] = tplYAMLToJson
	tplFuncs["cleanNewLines"] = tplCleanNewLines
	tplFuncs["backoff"] = tplBackoff

	// Override of Sprig function, waiting for https://github.com/Masterminds/sprig/pull/275
	tplFuncs["duration"] = tplDuration

	tplFuncs["eq"] = tplEq
	tplFuncs["ne"] = tplNE
	tplFuncs["lt"] = tplLT
	tplFuncs["le"] = tplLE
	tplFuncs["gt"] = tplGT
	tplFuncs["ge"] = tplGE

	return tplFuncs
}

func ExecuteTextTemplate(tpl *template.Template, args interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, args); err != nil {
		return "", errors.WithMessage(err, "failed to execute template")
	}
	return buf.String(), nil
}

func tplYAMLDecode(value string) (interface{}, error) {
	var outYAML interface{}
	if err := yaml.Unmarshal([]byte(value), &outYAML); err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal yaml value")
	}
	jsonCompatibleMap := SanitizeInterfaceToMapString(outYAML)
	return jsonCompatibleMap, nil
}

func tplYAMLToJson(value string) (string, error) {
	intf, err := tplYAMLDecode(value)
	if err != nil {
		return "", err
	}
	bodyJSON, err := json.Marshal(intf)
	if err != nil {
		return "", errors.WithMessage(err, "failed to marshal json from yaml")
	}

	return string(bodyJSON), nil
}

func tplFileReadToString(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errors.WithMessage(err, "failed to read file")
	}
	return string(b), nil
}

// Override of Sprig function, waiting for https://github.com/Masterminds/sprig/pull/275
func tplDuration(sec interface{}) string {
	var n int64
	switch value := sec.(type) {
	default:
		n = 0
	case string:
		n, _ = strconv.ParseInt(value, 10, 64)
	case int64:
		n = value
	case int:
		n = int64(value)
	}
	return (time.Duration(n) * time.Second).String()
}

// Straight from sprig
func strval(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

func tplDump(value interface{}) (string, error) {
	rt := reflect.ValueOf(value)

	if rt.Kind() == reflect.Ptr {
		rt, _ = indirect(rt)
	}
	if !rt.IsValid() {
		return "<no value>", nil
	}

	switch rt.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		marshaled, err := yaml.MarshalWithOptions(value, yaml.UseLiteralStyleIfMultiline(true))
		if err != nil {
			return "", err
		}
		return string(marshaled), nil
	default:
		return strval(value), nil
	}
}

var regexCleanNewLines = regexp.MustCompile(`(\n\s*){3,}`)

func tplCleanNewLines(text string) string {
	return regexCleanNewLines.ReplaceAllString(text, "\n\n")
}

func tplDurationFromValue(val reflect.Value) (time.Duration, error) {
	if val.IsZero() {
		return 0, nil
	}

	if val.Type() == reflect.TypeOf(time.Duration(0)) {
		return val.Interface().(time.Duration), nil
	}

	if val.Kind() == reflect.String {
		str := val.String()
		return time.ParseDuration(str)
	}

	return 0, errors.Errorf("failed to cast duration from value: %+v", val.Interface())
}

const (
	backoffDefaultInitialInterval = 500 * time.Millisecond
	backoffDefaultMultiplier      = 1.5
	backoffDefaultMaxInterval     = 60 * time.Second
)

func tplBackoff(args ...reflect.Value) (time.Duration, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return 0, errors.New("at least one argument is required")
	}

	idx := 0

	initialInterval := backoffDefaultInitialInterval
	if argsLen > idx {
		valInitialInterval := indirectInterface(args[idx])
		idx++

		_initialInterval, err := tplDurationFromValue(valInitialInterval)
		if err != nil {
			return 0, errors.WithMessage(err, "failed to parse initial interval argument")
		}

		initialInterval = _initialInterval
	}

	multiplier := backoffDefaultMultiplier
	if argsLen > idx {
		valMultiplier := indirectInterface(args[idx])
		idx++

		if valMultiplier.IsZero() {
			return 0, errors.New("the backoff multiplier must be greater than 0")
		}

		casted, err := cast.ToFloat64E(valMultiplier.Interface())
		if err != nil {
			return 0, errors.WithMessage(err, "failed to parse multiplier argument")
		}

		multiplier = casted
	}

	maxInterval := backoffDefaultMaxInterval
	if argsLen > idx {
		valMaxInterval := indirectInterface(args[idx])
		idx++

		_maxInterval, err := tplDurationFromValue(valMaxInterval)
		if err != nil {
			return 0, errors.WithMessage(err, "failed to parse max interval argument")
		}
		maxInterval = _maxInterval
	}

	if maxInterval > 0 {
		if float64(initialInterval) >= float64(maxInterval)/multiplier {
			return maxInterval, nil
		}
	}

	return time.Duration(float64(initialInterval) * multiplier), nil
}
