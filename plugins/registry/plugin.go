package registry

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/internal"
)

type plugin struct {
	endpoint string
}

// NewRemote returns a new remote registry service.
func NewRemote(endpoint string) model.RegistryService {
	return &plugin{endpoint}
}

func (p *plugin) RegistryFind(repo *model.Repo, name string) (*model.Registry, error) {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, name)
	out := new(model.Registry)
	err := internal.Send("GET", path, nil, out)
	return out, err
}

func (p *plugin) RegistryList(repo *model.Repo) ([]*model.Registry, error) {
	path := fmt.Sprintf("%s/registry/%s/%s", p.endpoint, repo.Owner, repo.Name)
	out := []*model.Registry{}
	err := internal.Send("GET", path, nil, out)
	return out, err
}

func (p *plugin) RegistryCreate(repo *model.Repo, in *model.Registry) error {
	path := fmt.Sprintf("%s/registry/%s/%s", p.endpoint, repo.Owner, repo.Name)
	return internal.Send("PATCH", path, in, nil)
}

func (p *plugin) RegistryUpdate(repo *model.Repo, in *model.Registry) error {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, in.Address)
	return internal.Send("PATCH", path, in, nil)
}

func (p *plugin) RegistryDelete(repo *model.Repo, name string) error {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, name)
	return internal.Send("DELETE", path, nil, nil)
}
