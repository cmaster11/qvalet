package pkg

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

var _ PluginInterface = (*PluginRetry)(nil)
var _ PluginHookRetry = (*PluginRetry)(nil)
var _ PluginConfig = (*PluginRetryConfig)(nil)

// @formatter:off
/// [config]
const (
	pluginRetryDefaultDelay      = 3 * time.Second
	pluginRetryDefaultMaxRetries = 3
)

type PluginRetryConfig struct {
	// Condition template for when to retry the command execution
	Condition *ListenerIfTemplate `mapstructure:"condition" validate:"required"`

	// Delay template you can use to decide how long to wait before the
	// next retry. Needs to return a value compatible with [https://pkg.go.dev/time#ParseDuration].
	// Defaults to [pluginRetryDefaultDelay].
	Delay *ListenerTemplate `mapstructure:"delay"`

	// NOTE: If neither MaxRetries nor MaxElapsed are provided, the plugin
	// will default to a MaxRetries of [pluginRetryDefaultMaxRetries].

	// If provided, limits max amount of retries
	MaxRetries *int `mapstructure:"maxRetries"`

	// If provided, limits the maximum amount of time spent retrying
	MaxElapsed *time.Duration `mapstructure:"maxElapsed"`
}

/// [config]
// @formatter:on

// @formatter:off
/// [retry-payload]
// On every retry cycle, the [PluginRetryInfo] payload can be accessed
// under the [pluginRetryKeyRetryInfo] key.
const pluginRetryKeyRetryInfo = "__qvRetry"

type PluginRetryInfo struct {
	// How much time has passed since the request started?
	Elapsed time.Duration

	// Which retry are we at? Starts from 1.
	RetryCount int

	// What has been the previous execution result?
	PreviousResult *ExecCommandResult
}

/// [retry-payload]
// @formatter:on

func (c *PluginRetryConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	return &PluginRetry{
		NewPluginBase("retry"),
		c,
		c.Condition,
		c.Delay,
	}, nil
}

func (c *PluginRetryConfig) IsUnique() bool {
	return false
}

type PluginRetry struct {
	PluginBase

	config *PluginRetryConfig

	tplCondition *ListenerIfTemplate
	tplDelay     *ListenerTemplate
}

func (p *PluginRetry) Clone(newListener *CompiledListener) (PluginInterface, error) {
	tplConditionClone, err := p.config.Condition.CloneForListener(newListener)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to clone condition template")
	}

	var tplDelayClone *ListenerTemplate
	if p.config.Delay != nil {
		_tplDelayClone, err := p.config.Delay.CloneForListener(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone delay template")
		}
		tplDelayClone = _tplDelayClone
	}

	return &PluginRetry{
		PluginBase:   p.PluginBase,
		config:       p.config,
		tplCondition: tplConditionClone,
		tplDelay:     tplDelayClone,
	}, nil
}

func (p *PluginRetry) HookShouldRetry(currentHookRetryInfo *HookShouldRetryInfo, args map[string]interface{}, commandResult *ExecCommandResult) (*time.Duration, map[string]interface{}, error) {
	if p.config.MaxElapsed != nil && currentHookRetryInfo.Elapsed > *p.config.MaxElapsed {
		// Do not retry if we exceed the max allowed execution time
		return nil, nil, errors.Errorf("max execution time reached (%s), cannot retry", p.config.MaxElapsed.String())
	}

	var maxRetries *int
	if p.config.MaxRetries != nil {
		maxRetries = p.config.MaxRetries
	} else if p.config.MaxElapsed == nil {
		max := pluginRetryDefaultMaxRetries
		maxRetries = &max
	}

	if maxRetries != nil && currentHookRetryInfo.RetryCount > *maxRetries {
		return nil, nil, errors.Errorf("max amount of retries reached (%d), cannot retry", *maxRetries)
	}

	newArgs := make(map[string]interface{})
	for key, val := range args {
		newArgs[key] = val
	}

	currentRetryInfo := &PluginRetryInfo{
		Elapsed:        currentHookRetryInfo.Elapsed,
		RetryCount:     currentHookRetryInfo.RetryCount,
		PreviousResult: commandResult,
	}

	newArgs[pluginRetryKeyRetryInfo] = *currentRetryInfo

	{
		isTrue, err := p.tplCondition.IsTrue(newArgs)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed to verify retry condition")
		}

		if !isTrue {
			return nil, nil, nil
		}
	}

	if p.tplDelay == nil {
		delay := pluginRetryDefaultDelay
		return &delay, newArgs, nil
	}

	delayStr, err := p.tplDelay.Execute(newArgs)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to evaluate retry delay template")
	}

	delayStr = strings.TrimSpace(delayStr)

	d, err := time.ParseDuration(delayStr)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to parse retry delay: %s", delayStr)
	}

	return &d, newArgs, nil
}
