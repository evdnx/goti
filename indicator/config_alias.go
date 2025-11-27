package indicator

import "github.com/evdnx/goti/config"

// Re-export config defaults and types so existing indicator code can stay lean.
type IndicatorConfig = config.IndicatorConfig

const (
	DefaultAMDOOverbought = config.DefaultAMDOOverbought
	DefaultAMDOOversold   = config.DefaultAMDOOversold
)

func DefaultConfig() IndicatorConfig {
	return config.DefaultConfig()
}
