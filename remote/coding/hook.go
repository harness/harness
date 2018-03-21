// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coding

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/drone/drone/model"
)

const (
	hookEvent = "X-Coding-Event"
	hookPush  = "push"
	hookPR    = "pull_request"
	hookMR    = "merge_request"
)

type User struct {
	GlobalKey string `json:"global_key"`
	Avatar    string `json:"avatar"`
}

type Repository struct {
	Name     string `json:"name"`
	HttpsURL string `json:"https_url"`
	SshURL   string `json:"ssh_url"`
	WebURL   string `json:"web_url"`
	Owner    *User  `json:"owner"`
}

type Committer struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Commit struct {
	SHA          string     `json:"sha"`
	ShortMessage string     `json:"short_message"`
	Committer    *Committer `json:"committer"`
}

type PullRequest MergeRequest

type MergeRequest struct {
	SourceBranch string  `json:"source_branch"`
	TargetBranch string  `json:"target_branch"`
	CommitSHA    string  `json:"merge_commit_sha"`
	Status       string  `json:"status"`
	Action       string  `json:"action"`
	Number       float64 `json:"number"`
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	WebURL       string  `json:"web_url"`
	User         *User   `json:"user"`
}

type PushHook struct {
	Event      string      `json:"event"`
	Repository *Repository `json:"repository"`
	Ref        string      `json:"ref"`
	Before     string      `json:"before"`
	After      string      `json:"after"`
	Commits    []*Commit   `json:"commits"`
	User       *User       `json:"user"`
}

type PullRequestHook struct {
	Event       string       `json:"event"`
	Repository  *Repository  `json:"repository"`
	PullRequest *PullRequest `json:"pull_request"`
}

type MergeRequestHook struct {
	Event        string        `json:"event"`
	Repository   *Repository   `json:"repository"`
	MergeRequest *MergeRequest `json:"merge_request"`
}

func parseHook(r *http.Request) (*model.Repo, *model.Build, error) {
	raw, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, nil, err
	}

	switch r.Header.Get(hookEvent) {
	case hookPush:
		return parsePushHook(raw)
	case hookPR:
		return parsePullRequestHook(raw)
	case hookMR:
		return parseMergeReuqestHook(raw)
	}
	return nil, nil, nil
}

func findLastCommit(commits []*Commit, sha string) *Commit {
	var lastCommit *Commit
	for _, commit := range commits {
		if commit.SHA == sha {
			lastCommit = commit
			break
		}
	}
	if lastCommit == nil {
		lastCommit = &Commit{}
	}
	if lastCommit.Committer == nil {
		lastCommit.Committer = &Committer{}
	}
	return lastCommit
}

func convertRepository(repo *Repository) (*model.Repo, error) {
	// tricky stuff for a team project without a team owner instead of a user owner
	re := regexp.MustCompile(`git@.+:([^/]+)/.+\.git`)
	matches := re.FindStringSubmatch(repo.SshURL)
	if len(matches) != 2 {
		return nil, fmt.Errorf("Unable to resolve owner from ssh url %q", repo.SshURL)
	}

	return &model.Repo{
		Owner:    matches[1],
		Name:     repo.Name,
		FullName: projectFullName(repo.Owner.GlobalKey, repo.Name),
		Link:     repo.WebURL,
		Kind:     model.RepoGit,
	}, nil
}

func parsePushHook(raw []byte) (*model.Repo, *model.Build, error) {
	hook := &PushHook{}
	err := json.Unmarshal(raw, hook)
	if err != nil {
		return nil, nil, err
	}

	// no build triggered when removing ref
	if hook.After == "0000000000000000000000000000000000000000" {
		return nil, nil, nil
	}

	repo, err := convertRepository(hook.Repository)
	if err != nil {
		return nil, nil, err
	}

	lastCommit := findLastCommit(hook.Commits, hook.After)
	build := &model.Build{
		Event:   model.EventPush,
		Commit:  hook.After,
		Ref:     hook.Ref,
		Link:    fmt.Sprintf("%s/git/commit/%s", hook.Repository.WebURL, hook.After),
		Branch:  strings.Replace(hook.Ref, "refs/heads/", "", -1),
		Message: lastCommit.ShortMessage,
		Email:   lastCommit.Committer.Email,
		Avatar:  hook.User.Avatar,
		Author:  hook.User.GlobalKey,
		Remote:  hook.Repository.HttpsURL,
	}
	return repo, build, nil
}

func parsePullRequestHook(raw []byte) (*model.Repo, *model.Build, error) {
	hook := &PullRequestHook{}
	err := json.Unmarshal(raw, hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.PullRequest.Status != "CANMERGE" ||
		(hook.PullRequest.Action != "create" && hook.PullRequest.Action != "synchronize") {
		return nil, nil, nil
	}

	repo, err := convertRepository(hook.Repository)
	if err != nil {
		return nil, nil, err
	}
	build := &model.Build{
		Event:   model.EventPull,
		Commit:  hook.PullRequest.CommitSHA,
		Link:    hook.PullRequest.WebURL,
		Ref:     fmt.Sprintf("refs/pull/%d/MERGE", int(hook.PullRequest.Number)),
		Branch:  hook.PullRequest.TargetBranch,
		Message: hook.PullRequest.Body,
		Author:  hook.PullRequest.User.GlobalKey,
		Avatar:  hook.PullRequest.User.Avatar,
		Title:   hook.PullRequest.Title,
		Remote:  hook.Repository.HttpsURL,
		Refspec: fmt.Sprintf("%s:%s", hook.PullRequest.SourceBranch, hook.PullRequest.TargetBranch),
	}

	return repo, build, nil
}

func parseMergeReuqestHook(raw []byte) (*model.Repo, *model.Build, error) {
	hook := &MergeRequestHook{}
	err := json.Unmarshal(raw, hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.MergeRequest.Status != "CANMERGE" ||
		(hook.MergeRequest.Action != "create" && hook.MergeRequest.Action != "synchronize") {
		return nil, nil, nil
	}

	repo, err := convertRepository(hook.Repository)
	if err != nil {
		return nil, nil, err
	}

	build := &model.Build{
		Event:   model.EventPull,
		Commit:  hook.MergeRequest.CommitSHA,
		Link:    hook.MergeRequest.WebURL,
		Ref:     fmt.Sprintf("refs/merge/%d/MERGE", int(hook.MergeRequest.Number)),
		Branch:  hook.MergeRequest.TargetBranch,
		Message: hook.MergeRequest.Body,
		Author:  hook.MergeRequest.User.GlobalKey,
		Avatar:  hook.MergeRequest.User.Avatar,
		Title:   hook.MergeRequest.Title,
		Remote:  hook.Repository.HttpsURL,
		Refspec: fmt.Sprintf("%s:%s", hook.MergeRequest.SourceBranch, hook.MergeRequest.TargetBranch),
	}
	return repo, build, nil
}
