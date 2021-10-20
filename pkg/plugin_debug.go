package pkg

import (
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

var _ PluginHookPreExecute = (*PluginDebug)(nil)
var _ PluginHookPostExecute = (*PluginDebug)(nil)
var _ PluginHookOutput = (*PluginDebug)(nil)
var _ PluginConfig = (*PluginDebugConfig)(nil)

const pluginDebugDefaultPrefix = "DEBUG"

type PluginDebugConfig struct {
	// The prefix can be used to properly identify log messages, defaults to [pluginDebugDefaultPrefix]
	Prefix string `mapstructure:"prefix"`

	// You can use the Args map to overwrite/add arguments passed to the command
	// for debugging purposes
	Args map[string]interface{} `mapstructure:"args"`

	// If true, logs the content of all temporary files
	LogFiles bool `mapstructure:"logFiles"`
}

func (c *PluginDebugConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	if c.Prefix == "" {
		c.Prefix = pluginDebugDefaultPrefix
	}
	return &PluginDebug{
		NewPluginBase("debug"),
		c,
		listener,
	}, nil
}

func (c *PluginDebugConfig) IsUnique() bool {
	return false
}

type PluginDebug struct {
	PluginBase

	config   *PluginDebugConfig
	listener *CompiledListener
}

func (p *PluginDebug) Clone(newListener *CompiledListener) (PluginInterface, error) {
	return &PluginDebug{
		PluginBase: NewPluginBase("debug"),
		config:     p.config,
		listener:   newListener,
	}, nil
}

func (p *PluginDebug) HookPreExecute(args map[string]interface{}) (map[string]interface{}, error) {
	// Merge args with any provided ones
	for key, val := range p.config.Args {
		args[key] = val
	}

	p.listener.Logger().WithField("args", args).Warnf("[%s] PRE-EXECUTE", p.config.Prefix)

	return args, nil
}

func (p *PluginDebug) HookPostExecute(commandResult *ExecCommandResult) error {
	p.listener.Logger().WithField("commandResult", commandResult).Warnf("[%s] POST-EXECUTE", p.config.Prefix)

	if p.config.LogFiles {
		for k, vIntf := range p.listener.tplTmpFileNames {
			content, _ := ioutil.ReadFile(vIntf.(string))
			p.listener.Logger().WithField("key", k).WithField("value", string(content)).Warnf("[%s] POST-EXECUTE FILES", p.config.Prefix)
		}
	}

	return nil
}

func (p *PluginDebug) HookOutput(_ *gin.Context, args map[string]interface{}, listenerResponse *ListenerResponse) (handled bool, err error) {
	p.listener.Logger().WithField("args", args).WithField("listenerResponse", listenerResponse).Warnf("[%s] OUTPUT", p.config.Prefix)
	return false, nil
}
