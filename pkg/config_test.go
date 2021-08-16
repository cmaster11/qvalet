package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeListenerConfig(t *testing.T) {
	tests := []struct {
		exp ListenerConfig
		def ListenerConfig
		ove ListenerConfig
	}{
		{ListenerConfig{}, ListenerConfig{}, ListenerConfig{}},
		// Simple overwrite
		{ListenerConfig{Command: "hello"}, ListenerConfig{Command: "hello"}, ListenerConfig{}},
		{ListenerConfig{Command: "hello2"}, ListenerConfig{Command: "hello"}, ListenerConfig{Command: "hello2"}},
		// Slice overwrite
		{ListenerConfig{Args: []string{"a", "b"}}, ListenerConfig{Args: []string{"a", "b"}}, ListenerConfig{}},
		{ListenerConfig{Args: []string{"e", "c"}}, ListenerConfig{Args: []string{"a", "b"}}, ListenerConfig{Args: []string{"e", "c"}}},
		{ListenerConfig{Args: []string{"e", "c"}}, ListenerConfig{}, ListenerConfig{Args: []string{"e", "c"}}},
		//  Map overwrite
		{ListenerConfig{Files: map[string]string{"a": "c1", "b": "c2"}}, ListenerConfig{Files: map[string]string{"a": "c1", "b": "c2"}}, ListenerConfig{}},
		{ListenerConfig{Files: map[string]string{"c": "c3", "d": "c4"}}, ListenerConfig{Files: map[string]string{"a": "c1", "b": "c2"}}, ListenerConfig{Files: map[string]string{"c": "c3", "d": "c4"}}},
		{ListenerConfig{Files: map[string]string{"c": "c3", "d": "c4"}}, ListenerConfig{}, ListenerConfig{Files: map[string]string{"c": "c3", "d": "c4"}}},
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
