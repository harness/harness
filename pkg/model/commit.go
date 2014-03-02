package model

import (
	"fmt"
	"time"
)

type Commit struct {
	ID          int64     `meddler:"id,pk"            json:"id"`
	RepoID      int64     `meddler:"repo_id"          json:"-"`
	Status      string    `meddler:"status"           json:"status"`
	Started     time.Time `meddler:"started,utctime"  json:"started"`
	Finished    time.Time `meddler:"finished,utctime" json:"finished"`
	Duration    int64     `meddler:"duration"         json:"duration"`
	Hash        string    `meddler:"hash"             json:"hash"`
	Branch      string    `meddler:"branch"           json:"branch"`
	PullRequest string    `meddler:"pull_request"     json:"pull_request"`
	Author      string    `meddler:"author"           json:"author"`
	Gravatar    string    `meddler:"gravatar"         json:"gravatar"`
	Timestamp   string    `meddler:"timestamp"        json:"timestamp"`
	Message     string    `meddler:"message"          json:"message"`

	Created time.Time `meddler:"created,utctime"  json:"created"`
	Updated time.Time `meddler:"updated,utctime"  json:"updated"`
}

// Returns the Short (--short) Commit Hash.
func (c *Commit) HashShort() string {
	if len(c.Hash) > 6 {
		return c.Hash[:6]
	} else {
		return c.Hash
	}
}

// Returns the Gravatar Image URL.
func (c *Commit) Image() string      { return fmt.Sprintf(GravatarPattern, c.Gravatar, 58) }
func (c *Commit) ImageSmall() string { return fmt.Sprintf(GravatarPattern, c.Gravatar, 32) }
func (c *Commit) ImageLarge() string { return fmt.Sprintf(GravatarPattern, c.Gravatar, 160) }

// Returns the Started Date as an ISO8601
// formatted string.
func (c *Commit) StartedString() string {
	return c.Started.Format("2006-01-02T15:04:05Z")
}

// Returns the Created Date as an ISO8601
// formatted string.
func (c *Commit) CreatedString() string {
	return c.Created.Format("2006-01-02T15:04:05Z")
}

// Returns the Started Date as an ISO8601
// formatted string.
func (c *Commit) FinishedString() string {
	return c.Finished.Format("2006-01-02T15:04:05Z")
}

// Set the Author's email address and calculate the
// Gravatar hash.
func (c *Commit) SetAuthor(email string) {
	c.Author = email
	c.Gravatar = createGravatar(email)
}

// Combined Repository and Commit details
type RepoCommit struct {
	// Repo Details
	Slug  string `meddler:"slug"  json:"slug"`
	Host  string `meddler:"host"  json:"host"`
	Owner string `meddler:"owner" json:"owner"`
	Name  string `meddler:"name"  json:"name"`

	// Commit Details
	Status      string    `meddler:"status"           json:"status"`
	Started     time.Time `meddler:"started,utctime"  json:"started"`
	Finished    time.Time `meddler:"finished,utctime" json:"finished"`
	Duration    int64     `meddler:"duration"         json:"duration"`
	Hash        string    `meddler:"hash"             json:"hash"`
	Branch      string    `meddler:"branch"           json:"branch"`
	PullRequest string    `meddler:"pull_request"     json:"pull_request"`
	Author      string    `meddler:"author"           json:"author"`
	Gravatar    string    `meddler:"gravatar"         json:"gravatar"`
	Timestamp   string    `meddler:"timestamp"        json:"timestamp"`
	Message     string    `meddler:"message"          json:"message"`
	Created     time.Time `meddler:"created,utctime"  json:"created"`
	Updated     time.Time `meddler:"updated,utctime"  json:"updated"`
}

// Returns the Short (--short) Commit Hash.
func (c *RepoCommit) HashShort() string {
	if len(c.Hash) > 6 {
		return c.Hash[:6]
	} else {
		return c.Hash
	}
}

// Returns the Gravatar Image URL.
func (c *RepoCommit) Image() string      { return fmt.Sprintf(GravatarPattern, c.Gravatar, 42) }
func (c *RepoCommit) ImageSmall() string { return fmt.Sprintf(GravatarPattern, c.Gravatar, 32) }
func (c *RepoCommit) ImageLarge() string { return fmt.Sprintf(GravatarPattern, c.Gravatar, 160) }

// Returns the Started Date as an ISO8601
// formatted string.
func (c *RepoCommit) StartedString() string {
	return c.Started.Format("2006-01-02T15:04:05Z")
}

// Returns the Created Date as an ISO8601
// formatted string.
func (c *RepoCommit) CreatedString() string {
	return c.Created.Format("2006-01-02T15:04:05Z")
}

// Returns the Started Date as an ISO8601
// formatted string.
func (c *RepoCommit) FinishedString() string {
	return c.Finished.Format("2006-01-02T15:04:05Z")
}
