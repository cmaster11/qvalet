package pkg

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type ListenerTemplate = Template
type ListenerIfTemplate = IfTemplate

var rootListenerTemplateListener = &CompiledListener{}
var rootListenerTemplateListenerFuncMap = rootListenerTemplateListener.TplFuncMap()

// DecodeHook used by mapstructure
func StringToPointerListenerTemplateHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*ListenerTemplate)(nil)) {
			return data, nil
		}

		str := data.(string)
		return ParseTemplate("list-tpl", str, rootListenerTemplateListenerFuncMap)
	}
}

func MustParseListenerTemplate(templateKey string, template string) *ListenerTemplate {
	return MustParseTemplate(templateKey, template, rootListenerTemplateListenerFuncMap)
}

func MustParseListenerIfTemplate(templateKey string, template string) *ListenerIfTemplate {
	return MustParseIfTemplate(templateKey, template, rootListenerTemplateListenerFuncMap)
}

// DecodeHook used by mapstructure
func StringToPointerListenerIfTemplateHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*ListenerIfTemplate)(nil)) {
			return data, nil
		}

		str := data.(string)
		return ParseIfTemplate("list-ift", str, rootListenerTemplateListenerFuncMap)
	}
}

func (tpl *ListenerTemplate) CloneForListener(listener *CompiledListener) (*ListenerTemplate, error) {
	clone, err := tpl.Clone()
	if err != nil {
		return nil, err
	}
	clone.tpl.Funcs(listener.TplFuncMap())
	return clone, nil
}

func (ift *ListenerIfTemplate) CloneForListener(listener *CompiledListener) (*ListenerIfTemplate, error) {
	clone, err := ift.Clone()
	if err != nil {
		return nil, err
	}
	clone.tpl.Funcs(listener.TplFuncMap())
	return clone, nil
}
