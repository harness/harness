package gogitlab

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const (
	repo_url_branches        = "/projects/:id/repository/branches"              // List repository branches
	repo_url_branch          = "/projects/:id/repository/branches/:branch"      // Get a specific branch of a project.
	repo_url_tags            = "/projects/:id/repository/tags"                  // List project repository tags
	repo_url_commits         = "/projects/:id/repository/commits"               // List repository commits
	repo_url_commit_comments = "/projects/:id/repository/commits/:sha/comments" // New comment or list of commit comments
	repo_url_tree            = "/projects/:id/repository/tree"                  // List repository tree
	repo_url_raw_file        = "/projects/:id/repository/blobs/:sha"            // Get raw file content for specific commit/branch
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

type File struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Mode string `json:"mode,omitempty"`

	Children []*File
}

type CommitComment struct {
	Author   *Member `json:"author,omitempty"`
	Line     int     `json:"line,omitempty"`
	LineType string  `json:"line_type,omitempty"`
	Note     string  `json:"note,omitempty"`
	Path     string  `json:"path,omitempty"`
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
Get a list of comments in a repository commit.

    GET /projects/:id/repository/commits/:sha/comments

Parameters:

	id  The ID of a project
	sha The sha of the commit

Usage:

	comments, err := gitlab.RepoCommitComments("your_projet_id", "commit_sha")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, comment := range comments {
		fmt.Printf("%+v\n", comment)
	}
*/
func (g *Gitlab) RepoCommitComments(id string, sha string) ([]*CommitComment, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_commit_comments, map[string]string{":id": id, ":sha": sha})

	var comments []*CommitComment

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &comments)
	}

	return comments, err
}

/*
Create a comment in a repository commit.

    POST /projects/:id/repository/commits/:sha/comments

Parameters:

	id   The ID of a project
	sha  The sha of the commit
	body The body of the comment

Usage:

	comment, err := gitlab.SendRepoCommitComment("your_projet_id", "commit_sha", "your comment goes here")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("%+v\n", comment)
*/
func (g *Gitlab) SendRepoCommitComment(id string, sha string, body string) (*CommitComment, error) {

	url, opaque := g.ResourceUrlRaw(repo_url_commit_comments, map[string]string{":id": id, ":sha": sha})

	var comment *CommitComment

	contents, err := g.buildAndExecRequestRaw("POST", url, opaque, []byte(fmt.Sprintf("note=%s", body)))
	if err == nil {
		err = json.Unmarshal(contents, &comment)
	}

	return comment, err
}

/*
Get Raw file content
*/
func (g *Gitlab) RepoRawFile(id, sha, filepath string) ([]byte, error) {
	url_ := g.ResourceUrlQuery(repo_url_raw_file, map[string]string{
		":id":  id,
		":sha": sha,
	}, map[string]string{
		"filepath": filepath,
	})

	p, err := url.Parse(url_)
	if err != nil {
		return nil, err
	}

	opaque := "//" + p.Host + p.Path
	contents, err := g.buildAndExecRequestRaw("GET", url_, opaque, nil)

	return contents, err
}

/*
Get Raw file content
*/
func (g *Gitlab) RepoTree(id, ref, path string) ([]*File, error) {

	url := g.ResourceUrlQuery(repo_url_tree, map[string]string{
		":id": id,
	}, map[string]string{
		"ref":  ref,
		"path": path,
	})

	var files []*File

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &files)
	}

	for _, f := range files {
		if f.Type == "tree" {
			f.Children, err = g.RepoTree(id, ref, path+"/"+f.Name)
		}
	}

	return files, err
}
