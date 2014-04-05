package github

import (
	"fmt"
)

// Owner represents the owner of a Github Repository.
type Owner struct {
	Type  string `json:"type"`
	Login string `json:"login"`
}

// Permissions
type Permissions struct {
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
	Admin bool `json:"admin"`
}

type Source struct {
	Owner *Owner `json:"owner"`
}

// Repo represents a Github-hosted Git Repository.
type Repo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	Fork     bool   `json:"fork"`
	SshUrl   string `json:"ssh_url"`
	GitUrl   string `json:"git_url"`
	CloneUrl string `json:"clone_url"`
	HtmlUrl  string `json:"html_url"`

	Owner       *Owner       `json:"owner"`
	Permissions *Permissions `json:"permissions"`
	Source      *Source      `json:"source"`
}

type RepoResource struct {
	client *Client
}

func (r *RepoResource) List() ([]*Repo, error) {
	repos := []*Repo{}
	if err := r.client.do("GET", "/user/repos?per_page=100&type=owner", nil, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (r *RepoResource) ListUser(username string) ([]*Repo, error) {
	repos := []*Repo{}
	path := fmt.Sprintf("/users/%s/repos?per_page=100&type=owner", username)
	if err := r.client.do("GET", path, nil, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (r *RepoResource) ListOrg(orgname string) ([]*Repo, error) {
	repos := []*Repo{}
	path := fmt.Sprintf("/orgs/%s/repos?page=0&per_page=100&type=owner", orgname)
	if err := r.client.do("GET", path, nil, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (r *RepoResource) ListAll() ([]*Repo, error) {
	repos, err := r.List()
	if err != nil {
		return nil, err
	}

	// now get all teams
	orgs, _ := r.client.Orgs.List()
	for _, org := range orgs {
		orgRepos, _ := r.ListOrg(org.Login)
		if orgRepos != nil {
			repos = append(repos, orgRepos...)
		}
	}

	return repos, nil
}

func (r *RepoResource) Find(owner, repo string) (*Repo, error) {
	rpo := Repo{}
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	if err := r.client.do("GET", path, nil, &rpo); err != nil {
		return nil, err
	}

	return &rpo, nil
}

func (r *RepoResource) FindRaw(owner, repo string) ([]byte, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	return r.client.raw("GET", path)
}

type CommitStatus struct {
	State       string `json:"state"` // pending, success, error, failure
	TargetUrl   string `json:"target_url"`
	Description string `json:"description"`
}

func (r *RepoResource) CreateStatus(owner, repo, state, link, comment, sha string) error {
	in := CommitStatus{state, link, comment}
	var out interface{}

	path := fmt.Sprintf("/repos/%s/%s/statuses/%s", owner, repo, sha)
	if err := r.client.do("POST", path, &in, &out); err != nil {
		return err
	}

	return nil
}
