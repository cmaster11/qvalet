package pkg

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func GetTPLFuncsMap() template.FuncMap {
	tplFuncs := make(template.FuncMap)

	// Add all Sprig functions
	for key, fn := range sprig.TxtFuncMap() {
		tplFuncs[key] = fn
	}

	// Own functions
	tplFuncs["yamlDecode"] = tplYAMLDecode
	tplFuncs["yamlToJson"] = tplYAMLToJson
	tplFuncs["fileReadToString"] = tplFileReadToString

	return tplFuncs
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
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", errors.WithMessage(err, "failed to read file")
	}
	return string(bytes), nil
}
