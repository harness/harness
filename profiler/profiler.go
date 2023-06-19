package profiler

import "fmt"

type Profiler interface {
	StartProfiling(serviceName, serviceVersion string)
}

type ProfilerType string

const (
	ProfilerTypeGCP ProfilerType = "GCP"
)

func ParseProfiler(profiler string) (ProfilerType, bool) {
	if profiler == string(ProfilerTypeGCP) {
		return ProfilerTypeGCP, true
	}
	return "", false
}

func GetProfiler(profiler ProfilerType) (Profiler, error) {
	switch profiler {
	case ProfilerTypeGCP:
		return &GCPProfiler{}, nil
	default:
		return &NoopProfiler{}, fmt.Errorf("profiler '%s' not supported", profiler)
	}
}
