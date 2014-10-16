package client

import (
	"fmt"
	"io"

	"github.com/drone/drone/shared/model"
)

type CommitService struct {
	*Client
}

// GET /api/repos/{host}/{owner}/{name}/branch/{branch}/commit/{commit}
func (s *CommitService) Get(host, owner, name, branch, sha string) (*model.Commit, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/branches/%s/commits/%s", host, owner, name, branch, sha)
	var commit = model.Commit{}
	var err = s.run("GET", path, nil, &commit)
	return &commit, err
}

// GET /api/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}/console
func (s *CommitService) GetOutput(host, owner, name, branch, sha string) (io.ReadCloser, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/branches/%s/commits/%s/console", host, owner, name, branch, sha)
	resp, err := s.do("GET", path)
	if err != nil {
		return nil, nil
	}
	return resp.Body, nil
}

// POST /api/repos/{host}/{owner}/{name}/branches/{branch}/commits/{commit}?action=rebuild
func (s *CommitService) Rebuild(host, owner, name, branch, sha string) error {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/branches/%s/commits/%s?action=rebuild", host, owner, name, branch, sha)
	return s.run("POST", path, nil, nil)
}

// GET /api/repos/{host}/{owner}/{name}/feed
func (s *CommitService) List(host, owner, name string) ([]*model.Commit, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/feed", host, owner, name)
	var list []*model.Commit
	var err = s.run("GET", path, nil, &list)
	return list, err
}

// GET /api/repos/{host}/{owner}/{name}/branch/{branch}
func (s *CommitService) ListBranch(host, owner, name, branch string) ([]*model.Commit, error) {
	var path = fmt.Sprintf("/api/repos/%s/%s/%s/commits", host, owner, name)
	var list []*model.Commit
	var err = s.run("GET", path, nil, &list)
	return list, err
}
