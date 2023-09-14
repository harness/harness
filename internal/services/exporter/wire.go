package exporter

import (
	"github.com/google/wire"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
)

var WireSet = wire.NewSet(
	ProvideSpaceExporter,
)

func ProvideSpaceExporter(
	urlProvider *url.Provider,
	git gitrpc.Interface,
	repoStore store.RepoStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
	encrypter encrypt.Encrypter,
) (*Repository, error) {
	exporter := &Repository{
		urlProvider: urlProvider,
		git:         git,
		repoStore:   repoStore,
		scheduler:   scheduler,
		encrypter:   encrypter,
	}

	err := executor.Register(jobType, exporter)
	if err != nil {
		return nil, err
	}

	return exporter, nil
}
