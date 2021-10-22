package pkg

import (
	"testing"

	"gotoexec/pkg/utils"

	"github.com/stretchr/testify/require"
)

var _ PluginConfig = (*PluginTestUniqueConfig)(nil)

type PluginTestUniqueConfig struct {
	Name string `mapstructure:"name"`
}

func (c *PluginTestUniqueConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	panic("not implemented")
}

func (c *PluginTestUniqueConfig) IsUnique() bool {
	return true
}

type TestPluginEntryConfig struct {
	TestUnique *PluginTestUniqueConfig `mapstructure:"testUnique"`
}

type TestConfigWrapper struct {
	Plugins []*TestPluginEntryConfig `mapstructure:"plugins" validate:"uniquePlugins,dive,required"`
}

func TestUniquePlugins(t *testing.T) {
	require.NoError(t, utils.Validate.Struct(TestConfigWrapper{
		Plugins: []*TestPluginEntryConfig{
			{
				TestUnique: &PluginTestUniqueConfig{Name: "Mr."},
			},
		},
	}))

	require.Error(t, utils.Validate.Struct(TestConfigWrapper{
		Plugins: []*TestPluginEntryConfig{
			{
				TestUnique: &PluginTestUniqueConfig{Name: "Mr."},
			},
			{
				TestUnique: &PluginTestUniqueConfig{Name: "Anderson"},
			},
		},
	}))
}
