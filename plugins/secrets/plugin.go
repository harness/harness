package secrets

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/internal"
)

type plugin struct {
	endpoint string
}

// NewRemote returns a new remote secret service.
func NewRemote(endpoint string) model.SecretService {
	return &plugin{endpoint}
}

func (p *plugin) SecretFind(repo *model.Repo, name string) (*model.Secret, error) {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, name)
	out := new(model.Secret)
	err := internal.Send("GET", path, nil, out)
	return out, err
}

func (p *plugin) SecretList(repo *model.Repo) ([]*model.Secret, error) {
	path := fmt.Sprintf("%s/secrets/%s/%s", p.endpoint, repo.Owner, repo.Name)
	out := []*model.Secret{}
	err := internal.Send("GET", path, nil, out)
	return out, err
}

func (p *plugin) SecretListBuild(repo *model.Repo, build *model.Build) ([]*model.Secret, error) {
	path := fmt.Sprintf("%s/secrets/%s/%s/%d", p.endpoint, repo.Owner, repo.Name, build.Number)
	out := []*model.Secret{}
	err := internal.Send("GET", path, nil, out)
	return out, err
}

func (p *plugin) SecretCreate(repo *model.Repo, in *model.Secret) error {
	path := fmt.Sprintf("%s/secrets/%s/%s", p.endpoint, repo.Owner, repo.Name)
	return internal.Send("POST", path, in, nil)
}

func (p *plugin) SecretUpdate(repo *model.Repo, in *model.Secret) error {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, in.Name)
	return internal.Send("PATCH", path, in, nil)
}

func (p *plugin) SecretDelete(repo *model.Repo, name string) error {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, name)
	return internal.Send("DELETE", path, nil, nil)
}
