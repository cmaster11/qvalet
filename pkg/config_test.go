package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeListenerConfig(t *testing.T) {
	tplHello := MustParseListenerTemplate("", "hello")
	tplHello2 := MustParseListenerTemplate("", "hello2")

	tplA := MustParseListenerTemplate("", "a")
	tplB := MustParseListenerTemplate("", "b")
	tplC := MustParseListenerTemplate("", "c")
	tplD := MustParseListenerTemplate("", "d")

	tests := []struct {
		exp ListenerConfig
		def ListenerConfig
		ove ListenerConfig
	}{
		{ListenerConfig{}, ListenerConfig{}, ListenerConfig{}},
		// Simple overwrite
		{ListenerConfig{Command: tplHello}, ListenerConfig{Command: tplHello}, ListenerConfig{}},
		{ListenerConfig{Command: tplHello2}, ListenerConfig{Command: tplHello}, ListenerConfig{Command: tplHello2}},
		// Slice overwrite
		{ListenerConfig{Args: []*ListenerTemplate{tplA, tplB}}, ListenerConfig{Args: []*ListenerTemplate{tplA, tplB}}, ListenerConfig{}},
		{ListenerConfig{Args: []*ListenerTemplate{tplD, tplC}}, ListenerConfig{Args: []*ListenerTemplate{tplA, tplB}}, ListenerConfig{Args: []*ListenerTemplate{tplD, tplC}}},
		{ListenerConfig{Args: []*ListenerTemplate{tplD, tplC}}, ListenerConfig{}, ListenerConfig{Args: []*ListenerTemplate{tplD, tplC}}},
		//  Map overwrite
		{ListenerConfig{Files: map[string]*ListenerTemplate{"a": tplA, "b": tplB}}, ListenerConfig{Files: map[string]*ListenerTemplate{"a": tplA, "b": tplB}}, ListenerConfig{}},
		{ListenerConfig{Files: map[string]*ListenerTemplate{"c": tplC, "d": tplD}}, ListenerConfig{Files: map[string]*ListenerTemplate{"a": tplA, "b": tplB}}, ListenerConfig{Files: map[string]*ListenerTemplate{"c": tplC, "d": tplD}}},
		{ListenerConfig{Files: map[string]*ListenerTemplate{"c": tplC, "d": tplD}}, ListenerConfig{}, ListenerConfig{Files: map[string]*ListenerTemplate{"c": tplC, "d": tplD}}},
		// Log key overwrite
		{ListenerConfig{Log: []LogKey{LogKeyArgs, LogKeyOutput}}, ListenerConfig{Log: []LogKey{LogKeyArgs, LogKeyOutput}}, ListenerConfig{}},
		{ListenerConfig{Log: []LogKey{LogKeyAll}}, ListenerConfig{Log: []LogKey{LogKeyArgs, LogKeyOutput}}, ListenerConfig{Log: []LogKey{LogKeyAll}}},
	}

	for idx, test := range tests {
		merged, err := MergeListenerConfig(&test.def, &test.ove)
		require.NoError(t, err)
		require.EqualValuesf(t, test.exp, *merged, "test %d", idx)
	}
}
