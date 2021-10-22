package pkg

import (
	"reflect"
)

// --- Mergo

var MergoTransformerCustomInstance = mergoTransformerCustom{}

type mergoTransformerCustom struct {
}

var mergoTypePtrBool reflect.Type
var mergoTypeMapStringString reflect.Type
var mergoTypeMapStringTemplate reflect.Type
var mergoTypeMapStringIfTemplate reflect.Type
var mergoTypeMapStringListenerTemplate reflect.Type
var mergoTypeMapStringListenerIfTemplate reflect.Type
var mergoTypePtrTemplate reflect.Type
var mergoTypePtrIfTemplate reflect.Type
var mergoTypePtrListenerTemplate reflect.Type
var mergoTypePtrListenerIfTemplate reflect.Type
var mergoTypeLogKeySlice reflect.Type
var mergoTypeReturnKeySlice reflect.Type
var mergoTypeStorageKeySlice reflect.Type

func init() {
	b := true
	pB := &b
	mergoTypePtrBool = reflect.TypeOf(pB)
	mergoTypeMapStringString = reflect.TypeOf(map[string]string{})
	mergoTypeMapStringTemplate = reflect.TypeOf(map[string]*Template{})
	mergoTypeMapStringIfTemplate = reflect.TypeOf(map[string]*IfTemplate{})
	mergoTypeMapStringListenerTemplate = reflect.TypeOf(map[string]*ListenerTemplate{})
	mergoTypeMapStringListenerIfTemplate = reflect.TypeOf(map[string]*ListenerIfTemplate{})
	mergoTypePtrTemplate = reflect.TypeOf(&Template{})
	mergoTypePtrIfTemplate = reflect.TypeOf(&IfTemplate{})
	mergoTypePtrListenerTemplate = reflect.TypeOf(&ListenerTemplate{})
	mergoTypePtrListenerIfTemplate = reflect.TypeOf(&ListenerIfTemplate{})
	mergoTypeLogKeySlice = reflect.TypeOf([]LogKey{})
	mergoTypeReturnKeySlice = reflect.TypeOf([]ReturnKey{})
	mergoTypeStorageKeySlice = reflect.TypeOf([]StoreKey{})
}

func (t mergoTransformerCustom) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == mergoTypePtrBool ||
		typ == mergoTypeMapStringString ||
		typ == mergoTypePtrTemplate ||
		typ == mergoTypePtrIfTemplate ||
		typ == mergoTypePtrListenerTemplate ||
		typ == mergoTypePtrListenerIfTemplate ||
		typ == mergoTypeMapStringTemplate ||
		typ == mergoTypeMapStringIfTemplate ||
		typ == mergoTypeMapStringListenerTemplate ||
		typ == mergoTypeMapStringListenerIfTemplate ||
		typ == mergoTypeLogKeySlice ||
		typ == mergoTypeStorageKeySlice ||
		typ == mergoTypeReturnKeySlice {
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
