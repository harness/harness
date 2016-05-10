// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gogs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Permission represents a API permission.
type Permission struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// Repository represents a API repository.
type Repository struct {
	Id          int64      `json:"id"`
	Owner       User       `json:"owner"`
	FullName    string     `json:"full_name"`
	Private     bool       `json:"private"`
	Fork        bool       `json:"fork"`
	HtmlUrl     string     `json:"html_url"`
	CloneUrl    string     `json:"clone_url"`
	SshUrl      string     `json:"ssh_url"`
	Permissions Permission `json:"permissions"`
}

// ListMyRepos lists all repositories for the authenticated user that has access to.
func (c *Client) ListMyRepos() ([]*Repository, error) {
	repos := make([]*Repository, 0, 10)
	return repos, c.getParsedResponse("GET", "/user/repos", nil, nil, &repos)
}

type CreateRepoOption struct {
	Name        string `json:"name" binding:"Required;AlphaDashDot;MaxSize(100)"`
	Description string `json:"description" binding:"MaxSize(255)"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
	Gitignores  string `json:"gitignores"`
	License     string `json:"license"`
	Readme      string `json:"readme"`
}

// CreateRepo creates a repository for authenticated user.
func (c *Client) CreateRepo(opt CreateRepoOption) (*Repository, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	repo := new(Repository)
	return repo, c.getParsedResponse("POST", "/user/repos",
		http.Header{"content-type": []string{"application/json"}}, bytes.NewReader(body), repo)
}

// CreateOrgRepo creates an organization repository for authenticated user.
func (c *Client) CreateOrgRepo(org string, opt CreateRepoOption) (*Repository, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	repo := new(Repository)
	return repo, c.getParsedResponse("POST", fmt.Sprintf("/org/%s/repos", org),
		http.Header{"content-type": []string{"application/json"}}, bytes.NewReader(body), repo)
}

// GetRepo returns information of a repository of given owner.
func (c *Client) GetRepo(owner, reponame string) (*Repository, error) {
	repo := new(Repository)
	return repo, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s", owner, reponame), nil, nil, repo)
}

// DeleteRepo deletes a repository of user or organization.
func (c *Client) DeleteRepo(owner, repo string) error {
	_, err := c.getResponse("DELETE", fmt.Sprintf("/repos/%s/%s", owner, repo), nil, nil)
	return err
}

type MigrateRepoOption struct {
	CloneAddr    string `json:"clone_addr" binding:"Required"`
	AuthUsername string `json:"auth_username"`
	AuthPassword string `json:"auth_password"`
	UID          int    `json:"uid" binding:"Required"`
	RepoName     string `json:"repo_name" binding:"Required"`
	Mirror       bool   `json:"mirror"`
	Private      bool   `json:"private"`
	Description  string `json:"description"`
}

// MigrateRepo migrates a repository from other Git hosting sources for the
// authenticated user.
//
// To migrate a repository for a organization, the authenticated user must be a
// owner of the specified organization.
func (c *Client) MigrateRepo(opt MigrateRepoOption) (*Repository, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	repo := new(Repository)
	return repo, c.getParsedResponse("POST", "/repos/migrate",
		http.Header{"content-type": []string{"application/json"}}, bytes.NewReader(body), repo)
}
