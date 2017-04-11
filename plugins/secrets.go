package plugins

import (
	"fmt"

	"github.com/drone/drone/model"
)

type secretPlugin struct {
	endpoint string
}

// NewSecret returns a new secret plugin.
func NewSecret(endpoint string) interface{} {
	return secretPlugin{endpoint}
}

func (s *secretPlugin) SecretFind(repo *model.Repo, name string) (*model.Secret, error) {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", s.endpoint, repo.Owner, repo.Name, name)
	out := new(model.Secret)
	err := do("GET", path, nil, out)
	return out, err
}

func (s *secretPlugin) SecretList(repo *model.Repo) ([]*model.Secret, error) {
	path := fmt.Sprintf("%s/secrets/%s/%s", s.endpoint, repo.Owner, repo.Name)
	out := []*model.Secret{}
	err := do("GET", path, nil, out)
	return out, err
}

func (s *secretPlugin) SecretCreate(repo *model.Repo, in *model.Secret) error {
	path := fmt.Sprintf("%s/secrets/%s/%s", s.endpoint, repo.Owner, repo.Name)
	return do("POST", path, in, nil)
}

func (s *secretPlugin) SecretUpdate(repo *model.Repo, in *model.Secret) error {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", s.endpoint, repo.Owner, repo.Name, in.Name)
	return do("PATCH", path, in, nil)
}

func (s *secretPlugin) SecretDelete(repo *model.Repo, name string) error {
	path := fmt.Sprintf("%s/secrets/%s/%s/%s", s.endpoint, repo.Owner, repo.Name, name)
	return do("DELETE", path, nil, nil)
}
