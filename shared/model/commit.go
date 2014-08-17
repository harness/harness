package model

import (
	"time"
)

type Commit struct {
	Id          int64  `gorm:"primary_key:yes"     json:"id"`
	RepoId      int64  `json:"-"`
	Status      string `json:"status"`
	Started     int64  `json:"started_at"`
	Finished    int64  `json:"finished_at"`
	Duration    int64  `json:"duration"`
	Sha         string `json:"sha"`
	Branch      string `json:"branch"`
	PullRequest string `json:"pull_request"`
	Author      string `json:"author"`
	Gravatar    string `json:"gravatar"`
	Timestamp   string `json:"timestamp"`
	Message     string `json:"message"`
	Config      string `json:"-"`
	Created     int64  `json:"created_at"`
	Updated     int64  `json:"updated_at"`
}

// SetAuthor sets the author's email address and calculate the Gravatar hash.
func (c *Commit) SetAuthor(email string) {
	c.Author = email
	c.Gravatar = createGravatar(email)
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
	Remote string `json:"remote"`
	Host   string `json:"host"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`

	CommitID    int64  `son:"-"`
	RepoID      int64  `json:"-"`
	Status      string `json:"status"`
	Started     int64  `json:"started_at"`
	Finished    int64  `json:"finished_at"`
	Duration    int64  `json:"duration"`
	Sha         string `json:"sha"`
	Branch      string `json:"branch"`
	PullRequest string `json:"pull_request"`
	Author      string `json:"author"`
	Gravatar    string `json:"gravatar"`
	Timestamp   string `json:"timestamp"`
	Message     string `json:"message"`
	Config      string `json:"-"`
	Created     int64  `json:"created_at"`
	Updated     int64  `json:"updated_at"`
}

// Returns the Short (--short) Commit Hash.
func (c *CommitRepo) ShaShort() string {
	if len(c.Sha) > 8 {
		return c.Sha[:8]
	} else {
		return c.Sha
	}
}

// Returns the Started Date as an ISO8601
// formatted string.
func (c *CommitRepo) FinishedString() string {
	return time.Unix(c.Finished, 0).Format("2006-01-02T15:04:05Z")
}
