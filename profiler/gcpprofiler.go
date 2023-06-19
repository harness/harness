package profiler

import (
	"cloud.google.com/go/profiler"
	"github.com/rs/zerolog/log"
)

type GCPProfiler struct {
}

func (gcpProfiler *GCPProfiler) StartProfiling(serviceName, serviceVersion string) {
	// Need to add env/namespace with service name to uniquely identify this
	cfg := profiler.Config{
		Service:        serviceName,
		ServiceVersion: serviceVersion,
	}

	if err := profiler.Start(cfg); err != nil {
		log.Warn().Err(err).Msg("unable to start profiler")
	}
}
