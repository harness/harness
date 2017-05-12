package secrets

import (
	"github.com/drone/drone/model"
)

type builtin struct {
	store model.SecretStore
}

// New returns a new local secret service.
func New(store model.SecretStore) model.SecretService {
	return &builtin{store}
}

func (b *builtin) SecretFind(repo *model.Repo, name string) (*model.Secret, error) {
	return b.store.SecretFind(repo, name)
}

func (b *builtin) SecretList(repo *model.Repo) ([]*model.Secret, error) {
	return b.store.SecretList(repo)
}

func (b *builtin) SecretListBuild(repo *model.Repo, build *model.Build) ([]*model.Secret, error) {
	return b.store.SecretList(repo)
}

func (b *builtin) SecretCreate(repo *model.Repo, in *model.Secret) error {
	return b.store.SecretCreate(in)
}

func (b *builtin) SecretUpdate(repo *model.Repo, in *model.Secret) error {
	return b.store.SecretUpdate(in)
}

func (b *builtin) SecretDelete(repo *model.Repo, name string) error {
	secret, err := b.store.SecretFind(repo, name)
	if err != nil {
		return err
	}
	return b.store.SecretDelete(secret)
}
