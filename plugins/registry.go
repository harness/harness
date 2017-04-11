package plugins

import (
	"fmt"

	"github.com/drone/drone/model"
)

type registryPlugin struct {
	endpoint string
}

// NewRegistry returns a new registry plugin.
func NewRegistry(endpoint string) interface{} {
	return registryPlugin{endpoint}
}

func (r *registryPlugin) RegistryFind(repo *model.Repo, name string) (*model.Registry, error) {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", r.endpoint, repo.Owner, repo.Name, name)
	out := new(model.Registry)
	err := do("GET", path, nil, out)
	return out, err
}

func (r *registryPlugin) RegistryList(repo *model.Repo) ([]*model.Registry, error) {
	path := fmt.Sprintf("%s/registry/%s/%s", r.endpoint, repo.Owner, repo.Name)
	out := []*model.Registry{}
	err := do("GET", path, nil, out)
	return out, err
}

func (r *registryPlugin) RegistryCreate(repo *model.Repo, in *model.Registry) error {
	path := fmt.Sprintf("%s/registry/%s/%s", r.endpoint, repo.Owner, repo.Name)
	return do("PATCH", path, in, nil)
}

func (r *registryPlugin) RegistryUpdate(repo *model.Repo, in *model.Registry) error {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", r.endpoint, repo.Owner, repo.Name, in.Address)
	return do("PATCH", path, in, nil)
}

func (r *registryPlugin) RegistryDelete(repo *model.Repo, name string) error {
	path := fmt.Sprintf("%s/registry/%s/%s/%s", r.endpoint, repo.Owner, repo.Name, name)
	return do("DELETE", path, nil, nil)
}
