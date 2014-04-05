package bitbucket

import (
	"fmt"
)

type Repo struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Owner    string `json:"owner"`
	Scm      string `json:"scm"`
	Logo     string `json:"logo"`
	Language string `json:"language"`
	Private  bool   `json:"is_private"`
	IsFork   bool   `json:"is_fork"`
	ForkOf   *Repo  `json:"fork_of"`
}

type Branch struct {
	Branch    string        `json:"branch"`
	Message   string        `json:"message"`
	Author    string        `json:"author"`
	RawAuthor string        `json:"raw_author"`
	Node      string        `json:"node"`
	RawNode   string        `json:"raw_node"`
	Files     []*BranchFile `json:"files"`
}

type BranchFile struct {
	File string `json:"file"`
	Type string `json:"type"`
}

type RepoResource struct {
	client *Client
}

// Gets the repositories owned by the individual or team account.
func (r *RepoResource) List() ([]*Repo, error) {
	repos := []*Repo{}
	const path = "/user/repositories"

	if err := r.client.do("GET", path, nil, nil, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// Gets the repositories list from the account's dashboard.
func (r *RepoResource) ListDashboard() ([]*Account, error) {
	var m [][]interface{}
	const path = "/user/repositories/dashboard"

	if err := r.client.do("GET", path, nil, nil, &m); err != nil {
		return nil, err
	}

	return unmarshalAccounts(m), nil
}

// Gets the repositories list from the account's dashboard, and
// converts the response to a list of Repos, instead of a
// list of Accounts.
func (r *RepoResource) ListDashboardRepos() ([]*Repo, error) {
	accounts, err := r.ListDashboard()
	if err != nil {
		return nil, nil
	}

	repos := []*Repo{}
	for _, acct := range accounts {
		repos = append(repos, acct.Repos...)
	}

	return repos, nil
}

// Gets the list of Branches for the repository
func (r *RepoResource) ListBranches(owner, slug string) ([]*Branch, error) {
	branchMap := map[string]*Branch{}
	path := fmt.Sprintf("/repositories/%s/%s/branches", owner, slug)

	if err := r.client.do("GET", path, nil, nil, &branchMap); err != nil {
		return nil, err
	}

	// The list is returned in a map ...
	// we really want a slice
	branches := []*Branch{}
	for _, branch := range branchMap {
		branches = append(branches, branch)
	}

	return branches, nil
}

// Gets the repositories list for the named user.
func (r *RepoResource) ListUser(owner string) ([]*Repo, error) {
	repos := []*Repo{}
	path := fmt.Sprintf("/repositories/%s", owner)

	if err := r.client.do("GET", path, nil, nil, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// Gets the named repository.
func (r *RepoResource) Find(owner, slug string) (*Repo, error) {
	repo := Repo{}
	path := fmt.Sprintf("/repositories/%s/%s", owner, slug)

	if err := r.client.do("GET", path, nil, nil, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// -----------------------------------------------------------------------------
// Helper Functions to parse odd Bitbucket JSON structure

func unmarshalAccounts(m [][]interface{}) []*Account {

	accts := []*Account{}
	for i := range m {
		a := Account{}
		for j := range m[i] {
			switch v := m[i][j].(type) {
			case []interface{}:
				a.Repos = unmarshalRepos(v)
			case map[string]interface{}:
				a.User = unmarshalUser(v)
			default: // Unknown...return error?
			}
		}
		accts = append(accts, &a)
	}

	return accts
}

func unmarshalUser(m map[string]interface{}) *User {
	u := User{}
	for k, v := range m {
		switch k {
		case "username":
			u.Username = v.(string)
		case "first_name":
			u.FirstName = v.(string)
		case "last_name":
			u.LastName = v.(string)
		case "display_name":
			u.DisplayName = v.(string)
		case "avatar":
			u.Avatar = v.(string)
		case "is_team":
			u.IsTeam = v.(bool)
		}
	}
	return &u
}

func unmarshalRepo(m map[string]interface{}) *Repo {
	r := Repo{}
	for k, v := range m {
		// make sure v.(type) is correct type each time?
		switch k {
		case "name":
			r.Name = v.(string)
		case "slug":
			r.Slug = v.(string)
		case "owner":
			r.Owner = v.(string)
		case "scm":
			r.Scm = v.(string)
		case "logo":
			r.Logo = v.(string)
		case "language":
			r.Language = v.(string)
		case "is_private":
			r.Private = v.(bool)
		}

	}
	return &r
}

func unmarshalRepos(m []interface{}) []*Repo {
	r := []*Repo{}
	for i := range m {
		switch v := m[i].(type) {
		case map[string]interface{}:
			r = append(r, unmarshalRepo(v))
			//default: fmt.Println("BAD")
		}
	}
	return r
}
