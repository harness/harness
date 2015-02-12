package client

import (
	"fmt"

	"github.com/drone/drone/shared/model"
)

type RepoService struct {
	*Client
}

// GET /api/repos/{host}/{owner}/{name}
func (s *RepoService) Get(host, owner, name string) (*model.Repo, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s", host, owner, name)
	var repo = model.Repo{}
	var err = s.run("GET", path, nil, &repo)
	return &repo, err
}

// PUT /api/repos/{host}/{owner}/{name}
func (s *RepoService) Update(repo *model.Repo) (*model.Repo, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s", repo.Host, repo.Owner, repo.Name)
	var result = model.Repo{}
	var err = s.run("PUT", path, &repo, &result)
	return &result, err
}

// POST /api/repos/{host}/{owner}/{name}
func (s *RepoService) Enable(host, owner, name string) error {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s", host, owner, name)
	return s.run("POST", path, nil, nil)
}

// POST /api/repos/{host}/{owner}/{name}/deactivate
func (s *RepoService) Disable(host, owner, name string) error {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/deactivate", host, owner, name)
	return s.run("POST", path, nil, nil)
}

// DELETE /api/repos/{host}/{owner}/{name}?remove=true
func (s *RepoService) Delete(host, owner, name string) error {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s", host, owner, name)
	return s.run("DELETE", path, nil, nil)
}

// PUT /api/repos/{host}/{owner}/{name}
func (s *RepoService) SetKey(host, owner, name, pub, priv string) error {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s", host, owner, name)
	var in = struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	}{pub, priv}
	return s.run("PUT", path, &in, nil)
}

// GET /api/user/repos
func (s *RepoService) List() ([]*model.Repo, error) {
	var repos []*model.Repo
	var err = s.run("GET", "/api/user/repos", nil, &repos)
	return repos, err
}
