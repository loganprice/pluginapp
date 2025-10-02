
package plugin

// ExecutionSummary contains all information about a plugin's execution
type ExecutionSummary struct {
	PluginName string
	StartTime  int64
	EndTime    int64
	Duration   float64 // in milliseconds
	Success    bool
	Error      error
	Metadata   map[string]string
	Metrics    map[string]float64
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name            string
	Version         string
	Description     string
	ParameterSchema map[string]ParameterSpec
}

// ParameterSpec describes a plugin parameter
type ParameterSpec struct {
	Name          string
	Description   string
	Required      bool
	DefaultValue  string
	Type          string
	AllowedValues []string
}

// Progress represents execution progress information
type Progress struct {
	PercentComplete float32
	Stage           string
	CurrentStep     int32
	TotalSteps      int32
}

// OutputHandler handles different types of plugin output
type OutputHandler interface {
	OnOutput(msg string) error
	OnProgress(progress Progress) error
	OnError(code, message, details string) error
}
