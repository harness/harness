package gogitlab

import (
	"encoding/json"
	"time"
)

const (
	repo_url_branches = "/projects/:id/repository/branches"         // List repository branches
	repo_url_branch   = "/projects/:id/repository/branches/:branch" // Get a specific branch of a project.
	repo_url_tags     = "/projects/:id/repository/tags"             // List project repository tags
	repo_url_commits  = "/projects/:id/repository/commits"          // List repository commits
	repo_url_tree     = "/projects/:id/repository/tree"             // List repository tree
	repo_url_raw_file = "/projects/:id/repository/blobs/:sha"       // Get raw file content for specific commit/branch
)

type BranchCommit struct {
	Id               string  `json:"id,omitempty"`
	Tree             string  `json:"tree,omitempty"`
	AuthoredDateRaw  string  `json:"authored_date,omitempty"`
	CommittedDateRaw string  `json:"committed_date,omitempty"`
	Message          string  `json:"message,omitempty"`
	Author           *Person `json:"author,omitempty"`
	Committer        *Person `json:"committer,omitempty"`
	/*
			"parents": [
			  {"id": "9b0c4b08e7890337fc8111e66f809c8bbec467a9"},
		      {"id": "3ac634dca850cab70ab14b43ad6073d1e0a7827f"}
		    ]
	*/
}

type Branch struct {
	Name      string        `json:"name,omitempty"`
	Protected bool          `json:"protected,omitempty"`
	Commit    *BranchCommit `json:"commit,omitempty"`
}

type Tag struct {
	Name      string        `json:"name,omitempty"`
	Protected bool          `json:"protected,omitempty"`
	Commit    *BranchCommit `json:"commit,omitempty"`
}

type Commit struct {
	Id           string
	Short_Id     string
	Title        string
	Author_Name  string
	Author_Email string
	Created_At   string
	CreatedAt    time.Time
}

/*
Get a list of repository branches from a project, sorted by name alphabetically.

    GET /projects/:id/repository/branches

Parameters:

    id The ID of a project

Usage:

	branches, err := gitlab.RepoBranches("your_projet_id")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, branch := range branches {
		fmt.Printf("%+v\n", branch)
	}
*/
func (g *Gitlab) RepoBranches(id string) ([]*Branch, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_branches, map[string]string{":id": id})

	var branches []*Branch

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &branches)
	}

	return branches, err
}

/*
Get a single project repository branch.

    GET /projects/:id/repository/branches/:branch

Parameters:

    id     The ID of a project
    branch The name of the branch

*/
func (g *Gitlab) RepoBranch(id, refName string) (*Branch, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_branch, map[string]string{
		":id":     id,
		":branch": refName,
	})

	branch := new(Branch)

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &branch)
	}
	return branch, err
}

/*
Get a list of repository tags from a project, sorted by name in reverse alphabetical order.

    GET /projects/:id/repository/tags

Parameters:

    id The ID of a project

Usage:

	tags, err := gitlab.RepoTags("your_projet_id")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, tag := range tags {
		fmt.Printf("%+v\n", tag)
	}
*/
func (g *Gitlab) RepoTags(id string) ([]*Tag, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_tags, map[string]string{":id": id})

	var tags []*Tag

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &tags)
	}

	return tags, err
}

/*
Get a list of repository commits in a project.

    GET /projects/:id/repository/commits

Parameters:

    id      The ID of a project
	refName The name of a repository branch or tag or if not given the default branch

Usage:

	commits, err := gitlab.RepoCommits("your_projet_id")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, commit := range commits {
		fmt.Printf("%+v\n", commit)
	}
*/
func (g *Gitlab) RepoCommits(id string) ([]*Commit, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_commits, map[string]string{":id": id})

	var commits []*Commit

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &commits)
		if err == nil {
			for _, commit := range commits {
				t, _ := time.Parse(dateLayout, commit.Created_At)
				commit.CreatedAt = t
			}
		}
	}

	return commits, err
}

/*
Get Raw file content
*/
func (g *Gitlab) RepoRawFile(id, sha, filepath string) ([]byte, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_raw_file, map[string]string{
		":id":  id,
		":sha": sha,
	})
	url += "&filepath=" + filepath

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)

	return contents, err
}
