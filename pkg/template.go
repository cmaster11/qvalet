package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
	textTemplate "text/template"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ fmt.Stringer = (*Template)(nil)

type Template struct {
	originalText string
	tpl          *textTemplate.Template
}

func MustParseTemplate(templateKey string, template string) *Template {
	ift, err := ParseTemplate(templateKey, template)
	if err != nil {
		logrus.WithError(err).WithField("template", template).Fatal("failed to parse if template")
	}
	return ift
}

func ParseTemplate(templateKey string, text string, funcs ...textTemplate.FuncMap) (*Template, error) {
	originalText := text

	funcMap := GetTPLFuncsMap()
	for _, m := range funcs {
		for k, v := range m {
			funcMap[k] = v
		}
	}

	wrapper, err := textTemplate.New(templateKey).Funcs(funcMap).Parse(text)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse template %s", templateKey)
	}

	return &Template{
		originalText: originalText,
		tpl:          wrapper,
	}, nil
}

func (tpl *Template) Execute(args interface{}) (string, error) {
	return ExecuteTextTemplate(tpl.tpl, args)
}

func init() {
	if err := validate.RegisterValidation("template", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		_, err := ParseTemplate("validate-template", value)
		return err == nil
	}); err != nil {
		logrus.WithError(err).Fatal("invalid validation function")
	}
}

// DecodeHook used by mapstructure
func StringToPointerTemplateHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*Template)(nil)) {
			return data, nil
		}

		str := data.(string)

		if str == "" {
			return nil, nil
		}

		return ParseTemplate("ift", str)
	}
}

func (tpl *Template) Clone() (*Template, error) {
	clone, err := tpl.tpl.Clone()
	if err != nil {
		return nil, err
	}
	return &Template{
		tpl.originalText,
		clone,
	}, nil
}

func (tpl *Template) Funcs(m textTemplate.FuncMap) {
	tpl.tpl.Funcs(m)
}

func (tpl *Template) Name() string {
	return tpl.tpl.Name()
}

func (tpl *Template) MarshalJSON() ([]byte, error) {
	return json.Marshal(tpl.originalText)
}

func (tpl *Template) String() string {
	return tpl.originalText
}

func (tpl *Template) UnmarshalJSON(data []byte) error {
	var tplText string
	if err := json.Unmarshal(data, &tplText); err != nil {
		return errors.WithMessage(err, "failed to unmarshal template text to string")
	}

	if tplText == "" {
		return nil
	}

	tpl, err := ParseTemplate("template", tplText)
	if err != nil {
		return errors.WithMessage(err, "failed to parse template (unmarshal)")
	}

	// Copy
	*tpl = *tpl
	return nil
}
