package profiler

import "github.com/rs/zerolog/log"

type NoopProfiler struct {
}

func (noopProfiler *NoopProfiler) StartProfiling(serviceName, serviceVersion string) {
	//do nothing
	log.Info().Msg("Not starting profiler")
}
