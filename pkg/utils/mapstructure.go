package utils

import (
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
