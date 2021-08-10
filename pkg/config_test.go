package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeListenerConfig(t *testing.T) {
	tr := true
	fa := false
	pTr := &tr
	pFa := &fa

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
		// Bool ptr overwrite
		{ListenerConfig{LogCommand: pTr}, ListenerConfig{LogCommand: pTr}, ListenerConfig{}},
		{ListenerConfig{LogCommand: pFa}, ListenerConfig{LogCommand: pTr}, ListenerConfig{LogCommand: pFa}},
		{ListenerConfig{LogCommand: pTr}, ListenerConfig{}, ListenerConfig{LogCommand: pTr}},
		{ListenerConfig{LogCommand: pFa}, ListenerConfig{}, ListenerConfig{LogCommand: pFa}},
	}

	for idx, test := range tests {
		merged, err := mergeListenerConfig(&test.def, &test.ove)
		require.NoError(t, err)
		require.EqualValuesf(t, test.exp, *merged, "test %d", idx)
	}
}
