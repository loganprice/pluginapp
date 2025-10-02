package plugin

import (
	"context"
)

// Version information
const (
	APIVersion = "1.0.0"
)

// Feature flags for plugin capabilities
const (
	FeatureProgress     = "progress"
	FeatureParamSchema  = "param-schema"
	FeatureRichErrors   = "rich-errors"
	FeatureInteractive  = "interactive"
	FeatureCancellation = "cancellation"
)

// Plugin is the interface that plugins must implement
type Plugin interface {
	GetInfo(ctx context.Context) (*PluginInfo, error)
	Execute(ctx context.Context, params map[string]string, output OutputHandler) error
	ReportExecutionSummary(startTime, endTime int64, success bool, err error, metadata map[string]string, metrics map[string]float64) (*ExecutionSummary, error)
	ValidateParameters(params map[string]string) error
	Close() error
}
