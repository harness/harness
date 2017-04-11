package registry

import (
	"github.com/drone/drone/model"
)

type builtin struct {
	store model.RegistryStore
}

// New returns a new local registry service.
func New(store model.RegistryStore) model.RegistryService {
	return &builtin{store}
}

func (b *builtin) RegistryFind(repo *model.Repo, name string) (*model.Registry, error) {
	return b.store.RegistryFind(repo, name)
}

func (b *builtin) RegistryList(repo *model.Repo) ([]*model.Registry, error) {
	return b.store.RegistryList(repo)
}

func (b *builtin) RegistryCreate(repo *model.Repo, in *model.Registry) error {
	return b.store.RegistryCreate(in)
}

func (b *builtin) RegistryUpdate(repo *model.Repo, in *model.Registry) error {
	return b.store.RegistryUpdate(in)
}

func (b *builtin) RegistryDelete(repo *model.Repo, addr string) error {
	registry, err := b.RegistryFind(repo, addr)
	if err != nil {
		return err
	}
	return b.store.RegistryDelete(registry)
}
