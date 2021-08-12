package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
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