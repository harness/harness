// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gogs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type PullRequestMeta struct {
	HasMerged bool       `json:"merged"`
	Merged    *time.Time `json:"merged_at"`
}

type Issue struct {
	ID          int64            `json:"id"`
	Index       int64            `json:"number"`
	State       string           `json:"state"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	User        *User            `json:"user"`
	Labels      []*Label         `json:"labels"`
	Assignee    *User            `json:"assignee"`
	Milestone   *Milestone       `json:"milestone"`
	Comments    int              `json:"comments"`
	PullRequest *PullRequestMeta `json:"pull_request"`
	Created     time.Time        `json:"created_at"`
	Updated     time.Time        `json:"updated_at"`
}

type ListIssueOption struct {
	Page int
}

func (c *Client) ListRepoIssues(owner, repo string, opt ListIssueOption) ([]*Issue, error) {
	issues := make([]*Issue, 0, 10)
	return issues, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/issues?page=%d", owner, repo, opt.Page), nil, nil, &issues)
}

func (c *Client) GetIssue(owner, repo string, index int64) (*Issue, error) {
	issue := new(Issue)
	return issue, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index), nil, nil, issue)
}

type CreateIssueOption struct {
	Title     string  `json:"title" binding:"Required"`
	Body      string  `json:"body"`
	Assignee  string  `json:"assignee"`
	Milestone int64   `json:"milestone"`
	Labels    []int64 `json:"labels"`
	Closed    bool    `json:"closed"`
}

func (c *Client) CreateIssue(owner, repo string, opt CreateIssueOption) (*Issue, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	issue := new(Issue)
	return issue, c.getParsedResponse("POST", fmt.Sprintf("/repos/%s/%s/issues", owner, repo),
		jsonHeader, bytes.NewReader(body), issue)
}

type EditIssueOption struct {
	Title     string  `json:"title"`
	Body      *string `json:"body"`
	Assignee  *string `json:"assignee"`
	Milestone *int64  `json:"milestone"`
}

func (c *Client) EditIssue(owner, repo string, index int64, opt EditIssueOption) (*Issue, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	issue := new(Issue)
	return issue, c.getParsedResponse("PATCH", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, index),
		jsonHeader, bytes.NewReader(body), issue)
}
