package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Different types of Hooks. See http://developer.github.com/v3/repos/hooks/
const (
	// Any git push to a Repository.
	HookPush = "push"

	// Any time an Issue is opened or closed.
	HookIssues = "issues"

	// Any time an Issue is commented on.
	HookIssueComment = "issue_comment"

	// Any time a Commit is commented on.
	HookCommitComment = "commit_comment"

	// Any time a Pull Request is opened, closed, or synchronized.
	HookPullRequest = "pull_request"

	// Any time a Commit is commented on while inside a Pull Request review.
	HookRequestReivewComment = "pull_request_review_comment"

	// Any time a Wiki page is updated.
	HookGollum = "gollum"

	// Any time a User watches the Repository.
	HookWatch = "watch"

	// Any time a Download is added to the Repository.
	HookDownload = "download"

	// Any time a Repository is forked.
	HookForm = "fork"

	// Any time a patch is applied to the Repository from the Fork Queue.
	HookForkApply = "fork_apply"

	// Any time a User is added as a collaborator to a non-Organization Repository.
	HookMember = "member"

	// Any time a Repository changes from private to public.
	HookPublic = "public"

	// Any time a team is added or modified on a Repository.
	HookTeamAdd = "team_add"

	// Any time a Repository has a status update from the API
	HookStatus = "status"
)

type Hook struct {
	Id     int         `json:"id"`
	Name   string      `json:"name"`
	Active bool        `json:"active"`
	Events []string    `json:"events"`
	Config *HookConfig `json:"config"`
}

type HookConfig struct {
	Url         string `json:"url"`
	ContentType string `json:"content_type"`
}

type HookResource struct {
	client *Client
}

func (r *HookResource) List(owner, repo string) ([]*Hook, error) {
	hooks := []*Hook{}
	path := fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)

	if err := r.client.do("GET", path, nil, &hooks); err != nil {
		return nil, err
	}

	return hooks, nil
}

func (r *HookResource) Create(owner, repo, link string) (*Hook, error) {
	// TODO: alter the method signature to take a list of Event types
	in := Hook{
		Name:   "web",
		Active: true,
		Events: []string{HookPush, HookPullRequest},
		Config: &HookConfig{
			Url:         link,
			ContentType: "application/json",
		},
	}

	out := Hook{}
	path := fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
	if err := r.client.do("POST", path, &in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (r *HookResource) Update(owner, repo string, hook *Hook) (*Hook, error) {
	out := Hook{}
	path := fmt.Sprintf("/repos/%s/%s/hooks/%v", owner, repo, hook.Id)
	if err := r.client.do("PATCH", path, hook, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (r *HookResource) CreateUpdate(owner, repo, link string) (*Hook, error) {
	// if the URL already exists, then no need to add
	if found, err := r.FindUrl(owner, repo, link); err == nil {
		return found, nil
	}

	return r.Create(owner, repo, link)
}

func (r *HookResource) Delete(owner, repo string, id int) error {
	path := fmt.Sprintf("/repos/%s/%s/hooks/%v", owner, repo, id)
	return r.client.do("DELETE", path, nil, nil)
}

func (r *HookResource) Find(owner, repo string, id int) (*Hook, error) {
	hook := Hook{}
	path := fmt.Sprintf("/repos/%s/%s/hooks/%v", owner, repo, id)

	if err := r.client.do("GET", path, nil, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

func (r *HookResource) FindUrl(owner, repo, link string) (*Hook, error) {
	hooks, err := r.List(owner, repo)
	if err != nil {
		return nil, err
	}

	for _, hook := range hooks {
		if hook.Config != nil && hook.Config.Url == link {
			return hook, nil
		}
	}

	return nil, ErrNotFound
}

func (r *HookResource) DeleteUrl(owner, repo, link string) error {
	hook, err := r.FindUrl(owner, repo, link)
	if err != nil {
		return err
	}

	return r.Delete(owner, repo, hook.Id)
}

// Github Pull Request Hook functions
// -----------------------------------------------------------------------------

type PullRequest struct {
	Id        int    `json:"int"`
	Number    int    `json:"int"`
	Url       string `json:"url"`
	State     string `json:"state"` // open, closed, etc
	Title     string `json:"title"`
	Merged    bool   `json:"merged"`
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changed   int    `json:"changed_files"`

	Base *Base `json:"base"`
	Head *Head `json:"head"`
	User *User `json:"user"`
}

type PullRequestHook struct {
	Action      string       `json:"action"`
	Number      int          `json:"number"`
	Sender      *User        `json:"sender"`
	Repo        *CommitRepo  `json:"repository"`
	PullRequest *PullRequest `json:"pull_request"`
}

type Head struct {
	Label string      `json:"label"`
	Ref   string      `json:"ref"`
	Sha   string      `json:"sha"`
	User  *User       `json:"user"`
	Repo  *CommitRepo `json:"repo"`
}

type Base struct {
	Label string      `json:"label"`
	Ref   string      `json:"ref"`
	Sha   string      `json:"sha"`
	User  *User       `json:"user"`
	Repo  *CommitRepo `json:"repo"`
}

func ParsePullRequestHook(raw []byte) (*PullRequestHook, error) {
	hook := PullRequestHook{}
	if err := json.Unmarshal(raw, &hook); err != nil {
		return nil, err
	}

	// it is possible the JSON was parsed, however,
	// was not from Github (maybe was from Bitbucket)
	// So we'll check to be sure certain key fields
	// were populated
	if hook.PullRequest == nil {
		return nil, ErrInvalidPostReceiveHook
	}

	return &hook, nil
}

func (h *PullRequestHook) IsOpened() bool {
	return h.Action == "opened"
}

// -----------------------------------------------------------------------------
// Github Post-Recieve Hook functions

var ErrInvalidPostReceiveHook = errors.New("Invalid Post Receive Hook")

type PostReceiveHook struct {
	Before  string      `json:"before"`
	After   string      `json:"after"`
	Ref     string      `json:"ref"`
	Repo    *CommitRepo `json:"repository"`
	Commits []*Commit   `json:"commits"`
	Head    *Commit     `json:"head_commit"`
	Deleted bool        `json:"deleted"`
}

type CommitRepo struct {
	Url    string `json:"url"`
	Name   string `json:"name"`
	Desc   string `json:"description"`
	Owner  *Owner `json:"owner"`
}

type Commit struct {
	Id        string   `json:"id"`
	Url       string   `json:"url"`
	Message   string   `json:"message"`
	Timestamp string   `json:timestamp`
	Author    *Author  `json:"author"`
	Added     []string `json:"added"`
}

/*
 Notes about commits..
 The id is the hash.  There's also a "head_commit" object.  It seems like
 head_commit is always the same as the last in the commit array.

*/

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func ParseHook(raw []byte) (*PostReceiveHook, error) {
	hook := PostReceiveHook{}
	if err := json.Unmarshal(raw, &hook); err != nil {
		return nil, err
	}

	// it is possible the JSON was parsed, however,
	// was not from Github (maybe was from Bitbucket)
	// So we'll check to be sure certain key fields
	// were populated
	switch {
	case hook.Repo == nil:
		return nil, ErrInvalidPostReceiveHook
	case len(hook.Ref) == 0:
		return nil, ErrInvalidPostReceiveHook
	}

	return &hook, nil
}

func (h *PostReceiveHook) IsGithubPages() bool {
	return strings.HasSuffix(h.Ref, "/gh-pages")
}

func (h *PostReceiveHook) IsTag() bool {
	return strings.HasPrefix(h.Ref, "refs/tags/")
}

func (h *PostReceiveHook) IsHead() bool {
	return strings.HasPrefix(h.Ref, "refs/heads/")
}

func (h *PostReceiveHook) Branch() string {
	return strings.Replace(h.Ref, "refs/heads/", "", -1)
}

func (h *PostReceiveHook) IsDeleted() bool {
	return h.Deleted || h.After == "0000000000000000000000000000000000000000"
}

// TODO update the list to match these:
//   207.97.227.253/32
//   50.57.128.197/32
//   108.171.174.178/32
//   50.57.231.61/32
//   204.232.175.64/27
//   192.30.252.0/22.

var ips = map[string]bool{
	"207.97.227.253":  true,
	"50.57.128.197":   true,
	"108.171.174.178": true,
	"50.57.231.61":    true,
	"204.232.175.64":  true,
}

// Check's to see if the Post-Receive Build Hook is coming
// from a valid sender (IP Address)
func IsValidSender(ip string) bool {
	return ips[ip]
}
