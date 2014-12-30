package model

import (
	"time"
)

type Commit struct {
	ID          int64  `meddler:"commit_id,pk"     json:"id"`
	RepoID      int64  `meddler:"repo_id"          json:"-"`
	Status      string `meddler:"commit_status"    json:"status"`
	Started     int64  `meddler:"commit_started"   json:"started_at"`
	Finished    int64  `meddler:"commit_finished"  json:"finished_at"`
	Duration    int64  `meddler:"commit_duration"  json:"duration"`
	Sha         string `meddler:"commit_sha"       json:"sha"`
	Branch      string `meddler:"commit_branch"    json:"branch"`
	PullRequest string `meddler:"commit_pr"        json:"pull_request"`
	Author      string `meddler:"commit_author"    json:"author"`
	Gravatar    string `meddler:"commit_gravatar"  json:"gravatar"`
	Timestamp   string `meddler:"commit_timestamp" json:"timestamp"`
	Message     string `meddler:"commit_message"   json:"message"`
	Config      string `meddler:"commit_yaml"      json:"-"`
	Created     int64  `meddler:"commit_created"   json:"created_at"`
	Updated     int64  `meddler:"commit_updated"   json:"updated_at"`
}

// SetAuthor sets the author's email address and calculate the Gravatar hash.
func (c *Commit) SetAuthor(email string) {
	c.Author = email
	c.Gravatar = CreateGravatar(email)
}

// Returns the Short (--short) Commit Hash.
func (c *Commit) ShaShort() string {
	if len(c.Sha) > 8 {
		return c.Sha[:8]
	} else {
		return c.Sha
	}
}

// Returns the Started Date as an ISO8601
// formatted string.
func (c *Commit) FinishedString() string {
	return time.Unix(c.Finished, 0).Format("2006-01-02T15:04:05Z")
}

type CommitRepo struct {
	Remote string `meddler:"repo_remote" json:"remote"`
	Host   string `meddler:"repo_host"   json:"host"`
	Owner  string `meddler:"repo_owner"  json:"owner"`
	Name   string `meddler:"repo_name"   json:"name"`

	CommitID    int64  `meddler:"commit_id,pk"     json:"-"`
	RepoID      int64  `meddler:"repo_id"          json:"-"`
	Status      string `meddler:"commit_status"    json:"status"`
	Started     int64  `meddler:"commit_started"   json:"started_at"`
	Finished    int64  `meddler:"commit_finished"  json:"finished_at"`
	Duration    int64  `meddler:"commit_duration"  json:"duration"`
	Sha         string `meddler:"commit_sha"       json:"sha"`
	Branch      string `meddler:"commit_branch"    json:"branch"`
	PullRequest string `meddler:"commit_pr"        json:"pull_request"`
	Author      string `meddler:"commit_author"    json:"author"`
	Gravatar    string `meddler:"commit_gravatar"  json:"gravatar"`
	Timestamp   string `meddler:"commit_timestamp" json:"timestamp"`
	Message     string `meddler:"commit_message"   json:"message"`
	Config      string `meddler:"commit_yaml"      json:"-"`
	Created     int64  `meddler:"commit_created"   json:"created_at"`
	Updated     int64  `meddler:"commit_updated"   json:"updated_at"`
}
