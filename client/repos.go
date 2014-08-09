package client

import (
	"fmt"

	"github.com/drone/drone/shared/model"
)

type RepoService struct {
	*Client
}

// GET /v1/repos/{host}/{owner}/{name}
func (s *RepoService) Get() (*model.Repo, error) {
	var path = fmt.Sprintf("/v1/repos/%s/%s/%s")
	var repo = model.Repo{}
	var err = s.run("PUT", path, nil, &repo)
	return &repo, err
}

// PUT /v1/repos/{host}/{owner}/{name}
func (s *RepoService) Update(repo *model.Repo) (*model.Repo, error) {
	var path = fmt.Sprintf("/v1/repos/%s/%s/%s")
	var result = model.Repo{}
	var err = s.run("PUT", path, &repo, &result)
	return &result, err
}

// POST /v1/repos/{host}/{owner}/{name}
func (s *RepoService) Enable(host, owner, name string) error {
	var path = fmt.Sprintf("/v1/repos/%s/%s/%s", host, owner, name)
	return s.run("POST", path, nil, nil)
}

// DELETE /v1/repos/{host}/{owner}/{name}
func (s *RepoService) Disable(host, owner, name string) error {
	var path = fmt.Sprintf("/v1/repos/%s/%s/%s", host, owner, name)
	return s.run("DELETE", path, nil, nil)
}

// GET /v1/user/repos
func (s *RepoService) List() ([]*model.Repo, error) {
	var repos []*model.Repo
	var err = s.run("GET", "/v1/user/repos", nil, &repos)
	return repos, err
}
