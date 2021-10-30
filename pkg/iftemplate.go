package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"qvalet/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ fmt.Stringer = (*IfTemplate)(nil)

type IfTemplate struct {
	originalText string
	tpl          *template.Template
}

func MustParseIfTemplate(templateKey string, ifTemplate string, funcs ...template.FuncMap) *IfTemplate {
	ift, err := ParseIfTemplate(templateKey, ifTemplate, funcs...)
	if err != nil {
		logrus.WithError(err).WithField("template", ifTemplate).Fatal("failed to parse if template")
	}
	return ift
}

func ParseIfTemplate(templateKey string, ifTemplate string, funcs ...template.FuncMap) (*IfTemplate, error) {
	originalText := ifTemplate

	// Simple validation to prevent hacks
	if strings.Contains(ifTemplate, "}}") {
		return nil, errors.New("if-template cannot contain `}}`")
	}

	templateContent := fmt.Sprintf(`
{{- if %s -}}
true
{{- else -}}
false
{{- end -}}
`, ifTemplate)

	funcMap := GetTPLFuncsMap()
	for _, m := range funcs {
		for k, v := range m {
			funcMap[k] = v
		}
	}

	wrapper, err := template.New(templateKey).Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse if-template %s", templateKey)
	}

	return &IfTemplate{
		originalText: originalText,
		tpl:          wrapper,
	}, nil
}

func (ift *IfTemplate) IsTrue(args interface{}) (bool, error) {
	result, err := ExecuteTextTemplate(ift.tpl, args)
	if err != nil {
		return false, errors.WithMessage(err, "failed to execute if-template")
	}
	return result == "true", nil
}

func init() {
	if err := utils.Validate.RegisterValidation("ifTemplate", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		_, err := ParseIfTemplate("validate-template", value)
		return err == nil
	}); err != nil {
		logrus.WithError(err).Fatal("invalid validation function")
	}
}

// DecodeHook used by mapstructure
func StringToPointerIfTemplateHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*IfTemplate)(nil)) {
			return data, nil
		}

		str := data.(string)
		return ParseIfTemplate("ift", str)
	}
}

func (ift *IfTemplate) Clone() (*IfTemplate, error) {
	clone, err := ift.tpl.Clone()
	if err != nil {
		return nil, err
	}
	return &IfTemplate{
		ift.originalText,
		clone,
	}, nil
}

func (ift *IfTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(ift.originalText)
}

func (ift *IfTemplate) String() string {
	return ift.originalText
}

func (ift *IfTemplate) UnmarshalJSON(data []byte) error {
	var tplText string
	if err := json.Unmarshal(data, &tplText); err != nil {
		return errors.WithMessage(err, "failed to unmarshal if-template text to string")
	}

	if tplText == "" {
		return nil
	}

	tpl, err := ParseIfTemplate("if-template", tplText)
	if err != nil {
		return errors.WithMessage(err, "failed to parse if-template (unmarshal)")
	}

	// Copy
	*ift = *tpl
	return nil
}
