package utils

import (
	"os"
	"reflect"
	"regexp"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var defaultJSONDecodeHook = mapstructure.ComposeDecodeHookFunc(
	// Default
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
)

func DecodeStructJSONToMap(input interface{}, output interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     output,
		TagName:    "json",
		DecodeHook: defaultJSONDecodeHook,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return errors.WithMessage(err, "failed to instantiate mapstructure decoder")
	}

	if err := decoder.Decode(input); err != nil {
		return errors.WithMessage(err, "failed to decode input")
	}

	return nil
}

var stringFromEnvVarRegex = regexp.MustCompile(`^ENV{(\w+)}$`)

type StringFromEnvVar struct {
	wrapped string
}

func (s *StringFromEnvVar) Value() string {
	return s.wrapped
}

func NewStringFromEnvVar(value string) *StringFromEnvVar {
	if match := stringFromEnvVarRegex.FindStringSubmatch(value); match != nil {
		return &StringFromEnvVar{
			wrapped: os.Getenv(match[1]),
		}
	}

	return &StringFromEnvVar{
		wrapped: value,
	}
}

func StringToStringFromEnvVarHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*StringFromEnvVar)(nil)) {
			return data, nil
		}

		return NewStringFromEnvVar(data.(string)), nil
	}
}
