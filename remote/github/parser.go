package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/drone/drone/model"
	"github.com/google/go-github/github"
)

// push parses a hook with event type `push` and returns
// the commit data.
func (c *client) push(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &pushHook{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.Deleted {
		return nil, nil, err
	}

	repo := &model.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	// Generating rather than using hook.Repo.FullName as it's
	// not always present
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = hook.Repo.HTMLURL
	repo.IsPrivate = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Branch = hook.Repo.DefaultBranch
	repo.Kind = model.RepoGit

	build := &model.Build{}
	build.Event = model.EventPush
	build.Commit = hook.Head.ID
	build.Ref = hook.Ref
	build.Link = hook.Head.URL
	build.Branch = strings.Replace(build.Ref, "refs/heads/", "", -1)
	build.Message = hook.Head.Message
	// build.Timestamp = hook.Head.Timestamp
	build.Email = hook.Head.Author.Email
	build.Avatar = hook.Sender.Avatar
	build.Author = hook.Sender.Login
	build.Remote = hook.Repo.CloneURL

	if len(build.Author) == 0 {
		build.Author = hook.Head.Author.Username
		// default gravatar?
	}

	if strings.HasPrefix(build.Ref, "refs/tags/") {
		// just kidding, this is actually a tag event
		build.Event = model.EventTag
	}

	return repo, build, nil
}

// pullRequest parses a hook with event type `pullRequest`
// and returns the commit data.
func (c *client) pullRequest(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &struct {
		Action      string              `json:"action"`
		PullRequest *github.PullRequest `json:"pull_request"`
		Repo        *github.Repository  `json:"repository"`
	}{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}

	// ignore these
	if hook.Action != "opened" && hook.Action != "synchronize" {
		return nil, nil, nil
	}
	if *hook.PullRequest.State != "open" {
		return nil, nil, nil
	}

	repo := &model.Repo{}
	repo.Owner = *hook.Repo.Owner.Login
	repo.Name = *hook.Repo.Name
	repo.FullName = *hook.Repo.FullName
	repo.Link = *hook.Repo.HTMLURL
	repo.IsPrivate = *hook.Repo.Private
	repo.Clone = *hook.Repo.CloneURL
	repo.Kind = model.RepoGit
	repo.Branch = "master"
	if hook.Repo.DefaultBranch != nil {
		repo.Branch = *hook.Repo.DefaultBranch
	}

	build := &model.Build{}
	build.Event = model.EventPull
	build.Commit = *hook.PullRequest.Head.SHA
	build.Link = *hook.PullRequest.HTMLURL
	build.Branch = *hook.PullRequest.Head.Ref
	build.Message = *hook.PullRequest.Title
	build.Author = *hook.PullRequest.User.Login
	build.Avatar = *hook.PullRequest.User.AvatarURL
	build.Remote = *hook.PullRequest.Base.Repo.CloneURL
	build.Title = *hook.PullRequest.Title

	if c.MergeRef {
		build.Ref = fmt.Sprintf("refs/pull/%d/merge", *hook.PullRequest.Number)
	} else {
		build.Ref = fmt.Sprintf("refs/pull/%d/head", *hook.PullRequest.Number)
	}

	// build.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")

	return repo, build, nil
}

func (c *client) deployment(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &deployHook{}

	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}

	// for older versions of GitHub. Remove.
	if hook.Deployment.ID == 0 {
		hook.Deployment.ID = hook.ID
		hook.Deployment.Sha = hook.Sha
		hook.Deployment.Ref = hook.Ref
		hook.Deployment.Task = hook.Name
		hook.Deployment.Env = hook.Env
		hook.Deployment.Desc = hook.Desc
	}

	repo := &model.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = hook.Repo.HTMLURL
	repo.IsPrivate = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Branch = hook.Repo.DefaultBranch
	repo.Kind = model.RepoGit

	// ref can be
	// branch, tag, or sha

	build := &model.Build{}
	build.Event = model.EventDeploy
	build.Commit = hook.Deployment.Sha
	build.Link = hook.Deployment.URL
	build.Message = hook.Deployment.Desc
	build.Avatar = hook.Sender.Avatar
	build.Author = hook.Sender.Login
	build.Ref = hook.Deployment.Ref
	build.Branch = hook.Deployment.Ref
	build.Deploy = hook.Deployment.Env

	// if the ref is a sha or short sha we need to manually
	// construct the ref.
	if strings.HasPrefix(build.Commit, build.Ref) || build.Commit == build.Ref {
		build.Branch = repo.Branch
		build.Ref = fmt.Sprintf("refs/heads/%s", repo.Branch)

	}
	// if the ref is a branch we should make sure it has refs/heads prefix
	if !strings.HasPrefix(build.Ref, "refs/") { // branch or tag
		build.Ref = fmt.Sprintf("refs/heads/%s", build.Branch)

	}

	return repo, build, nil
}
