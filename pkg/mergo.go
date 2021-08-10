package pkg

import (
	"reflect"
)

// --- Mergo

var mergoTransformerCustomInstance = mergoTransformerCustom{}

type mergoTransformerCustom struct {
}

var mergoTypePtrBool reflect.Type
var mergoTypeMapStringString reflect.Type
var mergoTypePtrIfTemplate reflect.Type

func init() {
	b := true
	pB := &b
	mergoTypePtrBool = reflect.TypeOf(pB)
	mergoTypeMapStringString = reflect.TypeOf(map[string]string{})
	ift := &IfTemplate{}
	mergoTypePtrIfTemplate = reflect.TypeOf(ift)
}

func (t mergoTransformerCustom) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == mergoTypePtrBool || typ == mergoTypeMapStringString || typ == mergoTypePtrIfTemplate {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				if src.IsNil() {
					return nil
				}
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}
