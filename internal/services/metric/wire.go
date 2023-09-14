package metric

import (
	"github.com/google/wire"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

var WireSet = wire.NewSet(
	ProvideCollector,
)

func ProvideCollector(
	config *types.Config,
	userStore store.PrincipalStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	executionStore store.ExecutionStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Collector, error) {
	job := &Collector{
		hostname:       config.InstanceID,
		enabled:        config.Metric.Enabled,
		endpoint:       config.Metric.Endpoint,
		token:          config.Metric.Token,
		userStore:      userStore,
		repoStore:      repoStore,
		pipelineStore:  pipelineStore,
		executionStore: executionStore,
		scheduler:      scheduler,
	}

	err := executor.Register(jobType, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}
