package gogitlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	projects_url               = "/projects"                                            // Get a list of projects owned by the authenticated user
	projects_search_url        = "/projects/search/:query"                              // Search for projects by name
	project_url                = "/projects/:id"                                        // Get a specific project, identified by project ID or NAME
	project_url_events         = "/projects/:id/events"                                 // Get project events
	project_url_branches       = "/projects/:id/repository/branches"                    // Lists all branches of a project
	project_url_members        = "/projects/:id/members"                                // List project team members
	project_url_member         = "/projects/:id/members/:user_id"                       // Get project team member
	project_url_merge_requests = "/projects/:id/merge_requests"                         // List all merge requests of a project
	merge_request_url_notes    = "/projects/:id/merge_requests/:merge_request_id/notes" // Manage comments for a given merge request
)

type Member struct {
	Id        int
	Username  string
	Email     string
	Name      string
	State     string
	CreatedAt string `json:"created_at,omitempty"`
	// AccessLevel int
}

type Namespace struct {
	Id          int
	Name        string
	Path        string
	Description string
	Owner_Id    int
	Created_At  string
	Updated_At  string
}

type ProjectAccess struct {
	AccessLevel       int `json:"access_level,omitempty"`
	NotificationLevel int `json:"notification_level,omitempty"`
}

type GroupAccess struct {
	AccessLevel       int `json:"access_level,omitempty"`
	NotificationLevel int `json:"notification_level,omitempty"`
}

type Permissions struct {
	ProjectAccess *ProjectAccess `json:"project_access,omitempty"`
	GroupAccess   *GroupAccess   `json:"group_access,omitempty"`
}

// A gitlab project
type Project struct {
	Id                   int          `json:"id,omitempty"`
	Name                 string       `json:"name,omitempty"`
	Description          string       `json:"description,omitempty"`
	DefaultBranch        string       `json:"default_branch,omitempty"`
	Owner                *Member      `json:"owner,omitempty"`
	Public               bool         `json:"public,omitempty"`
	Path                 string       `json:"path,omitempty"`
	PathWithNamespace    string       `json:"path_with_namespace,omitempty"`
	IssuesEnabled        bool         `json:"issues_enabled,omitempty"`
	MergeRequestsEnabled bool         `json:"merge_requests_enabled,omitempty"`
	WallEnabled          bool         `json:"wall_enabled,omitempty"`
	WikiEnabled          bool         `json:"wiki_enabled,omitempty"`
	CreatedAtRaw         string       `json:"created_at,omitempty"`
	Namespace            *Namespace   `json:"namespace,omitempty"`
	SshRepoUrl           string       `json:"ssh_url_to_repo"`
	HttpRepoUrl          string       `json:"http_url_to_repo"`
	Url                  string       `json:"web_url"`
	Permissions          *Permissions `json:"permissions,omitempty"`
}

type MergeRequest struct {
	Id int `json:"id,omitempty"`
	// IId
	TargetBranch string  `json:"target_branch,omitempty"`
	SourceBranch string  `json:"source_branch,omitempty"`
	ProjectId    int     `json:"project_id,omitempty"`
	Title        string  `json:"title,omitempty"`
	State        string  `json:"state,omitempty"`
	Upvotes      int     `json:"upvotes,omitempty"`
	Downvotes    int     `json:"downvotes,omitempty"`
	Author       *Member `json:"author,omitempty"`
	Assignee     *Member `json:"assignee,omitempty"`
	Description  string  `json:"description,omitempty"`
}

type MergeRequestNote struct {
	Attachment interface{} `json:"attachment"`
	Body       string      `json:"body"`
	CreatedAt  string      `json:"created_at"`
	Id         int         `json:"id"`
	Author     *Member     `json:"author"`
}

/*
Get a list of all projects owned by the authenticated user.
*/
func (g *Gitlab) AllProjects() ([]*Project, error) {
	var per_page = 100
	var projects []*Project

	for i := 1; true; i++ {
		contents, err := g.Projects(i, per_page)
		if err != nil {
			return projects, err
		}

		for _, value := range contents {
			projects = append(projects, value)
		}

		if len(projects) == 0 {
			break
		}

		if len(projects)/i < per_page {
			break
		}
	}

	return projects, nil
}

/*
Get a list of projects owned by the authenticated user.
*/
func (g *Gitlab) Projects(page int, per_page int) ([]*Project, error) {

	url := g.ResourceUrlQuery(projects_url, nil, map[string]string{"page": strconv.Itoa(page), "per_page": strconv.Itoa(per_page)})

	var projects []*Project

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &projects)
	}

	return projects, err
}

/*
Get a specific project, identified by project ID or NAME,
which is owned by the authentication user.
Namespaced project may be retrieved by specifying the namespace
and its project name like this:

	`namespace%2Fproject-name`

*/
func (g *Gitlab) Project(id string) (*Project, error) {

	url, opaque := g.ResourceUrlRaw(project_url, map[string]string{":id": id})

	var project *Project

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &project)
	}

	return project, err
}

/*
Lists all branches of a project.
*/
func (g *Gitlab) ProjectBranches(id string) ([]*Branch, error) {

	url, opaque := g.ResourceUrlRaw(project_url_branches, map[string]string{":id": id})

	var branches []*Branch

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &branches)
	}

	return branches, err
}

func (g *Gitlab) ProjectMembers(id string) ([]*Member, error) {
	url, opaque := g.ResourceUrlRaw(project_url_members, map[string]string{":id": id})

	var members []*Member

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &members)
	}

	return members, err
}

/*
Lists all merge requests of a project.
*/
func (g *Gitlab) ProjectMergeRequests(id string, page int, per_page int, state string) ([]*MergeRequest, error) {
	par := map[string]string{":id": id}
	qry := map[string]string{
		"state":    state,
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(per_page)}
	url := g.ResourceUrlQuery(project_url_merge_requests, par, qry)

	var mr []*MergeRequest

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &mr)
	}

	return mr, err
}

/*
Lists all comments on merge request.
*/
func (g *Gitlab) MergeRequestNotes(id string, merge_request_id string, page int, per_page int) ([]*MergeRequestNote, error) {
	par := map[string]string{":id": id, ":merge_request_id": merge_request_id}
	qry := map[string]string{
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(per_page)}
	url := g.ResourceUrlQuery(merge_request_url_notes, par, qry)

	var mr []*MergeRequestNote

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err == nil {
		err = json.Unmarshal(contents, &mr)
	}

	return mr, err
}

/*
Creates a new comment on a merge request.
*/
func (g *Gitlab) SendMergeRequestComment(id string, merge_request_id string, comment string) (*MergeRequestNote, error) {
	par := map[string]string{":id": id, ":merge_request_id": merge_request_id}
	url := g.ResourceUrlQuery(merge_request_url_notes, par, map[string]string{})

	var mr *MergeRequestNote

	contents, err := g.buildAndExecRequest("POST", url, []byte(fmt.Sprintf("body=%s", comment)))
	if err == nil {
		err = json.Unmarshal(contents, &mr)
	}

	return mr, err
}

/*
Get single project id.

    GET /projects/search/:query

Parameters:

    namespace The namespace of a project
    name      The id of a project

*/
func (g *Gitlab) SearchProjectId(namespace string, name string) (id int, err error) {

	url, opaque := g.ResourceUrlRaw(projects_search_url, map[string]string{
		":query": strings.ToLower(name),
	})

	var projects []*Project

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &projects)
	} else {
		return id, err
	}

	for _, project := range projects {
		if project.Namespace.Name == namespace && strings.ToLower(project.Name) == strings.ToLower(name) {
			id = project.Id
		}
	}

	return id, err
}
